package server

import (
	"database/sql"
	"os"
	"strconv"
	"strings"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/mattn/go-sqlite3"
)

const (
	sqliteFK = "sqlite_fk"
)

func init() {
	hook := func(conn *sqlite3.SQLiteConn) error {
		_, err := conn.Exec("PRAGMA foreign_keys = ON", nil)
		return err
	}
	driver := &sqlite3.SQLiteDriver{ConnectHook: hook}
	sql.Register(sqliteFK, driver)
}

type Data struct {
	err  error
	log  log15.Logger
	base string
	path string
	db   *sql.DB
}

func (d *Data) Setup(base, path string, log log15.Logger) error {
	d.log = log
	d.base = base
	d.err = os.MkdirAll(base, 0755)
	if d.err != nil {
		return d.err
	}
	d.path = path
	d.db, d.err = sql.Open(sqliteFK, path)
	d.initFiles()
	d.initJournal()
	d.initJData()
	d.initImages()
	return d.err
}

func (d *Data) Database() *sql.DB {
	return d.db
}

func (d *Data) initFiles() {
	if d.err != nil {
		return
	}
	_, d.err = d.db.Exec(tblFiles)
}

func (d *Data) initJournal() {
	if d.err != nil {
		return
	}
	_, d.err = d.db.Exec(tblJournal)
}

func (d *Data) initJData() {
	if d.err != nil {
		return
	}
	_, d.err = d.db.Exec(tblJData)
}

func (d *Data) initImages() {
	if d.err != nil {
		return
	}
	_, d.err = d.db.Exec(tblImages)
}

func (d *Data) insertFile(ftype, path, name string, r *Rules) error {
	_, err := d.db.Exec("INSERT INTO files(type, path, name, own, grp, mod)", ftype, path, name, r.Owner, r.Group, r.Mode)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "UNIQUE") {
			return err
		}
	}
	return nil
}

