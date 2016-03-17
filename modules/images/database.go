package images

import (
	"database/sql"
	"errors"
	"strings"
)

type record struct {
	name   string
	parent string
	dir    bool
	chk    string
	date   uint64
	auth   string
	desc   string
}

func recordFromRow(row *sql.Row) (*record, error) {
	rec := new(record)
	err := row.Scan(&rec.name, &rec.parent, &rec.dir, &rec.chk, &rec.date, &rec.auth, &rec.desc)
	return rec, err
}

func recordFromRows(rows *sql.Rows) *record {
	rec := new(record)
	rows.Scan(&rec.name, &rec.parent, &rec.dir, &rec.chk, &rec.date, &rec.auth, &rec.desc)
	return rec
}

func splitPath(path string) (string, string, error) {
	tmp := strings.TrimSuffix(path, "/")
	k := strings.LastIndex(tmp, "/")
	if k == -1 {
		return "", "", errors.New("bad path")
	}
	return tmp[:k], tmp[k+1:], nil
}

func (i *Images) lookup(path string) *record {
	parent, child, err := splitPath(path)
	if err != nil {
		return nil
	}
	row := i.db.QueryRow("SELECT * FROM images WHERE name=? AND parent=?", child, parent)
	rec, err := recordFromRow(row)
	if err != nil {
		return nil
	}
	return rec
}

func (i *Images) lookupChildren(path string, length, offset int) (*sql.Rows, int) {
	row := i.db.QueryRow("SELECT COUNT(*) FROM images WHERE parent=?", path)
	var count int
	err := row.Scan(&count)
	if err != nil {
		panic(err)
	}
	rows, err := i.db.Query("SELECT * FROM images WHERE parent=? ORDER BY directory DESC, name ASC LIMIT ?,?", path, offset, length)
	if err != nil {
		panic(err)
	}
	return rows, count
}
