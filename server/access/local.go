package access

import (
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"gopkg.in/inconshreveable/log15.v2"
)

const (
	root = "root"

	sqliteFK = "sqlite_fk"

	tblUsers = `CREATE TABLE IF NOT EXISTS users(
		name VARCHAR(32) NOT NULL,
		pgrp VARCHAR(32) NOT NULL,
		salt VARCHAR(128) NOT NULL,
		hash VARCHAR(128) NOT NULL,
		PRIMARY KEY (name),
		FOREIGN KEY(pgrp) REFERENCES groups(name) ON DELETE RESTRICT
		)`

	tblGroups = `CREATE TABLE IF NOT EXISTS groups(
		name VARCHAR(32) NOT NULL,
		PRIMARY KEY (name)
		)`

	tblMemberships = `CREATE TABLE IF NOT EXISTS memberships(
		usr VARCHAR(32) NOT NULL,
		grp VARCHAR(32) NOT NULL,
		PRIMARY KEY (usr,grp),
		FOREIGN KEY(usr) REFERENCES users(name) ON DELETE CASCADE
		FOREIGN KEY(grp) REFERENCES groups(name) ON DELETE CASCADE
		)`
)

var (
	ErrCredentials = errors.New("bad username or password")
	ErrDatabase    = errors.New("a database error occurred")
)

type record struct {
	name string
	pgrp string
	salt string
	hash string
}

func recordFromRow(row *sql.Row) (*record, error) {
	rec := new(record)
	err := row.Scan(&rec.name, &rec.pgrp, &rec.salt, &rec.hash)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

type Local struct {
	err error
	log log15.Logger
	db  *sql.DB
}

func (l *Local) setup(log log15.Logger, db *sql.DB) error {
	l.log = log
	l.db = db
	l.initGroups()
	l.initUsers()
	l.initMemberships()
	l.initRoot()
	return l.err
}

func hash(salt, password string) string {
	rawSalt, _ := hex.DecodeString(salt)
	hasher := sha512.New()
	hasher.Write(rawSalt)
	hasher.Write([]byte(password))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (l *Local) initGroups() {
	if l.err != nil {
		return
	}
	_, l.err = l.db.Exec(tblGroups)
}

func (l *Local) initUsers() {
	if l.err != nil {
		return
	}
	_, l.err = l.db.Exec(tblUsers)
}

func (l *Local) initMemberships() {
	if l.err != nil {
		return
	}
	_, l.err = l.db.Exec(tblMemberships)
}

func (l *Local) initRoot() {
	if l.err != nil {
		return
	}
	_, err := l.db.Exec("INSERT INTO groups(name) VALUES(?)", root)
	if err != nil && !strings.HasPrefix(err.Error(), "UNIQUE") {
		l.err = err
		return
	}
	salt := "0000000000000000000000000000000000000000000000000000000000000000"
	hash := hash(salt, root)
	l.log.Debug("stored: " + hash)
	_, err = l.db.Exec("INSERT INTO users(name, salt, hash, pgrp) VALUES(?, ?, ?, ?)", root, salt, hash, root)
	if err != nil && !strings.HasPrefix(err.Error(), "UNIQUE") {
		l.err = err
		return
	}
	_, err = l.db.Exec("INSERT INTO memberships(usr, grp) VALUES(?, ?)", root, root)
	if err != nil && !strings.HasPrefix(err.Error(), "UNIQUE") {
		l.err = err
		return
	}
}

func (l *Local) Login(username, password string) (User, error) {
	l.log.Debug("Local handling login request")
	rec, err := l.lookupUser(username)
	if err != nil {
		l.log.Debug("user doesn't exist in database")
		return nil, ErrCredentials
	}
	hashword := hash(rec.salt, password)
	if hashword != rec.hash {
		l.log.Debug("invalid password")
		l.log.Debug("received: " + password)
		l.log.Debug("hashed: " + hashword)
		l.log.Debug("stored: " + rec.pgrp)
		return nil, ErrCredentials
	}
	return l.makeUser(rec, hashword)
}

func (l *Local) makeUser(rec *record, hash string) (User, error) {
	u := &LocalUser{
		name:    rec.name,
		primary: rec.pgrp,
	}
	var err error
	u.groups, err = l.userListGroups(u.name)
	if err != nil {
		l.log.Debug("error retrieving groups")
		return nil, err
	}
	return u, nil
}

func (l *Local) lookupUser(username string) (*record, error) {
	row := l.db.QueryRow("SELECT * FROM users WHERE name=?", username)
	rec, err := recordFromRow(row)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (l *Local) userListGroups(username string) ([]string, error) {
	var groups []string
	rows, err := l.db.Query("SELECT grp FROM memberships WHERE usr=?", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		rows.Scan(&name)
		groups = append(groups, name)
	}
	return groups, nil
}

type LocalUser struct {
	name    string
	primary string
	groups  []string
}

func (u *LocalUser) Name() string {
	return u.name
}

func (u *LocalUser) Groups() []string {
	return u.groups
}

func (u *LocalUser) PrimaryGroup() string {
	return u.primary
}