func (d *Data) postMessagesLog(log *Log) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r, err := tx.Exec("INSERT INTO files(type, name, path, own, grp, mod) VALUES(?,?,?,?,?,?)", "log", "", "", log.Rules.Owner, log.Rules.Group, log.Rules.Mode)
	if err != nil {
		return err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return err
	}
	r, err = tx.Exec("INSERT INTO journal(id, time, sev, msg, code) VALUES(?,?,?,?,?)", id, log.Time, log.Severity, log.Message, log.Code)
	if err != nil {
		return err
	}
	id, err = r.LastInsertId()
	if err != nil {
		return err
	}
	if log.Args != nil {
		for key, value := range log.Args {
			_, err = tx.Exec(
				"INSERT INTO jdata(id, key, val) "+
					"VALUES(?,?,?)",
				id, key, value)
			if err != nil {
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (d *Data) getMessages(s *Session, severity Severity, sort string, start, end int64, offset, length int) (int, []message) {
	// Query
	// build strings for SQL request
	sevString := sqlSevString(severity)
	groupsString := sqlGroupsString(s.User.Groups())
	var rows *sql.Rows
	var err error
	var count int
	if s.SU {
		rows, err = d.db.Query("SELECT * FROM journal JOIN files ON journal.id = files.id WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+") ORDER BY "+sort+" LIMIT ?,?", start, end, offset, length)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		row := d.db.QueryRow("SELECT COUNT(*) FROM journal JOIN files ON journal.id = files.id WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+")", start, end)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		err = row.Scan(&count)
		if err != nil {
			panic(err)
		}
	} else {
		rows, err = d.db.Query("SELECT * FROM journal JOIN files ON journal.id = files.id WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+") AND ((own = ? AND (mod & 0x100) = 0x100) OR (own != ? AND grp IN "+groupsString+" AND (mod & 0x020) = 0x020) OR (own != ? AND grp NOT IN "+groupsString+" AND (mod & 0x004) = 0x004)) ORDER BY "+sort+" LIMIT ?,?", start, end, s.User.Name(), s.User.Name(), s.User.Name(), offset, length)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		row := d.db.QueryRow("SELECT COUNT(*) FROM journal JOIN files ON journal.id = files.id WHERE (time BETWEEN ? AND ?) AND (sev IN "+sevString+") AND ((own = ? AND (mod & 0x100) = 0x100) OR (own != ? AND grp IN "+groupsString+" AND (mod & 0x020) = 0x020) OR (own != ? AND grp NOT IN "+groupsString+" AND (mod & 0x004) = 0x004))", start, end, s.User.Name(), s.User.Name(), s.User.Name())
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		err = row.Scan(&count)
		if err != nil {
			panic(err)
		}
	}

	var out []message
	for rows.Next() {
		var id, time, severity, z int
		var msg, code, y string
		err = rows.Scan(&id, &time, &severity, &msg, &code, &z, &y, &y, &y, &y, &y, &z)
		if err != nil {
			panic(err)
		}

		ao := message{
			Time:     int64(time),
			Message:  msg,
			Code:     code,
			Args:     make(map[string]string),
			Severity: SeverityStrings[Severity(severity)],
		}

		argrows, err := d.db.Query("SELECT * FROM jdata WHERE id=?", id)
		if err != nil {
			panic(err)
		}
		defer argrows.Close()

		for argrows.Next() {
			var rid int
			var key, value string
			err = argrows.Scan(&rid, &key, &value)
			if err != nil {
				panic(err)
			}
			ao.Args[key] = value
		}

		out = append(out, ao)
	}
	return count, out
}

func sqlSevString(severity Severity) string {
	sevString := "("
	switch severity {
	case Alert:
		sevString += strconv.Itoa(Warn)
		sevString += ", "
		sevString += strconv.Itoa(Error)
		sevString += ", "
		sevString += strconv.Itoa(Crit)
	case All:
		sevString += strconv.Itoa(Debug)
		sevString += ", "
		sevString += strconv.Itoa(Info)
		sevString += ", "
		sevString += strconv.Itoa(Warn)
		sevString += ", "
		sevString += strconv.Itoa(Error)
		sevString += ", "
		sevString += strconv.Itoa(Crit)
	default:
		sevString += strconv.Itoa(int(severity))
	}
	sevString += ")"
	return sevString
}

func sqlGroupsString(groups []string) string {
	groupsString := "("
	for i, group := range groups {
		groupsString += "'" + group + "'"
		if i < len(groups)-1 {
			groupsString += ", "
		}
	}
	groupsString += ")"
	return groupsString
}

func (d *Data) getRules(path, name string) (*Rules, error) {
	row := d.db.QueryRow("SELECT own, grp, mod FROM files WHERE path=? AND name=?", path, name)
	r := new(Rules)
	err := row.Scan(&r.Owner, &r.Group, &r.Mode)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *Data) imagesGetAttributes(path, name string) (string, string, uint64, string, error) {
	row := d.db.QueryRow("SELECT auth, desc, time, chk FROM images JOIN files ON images.id = files.id WHERE path=? AND name=?", path, name)
	auth := ""
	desc := ""
	date := uint64(0)
	chk := ""
	err := row.Scan(&auth, &desc, &date, &chk)
	if err != nil {
		panic(err)
	}
	return auth, desc, date, chk, nil
}

func (d *Data) imagesGetInfo(path, name string) (string, error) {
	chkType := ""
	row := d.db.QueryRow("SELECT type FROM files where path=? AND name=?", path, name)
	err := row.Scan(&chkType)
	if err != nil {
		panic(err)
	}
	if chkType == "folder" {
		return chkType
	}
	row = d.db.QueryRow("SELECT chk FROM images JOIN files ON images.id = files.id WHERE path=? AND name=?", path, name)
	err = row.Scan(&chkType)
	if err != nil {
		panic(err)
	}
	return chkType
}

func (d *Data) imagesGetFolder(path, name, filter, sort, order string, offset, length int) *shortPL {
	ordstr := "type DESC, name DESC"
	if sort != "" {
		ordstr = "sort "
		if order == "ASC" || order == "DESC" {
			ordstr = append(ordstr, order)
		} else {
			ordstr = append(ordstr, "DESC")
		}
	}
	rows, err = d.db.Query("SELECT name, type FROM files WHERE path=? AND name LIKE '%?%' ORDER BY "+ordstr+" LIMIT ?,?", path+"/"+name, filter, offset, length)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	pl := new(shortPL)
	for rows.Next() {
		n, t := ""
		err = rows.Scan(&n, &t)
		if err != nil {
			panic(err)
		}
		pl.List = append(pl.List, folderPL{n, t})
	}

	count := 0
	row := d.db.QueryRow("SELECT COUNT(*) FROM files WHERE path=? AND name LIKE '%?%' ORDER BY "+ordstr+" LIMIT ?,?", path+"/"+name, filter, offset, length)
	if err != nil {
		panic(err)
	}
	err = row.Scan(&count)
	if err != nil {
		panic(err)
	}
	pl.Length = count

	return pl
}

func (d *Data) hasChildren(path, name string) bool {
	rows, err := d.db.Query("SELECT * FROM files WHERE path=?", path+"/"+name)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		return true
	}

	return false
}

func (d *Data) imagesDelete(path, name string) (bool, string) {
	row := d.db.QueryRow("SELECT chk FROM images JOIN files ON images.id = files.id WHERE path=? AND name=?", path, name)
	chk := ""
	err := row.Scan(&chk)
	if err != nil {
		return false, ""
	}

	_, err := d.db.Exec("DELETE FROM files WHERE path=? AND name=?", path, name)
	if err != nil {
		return false, ""
	}

	k := 0
	row := d.db.QueryRow("SELECT COUNT(*) FROM images WHERE chk=?", chk)
	err := row.Scan(&k)
	if err != nil {
		return false, ""
	}

	if k == 0 {
		return true, chk
	} else {
		return true, ""
	}

}

func (d *Data) imagesAdd(args uploadData) (bool, string) {
	path, name := splitPath(args.Target)
	row := d.db.QueryRow("SELECT COUNT (*) FROM images JOIN files ON images.id = files.id WHERE path=? AND name=?", path, name)
	count := 0
	err := row.Scan(&count)
	if err != nil {
		return false, ""
	}
	if count != 0 {
		return false, ""
	}

	ftype := "file"
	if args.Checksum == emptyChecksum {
		ftype = "folder"
	}

	res, err = d.db.Exec("INSERT INTO files(type, name, path, own, grp, mod)", ftype, name, path, args.Owner, args.Group, args.Mode)
	if err != nil {
		return false, ""
	}

	id, err := res.LastInsertId()
	if err != nil {
		return false, ""
	}

	_, err = d.db.Exec("INSERT INTO images(id, auth, desc, time, chk) VALUES(?,?,?,?,?)", id, args.Author, args.Description, args.Time, args.Checksum)
	if err != nil {
		return false, ""
	}

	return true, ""
}

func (d *Data) imagesOverwrite(args uploadData) (bool, string) {
	path, name := splitPath(args.Target)
	row := d.db.QueryRow("SELECT auth, desc, time, chk FROM images JOIN files ON images.id = files.id WHERE path=? AND name=?", path, name)
	var auth, desc, chk string
	var time uint64
	err := row.Scan(&auth, &desc, &time, &chk)
	if err != nil {
		return false, ""
	}

	if !args.AuthSet {
		args.Author = auth
	}
	if !args.DescSet {
		args.Description = desc
	}
	if !args.TimeSet {
		args.Time = time
	}

	_, err = d.db.Exec("UPDATE images SET auth=?, desc=?, time=?, chk=? WHERE path=? AND name=?", args.Author, args.Description, args.Time, args.Checksum, path, name)
	if err != nil {
		return false, ""
	}

	row = d.db.QueryRow("SELECT COUNT(*) FROM images WHERE chk=?", chk)
	count := 0
	err = row.Scan(&count)
	if err != nil {
		return false, ""
	}

	if count == 0 {
		return true, chk
	}
	return true, ""
}

func (d *Data) imagesAttr(args uploadData) bool {
	path, name := splitPath(args.Target)
	row := d.db.QueryRow("SELECT auth, desc, time, chk FROM images JOIN files ON images.id = files.id WHERE path=? AND name=?", path, name)
	var auth, desc, chk string
	var time uint64
	err := row.Scan(&auth, &desc, &time, &chk)
	if err != nil {
		return false
	}

	if !args.AuthSet {
		args.Author = auth
	}
	if !args.DescSet {
		args.Description = desc
	}
	if !args.TimeSet {
		args.Time = time
	}

	_, err = d.db.Exec("UPDATE images SET auth=?, desc=?, time=? WHERE path=? AND name=?", args.Author, args.Description, args.Time, path, name)
	if err != nil {
		return false
	}

	return true
}
