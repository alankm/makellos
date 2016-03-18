package core

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"reflect"

	"github.com/alankm/makellos/core/shared"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/mattn/go-sqlite3"
	"github.com/sisatech/multiserver"
	"github.com/sisatech/raft"

	"gopkg.in/inconshreveable/log15.v2"
)

type Core struct {
	err     error
	log     log15.Logger
	conf    config
	db      *sql.DB
	logging *shared.LoggingFunctions
	access  *shared.AuthorizationFunctions
	web     web
	raft    consensus
}

type web struct {
	server   *multiserver.HTTPServer
	mux      *mux.Router
	cookie   *securecookie.SecureCookie
	services []string
}

type consensus struct {
	config *raft.Config
	server *raft.Raft
	client *raft.Client
	fsm    *stateMachine
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
	return c
}

func (c *Core) init() {
	if c.err == nil {
		c.err = os.MkdirAll(c.conf.Base, 0775)
		if c.err != nil {
			return
		}
		c.db, c.err = sql.Open("sql_fk", c.conf.Base+"/vorteil.db")
		if c.err != nil {
			return
		}
		c.log = log15.New("module", reflect.TypeOf(c).Name())
		c.web.mux = mux.NewRouter()
		c.web.server = multiserver.NewHTTPServer(c.conf.Bind, c.web.mux, nil)
		c.web.cookie = securecookie.New(securecookie.GenerateRandomKey(64),
			securecookie.GenerateRandomKey(32))
		c.web.mux.HandleFunc("/services/login", c.loginHandler)
		c.raft.config = &c.conf.Raft
		if c.raft.config.BaseDir == "" {
			c.raft.config.BaseDir = c.conf.Base + "/raft"
		}
		c.raft.config.FillDefaults()
		c.raft.fsm = &stateMachine{
			c:          c,
			components: make(map[string]func([]byte) []byte),
		}
		c.raft.config.StateMachine = c.raft.fsm
		c.raft.client = raft.NewClient(nil)
		c.raft.server, c.err = raft.NewRaft(c.raft.config)
		if c.err != nil {
			return
		}
		properties := make(map[string]string)
		properties["HTTP"] = c.conf.Advertise
		c.raft.server.PropertiesSet(properties)
	}
}

func (c *Core) loginHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	val := make(map[string]string)
	err := dec.Decode(&val)
	if err != nil {
		w.Write(shared.ResponseBadLoginBody.JSON())
		return
	}

	username := val["username"]
	password := val["password"]

	if username != "root" || password != "guest" {
		w.Write(shared.ResponseBadLogin.JSON())
		return
	}

	_, err = c.Login(username, password)
	if err != nil {
		w.Write(shared.ResponseBadLogin.JSON())
		return
	}

	enc, err := c.web.cookie.Encode("vorteil", val)
	if err != nil {
		panic(err)
	}

	cookie := &http.Cookie{
		Name:  "vorteil",
		Value: enc,
		Path:  "/",
	}
	http.SetCookie(w, cookie)

	var services []string
	services = make([]string, 0)
	/*
		for _, val := range c.web.services {
			if session.Access(val) == nil {
				services = append(services, val)
			}
		}
	*/
	//l.vorteil.Alert(nil, Debug, "blah", "\""+session.User()+"\" logged in", l.Name(), nil)

	w.Write(shared.NewSuccessResponse(services).JSON())
}

func (c *Core) Start() error {
	if c.err == nil {
		c.log.Info("Starting...")
		return c.web.server.Start()
	}
	return c.err
}

func (c *Core) Stop() {

}

func (c *Core) IsLeader() bool {
	return true
}

func (c *Core) Leader() string {
	return c.conf.Advertise
}

func (c *Core) ServiceRouter(service string) *mux.Router {
	return c.web.mux.PathPrefix("/services/" + service).Subrouter()
}

func (c *Core) ValidateCookie(cookie *http.Cookie) shared.Session {
	value := make(map[string]string)
	err := c.web.cookie.Decode("vorteil", cookie.Value, &value)
	if err == nil {
		username := value["username"]
		password := value["password"]
		session, err := c.Login(username, password)
		if err == nil {
			return session
		}
	}
	return nil
}
