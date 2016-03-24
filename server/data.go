package server

import (
	"database/sql"
	"os"
	"strconv"

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
