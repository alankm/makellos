package core

import (
	"database/sql"
	"os"
	"reflect"

	"github.com/alankm/makellos/core/shared"
	"github.com/mattn/go-sqlite3"

	"gopkg.in/inconshreveable/log15.v2"
)

type Core struct {
	err     error
	log     log15.Logger
	conf    config
	db      *sql.DB
	logging *shared.LoggingFunctions
	access  *shared.AuthorizationFunctions
}

func init() {
	hook := func(conn *sqlite3.SQLiteConn) error {
		_, err := conn.Exec("PRAGMA foreign_keys = ON", nil)
		return err
	}
	driver := &sqlite3.SQLiteDriver{ConnectHook: hook}
	sql.Register("sql_fk", driver)
}

func New(config string) *Core {
	c := new(Core)
	c.err = c.conf.load(config)
	c.init()
	return nil
}

func (c *Core) init() {
	if c.err == nil {
		c.err = os.MkdirAll(c.conf.Base, 0775)
		if c.err != nil {
			return
		}
		c.db, c.err = sql.Open("sql_fk", c.conf.Base+"/vorteil.db")
		c.log = log15.New("module", reflect.TypeOf(c).Name())
	}
}

func (c *Core) Start() error {
	if c.err == nil {
		c.log.Info("Starting...")
	}
	return c.err
}

func (c *Core) Stop() {

}
