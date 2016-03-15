package privileges

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/alankm/makellos/core/shared"
)

const (
	root         = "root"
	rpwd         = "guest"
	defaultUmask = 0002
)

var (
	ErrCredentials         = errors.New("invalid username or password")
	ErrDatabase            = errors.New("database error")
	ErrDenied              = errors.New("insufficient permissions")
	ErrGID                 = errors.New("can't delete group whilst users have it as gid")
	ErrGroupExists         = errors.New("a group with that name already exists")
	ErrGroupName           = errors.New("the provided group name is not valid")
	ErrGroupNotFound       = errors.New("no group with that name exists")
	ErrHash                = errors.New("the provided hash is not valid")
	ErrRoot                = errors.New("cannot delete or separate root user and root group")
	ErrRules               = errors.New("the provided rules string is not valid octal permissions")
	ErrSalt                = errors.New("the provided salt is not valid")
	ErrSelf                = errors.New("the action cannot be performed on self")
	ErrSession             = errors.New("the session has expired")
	ErrSuperUser           = errors.New("root privileges are required to perform action")
	ErrUserExists          = errors.New("a user with that name already exists")
	ErrUsername            = errors.New("the provided username is not valid")
	ErrUserNotFound        = errors.New("no user with that name exists")
	ErrFileExists          = errors.New("file already exists")
	ErrChildOfNonDirectory = errors.New("can't create file as child of non-directory")
)

type Privileges struct {
	functions shared.AuthorizationFunctions
	db        *sql.DB
}

func (p *Privileges) Setup(core shared.Core) error {
	p.db = core.Database()
	p.setupTables()
	p.setupBasicRecords()
	p.functions.Login = p.Login
	p.functions.HashedLogin = p.LoginHash
	core.AuthorizationHook(&p.functions)
	return nil
}

func hash(salt, password string) string {
	rawSalt, _ := hex.DecodeString(salt)
	hasher := sha512.New()
	hasher.Write(rawSalt)
	hasher.Write([]byte(password))
	return hex.EncodeToString(hasher.Sum(nil))
}

func generateSalt64() []byte {
	salt := make([]byte, 64)
	rand.Read(salt)
	return salt
}

func (p *Privileges) setupTables() {
	_, err := p.db.Exec("CREATE TABLE IF NOT EXISTS groups (" +
		"name VARCHAR(64) PRIMARY KEY" +
		");")
	if err != nil {
		panic(err)
	}
	_, err = p.db.Exec("CREATE TABLE IF NOT EXISTS users (" +
		"name VARCHAR(64) PRIMARY KEY, " +
		"salt VARCHAR(128) NULL, " +
		"pass VARCHAR(128) NULL, " +
		"gid VARCHAR(64) NULL, " +
		"FOREIGN KEY (gid) REFERENCES groups(name)" +
		");")
	if err != nil {
		panic(err)
	}
	_, err = p.db.Exec("CREATE TABLE IF NOT EXISTS usersgroups (" +
		"username VARCHAR(64) NULL, " +
		"groupname VARCHAR(64) NULL, " +
		"PRIMARY KEY (username, groupname), " +
		"FOREIGN KEY (username) REFERENCES users(name) ON DELETE CASCADE, " +
		"FOREIGN KEY (groupname) REFERENCES groups(name) ON DELETE CASCADE" +
		");")
	if err != nil {
		panic(err)
	}
	_, err = p.db.Exec("CREATE TABLE IF NOT EXISTS files (" +
		"isdir BOOL NULL, " +
		"name VARCHAR(64) NULL, " +
		"parent VARCHAR(64) NULL, " +
		"owner_user VARCHAR(64) NULL, " +
		"owner_group VARCHAR(64) NULL, " +
		"rules VARCHAR(4) NULL, " +
		"PRIMARY KEY (name, parent) " +
		");")
	if err != nil {
		panic(err)
	}
}

func (p *Privileges) setupBasicRecords() {
	_, err := p.db.Exec("INSERT INTO groups(name) VALUES(?)", root)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "UNIQUE") {
			panic(err)
		}
	}
	salt := "0000000000000000000000000000000000000000000000000000000000000000"
	hash := hash(salt, rpwd)
	_, err = p.db.Exec("INSERT INTO users(name, salt, pass, gid) VALUES(?, ?, ?, ?)", root, salt, hash, root)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "UNIQUE") {
			panic(err)
		}
	}
	_, err = p.db.Exec("INSERT INTO usersgroups(username, groupname) VALUES(?, ?)", root, root)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "UNIQUE") {
			panic(err)
		}
	}
}

func (p *Privileges) Login(username, password string) (shared.Session, error) {
	rec, err := p.lookupUser(username)
	if err != nil {
		return nil, ErrCredentials
	}
	hashword := hash(rec.salt, password)
	if hashword != rec.pass {
		return nil, ErrCredentials
	}
	return p.makeSession(rec, hashword), nil
}

func (p *Privileges) LoginHash(username, hashword string) (shared.Session, error) {
	rec, err := p.lookupUser(username)
	if err != nil {
		return nil, ErrCredentials
	}
	if err != nil || hashword != rec.pass {
		return nil, ErrCredentials
	}
	return p.makeSession(rec, hashword), nil
}

type record struct {
	name string
	salt string
	pass string
	gid  string
}

func recordFromRow(row *sql.Row) (*record, error) {
	rec := new(record)
	err := row.Scan(&rec.name, &rec.salt, &rec.pass, &rec.gid)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (p *Privileges) lookupUser(username string) (*record, error) {
	row := p.db.QueryRow("SELECT * FROM users WHERE name=?", username)
	rec, err := recordFromRow(row)
	if err != nil {
		if err != sql.ErrNoRows {
			panic(err)
		}
		return nil, ErrUserNotFound
	}
	return rec, nil
}

type Session struct {
	p      *Privileges
	sid    string
	user   string
	hash   string
	su     bool
	gid    string
	groups []string
	umask  uint16
}

func (p *Privileges) makeSession(rec *record, hash string) *Session {
	s := &Session{
		p:     p,
		user:  rec.name,
		hash:  hash,
		umask: defaultUmask,
		gid:   rec.gid,
		sid:   string(generateSalt64()),
	}
	s.groups = s.userListGroups(s.user)
	s.su = s.inGroup(s.user, root)
	return s
}

func (s *Session) inGroup(username, group string) bool {
	var x string
	row := s.p.db.QueryRow("SELECT * FROM usersgroups WHERE username=? AND groupname=?", username, group)
	err := row.Scan(&x, &x)
	if err == nil {
		return true
	} else if err == sql.ErrNoRows {
		return false
	}
	panic(err)
}

func (s *Session) userListGroups(username string) []string {
	var groups []string
	rows, err := s.p.db.Query("SELECT groupname FROM usersgroups WHERE username=?", username)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		rows.Scan(&name)
		groups = append(groups, name)
	}
	return groups
}
