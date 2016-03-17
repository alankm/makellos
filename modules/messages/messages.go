package messages

import (
	"database/sql"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/alankm/makellos/core/shared"
	"github.com/gorilla/mux"
)

/*
The 'Messages' object implements Vorteil's 'Module' interface. It exists to add
a powerful and advanced logging & messaging system for use by other modules to
Vorteil. It is also a Vorteil service, providing web-accessible interfaces to
access and interact with its data.
*/
type Messages struct {
	functions shared.LoggingFunctions
	db        *sql.DB
	log       log15.Logger
	core      shared.Core
	inbox     chan *shared.Log
	outbox    map[shared.Severity](chan *shared.Log)
	clients   map[shared.Severity](map[chan *shared.Log]bool)
}

type listPayload struct {
	Length int       `json:"length"`
	List   []message `json:"list"`
}

type message struct {
	Severity string            `json:"severity"`
	Time     int64             `json:"timestamp"`
	Message  string            `json:"message"`
	Code     string            `json:"code"`
	Args     map[string]string `json:"info"`
}

/*
The Setup function is required to implement Core's Module interface. It
initializes appropriate tables within Core's database, spawns all of the
goroutines required to make websockets work, hooks into Core's logging system
to provide access to other modules, and then establishes its web services.
*/
func (m *Messages) Setup(core shared.Core) error {
	m.db = core.Database()
	m.core = core
	m.setupTables()
	m.inbox = make(chan *shared.Log)
	m.outbox = make(map[shared.Severity](chan *shared.Log))
	m.clients = make(map[shared.Severity](map[chan *shared.Log]bool))
	go m.dispatch()
	for i := shared.Debug; i <= shared.All; i++ {
		m.outbox[shared.Severity(i)] = make(chan *shared.Log)
		m.clients[shared.Severity(i)] = make(map[chan *shared.Log]bool)
		go m.postie(shared.Severity(i))
	}
	m.functions.Post = m.postLog
	core.LoggingHook(&m.functions)
	router := core.ServiceRouter("messages")
	m.setupRoutes(router)
	return nil
}

func (m *Messages) setupRoutes(r *mux.Router) {
	r.Handle("/", &shared.ProtectedHandler{m.core, m.post}).Methods("POST")
	r.Handle("/ws/debug", &shared.ProtectedHandler{m.core, m.wsDebug})
	r.Handle("/ws/info", &shared.ProtectedHandler{m.core, m.wsInfo})
	r.Handle("/ws/warning", &shared.ProtectedHandler{m.core, m.wsWarn})
	r.Handle("/ws/error", &shared.ProtectedHandler{m.core, m.wsError})
	r.Handle("/ws/critical", &shared.ProtectedHandler{m.core, m.wsCrit})
	r.Handle("/ws/alert", &shared.ProtectedHandler{m.core, m.wsAlert})
	r.Handle("/ws/all", &shared.ProtectedHandler{m.core, m.wsAll})
	r.Handle("/debug", &shared.ProtectedHandler{m.core, m.httpDebug}).Methods("GET")
	r.Handle("/info", &shared.ProtectedHandler{m.core, m.httpInfo}).Methods("GET")
	r.Handle("/warning", &shared.ProtectedHandler{m.core, m.httpWarn}).Methods("GET")
	r.Handle("/error", &shared.ProtectedHandler{m.core, m.httpError}).Methods("GET")
	r.Handle("/critical", &shared.ProtectedHandler{m.core, m.httpCrit}).Methods("GET")
	r.Handle("/alert", &shared.ProtectedHandler{m.core, m.httpAlert}).Methods("GET")
	r.Handle("/all", &shared.ProtectedHandler{m.core, m.httpAll}).Methods("GET")
}

func (m *Messages) setupTables() {
	_, err := m.db.Exec(
		"CREATE TABLE IF NOT EXISTS messages (" +
			"id INTEGER PRIMARY KEY, " +
			"time INTEGER NULL, " +
			"sev INTEGER NULL, " +
			"msg VARCHAR(128) NULL, " +
			"code VARCHAR(128) NULL, " +
			"own VARCHAR(128) NULL, " +
			"grp VARCHAR(128) NULL, " +
			"mode INTEGER NULL" +
			")",
	)
	if err != nil {
		panic(err)
	}
	_, err = m.db.Exec(
		"CREATE TABLE IF NOT EXISTS args (" +
			"id INTEGER, " +
			"key VARCHAR(128) NULL, " +
			"value VARCHAR(128) NULL, " +
			"PRIMARY KEY (id, key), " +
			"FOREIGN KEY(id) REFERENCES messages(id) ON DELETE CASCADE" +
			")",
	)
	if err != nil {
		panic(err)
	}
}

// dispatch receives all incoming messages and redistributes them based on type
// to various outboxes.
func (m *Messages) dispatch() {
	for {
		select {
		case x := <-m.inbox:
			m.outbox[x.Severity] <- x
			m.outbox[shared.All] <- x
			if x.Severity < shared.Warn {
				m.outbox[shared.Alert] <- x
			}
		}
	}
}

// posties takes messages from dispatch's outboxes and pass them on to clients
// listening on a particular department.
func (m *Messages) postie(department shared.Severity) {
	customers := m.clients[department]
	for {
		select {
		case x := <-m.outbox[department]:
			for c := range customers {
				c <- x
			}
		}
	}
}

func (m *Messages) postLog(log *shared.Log) {
	tx, err := m.db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()
	r, err := tx.Exec(
		"INSERT INTO messages(time, sev, msg, code, own, grp, mode) "+
			"VALUES(?,?,?,?,?,?,?)",
		log.Time, log.Severity, log.Message, log.Code, log.Rules.Owner,
		log.Rules.Group, log.Rules.Mode)
	if err != nil {
		panic(err)
	}
	id, err := r.LastInsertId()
	if err != nil {
		panic(err)
	}
	if log.Args != nil {
		for key, value := range log.Args {
			_, err = tx.Exec(
				"INSERT INTO args(id, key, value) "+
					"VALUES(?,?,?)",
				id, key, value)
			if err != nil {
				panic(err)
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		panic(err)
	}
	m.inbox <- log
}
