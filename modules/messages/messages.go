package messages

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/alankm/makellos/core/shared"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Messages struct {
	functions shared.LoggingFunctions
	db        *sql.DB
	core      shared.Core
	inbox     chan *shared.Log
	outbox    map[shared.Severity](chan *shared.Log)
	customers map[shared.Severity](map[chan *shared.Log]bool)
}

func (m *Messages) Setup(core shared.Core) error {
	m.db = core.Database()
	m.core = core
	m.setupTables()
	m.functions.Post = m.Post
	m.inbox = make(chan *shared.Log)
	m.outbox = make(map[shared.Severity](chan *shared.Log))
	m.customers = make(map[shared.Severity](map[chan *shared.Log]bool))
	go m.dispatch()
	for i := shared.Debug; i <= shared.All; i++ {
		m.outbox[shared.Severity(i)] = make(chan *shared.Log)
		m.customers[shared.Severity(i)] = make(map[chan *shared.Log]bool)
		go m.postie(shared.Severity(i))
	}
	core.LoggingHook(&m.functions)
	router := core.ServiceRouter("messages")
	m.setupServices(router)
	return nil
}

func (m *Messages) setupServices(router *mux.Router) {
	router.Handle("", &shared.ProtectedHandler{m.core, m.post}).Methods("POST")
	router.Handle("/debug", &shared.ProtectedHandler{m.core, m.wsDebug}).Schemes("ws")
	router.Handle("/debug", &shared.ProtectedHandler{m.core, m.httpDebug}).Methods("GET")
	router.Handle("/info", &shared.ProtectedHandler{m.core, m.wsInfo}).Schemes("ws")
	router.Handle("/info", &shared.ProtectedHandler{m.core, m.httpInfo}).Methods("GET")
	router.Handle("/warning", &shared.ProtectedHandler{m.core, m.wsWarn}).Schemes("ws")
	router.Handle("/warning", &shared.ProtectedHandler{m.core, m.httpWarn}).Methods("GET")
	router.Handle("/error", &shared.ProtectedHandler{m.core, m.wsError}).Schemes("ws")
	router.Handle("/error", &shared.ProtectedHandler{m.core, m.httpError}).Methods("GET")
	router.Handle("/critical", &shared.ProtectedHandler{m.core, m.wsCrit}).Schemes("ws")
	router.Handle("/critical", &shared.ProtectedHandler{m.core, m.httpCrit}).Methods("GET")
	router.Handle("/alert", &shared.ProtectedHandler{m.core, m.wsAlert}).Schemes("ws")
	router.Handle("/alert", &shared.ProtectedHandler{m.core, m.httpAlert}).Methods("GET")
	router.Handle("/all", &shared.ProtectedHandler{m.core, m.wsAll}).Schemes("ws")
	router.Handle("/all", &shared.ProtectedHandler{m.core, m.httpAll}).Methods("GET")
}

func (m *Messages) post(s shared.Session, w http.ResponseWriter, r *http.Request) {
	tx, err := m.db.Begin()
	if err != nil {
		w.Write(shared.ResponseVorteilInternal.JSON())
	}
	defer tx.Rollback()

	var sev string
	var sevno shared.Severity
	if val, ok := r.Header["Severity"]; ok {
		sev = val[0]
	}
	if val, ok := shared.SeverityCodes[sev]; !ok {
		sevno = val
		w.Write(shared.NewFailResponse(0, "missing or bad severity header").JSON())
	}

	var code string
	if val, ok := r.Header["Code"]; ok {
		code = val[0]
	}
	if code == "" {
		w.Write(shared.NewFailResponse(0, "missing or bad code header").JSON())
	}

	var message string
	if val, ok := r.Header["Message"]; ok {
		message = val[0]
	}
	if message == "" {
		w.Write(shared.NewFailResponse(0, "missing or bad message header").JSON())
	}

	var args = make(map[string]string)
	var argsArray = r.Header["Args"]
	if len(args)%2 == 1 {
		w.Write(shared.NewFailResponse(0, "bad args header").JSON())
	}
	for i := 0; i < len(argsArray); i = i + 2 {
		args[argsArray[i]] = argsArray[i+1]
	}

	log := &shared.Log{
		Severity: sevno,
		Time:     time.Now().Unix(),
		Rules: shared.Rules{
			Owner: s.Username(),
			Group: s.GID(),
			Mode:  s.Mode(),
		},
		Code:    code,
		Message: message,
		Args:    args,
	}
	m.Post(tx, log)
	tx.Commit()
}

func (m *Messages) wsDebug(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.Debug)
}

func (m *Messages) httpDebug(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.Debug)
}

func (m *Messages) wsInfo(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.Info)
}

func (m *Messages) httpInfo(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.Info)
}

func (m *Messages) wsWarn(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.Warn)
}

func (m *Messages) httpWarn(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.Warn)
}

func (m *Messages) wsError(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.Error)
}

func (m *Messages) httpError(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.Error)
}

func (m *Messages) wsCrit(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.Crit)
}

func (m *Messages) httpCrit(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.Crit)
}

func (m *Messages) wsAlert(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.Alert)
}

func (m *Messages) httpAlert(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.Alert)
}

func (m *Messages) wsAll(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.wsRequest(s, w, r, shared.All)
}

func (m *Messages) httpAll(s shared.Session, w http.ResponseWriter, r *http.Request) {
	m.httpRequest(s, w, r, shared.All)
}

func (m *Messages) wsRequest(s shared.Session, w http.ResponseWriter, r *http.Request, severity shared.Severity) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.Write(shared.ResponseVorteilInternal.JSON())
		return
	}

	connMonitor := make(chan bool)
	go func(c *websocket.Conn, monitor chan bool) {
		for {
			if _, _, err := c.NextReader(); err != nil {
				monitor <- true
				c.Close()
				break
			}
		}
	}(conn, connMonitor)

	monitor := make(chan *shared.Log)
	m.customers[severity][monitor] = true
	defer delete(m.customers[severity], monitor)

	for {
		select {
		//case <-a.vorteil.leaderMonitor.ch:
		//return
		// case <- connection lost
		case x := <-monitor:
			err := conn.WriteJSON(x)
			if err != nil {
				return
			}
		case <-connMonitor:
			return
		}
	}
}

func (m *Messages) httpRequest(s shared.Session, w http.ResponseWriter, r *http.Request, severity shared.Severity) {
	var offset int
	var length int
	vals := r.URL.Query()
	val, ok := vals["offset"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 32)
		offset = int(x)
	}

	val, ok = vals["length"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 32)
		length = int(x)
	} else {
		length = -1
	}

	//////	///////	//////	///////	//////	/////	//////	//////	//////
	/	//	//	//	////	/	/	/		/	/
	/	//	//	//////	///	///	//	/	////	////
	/	///	/////	/////	///	//	/////	//////	//////	//
	/	//	//	//	/////	///	////	//	///	/
	rows, err := a.db.Query("SELECT * FROM alerts WHERE severity!=? AND WHERE severity!=? ORDER BY id ASC LIMIT ?,?", "info", "debug", offset, length)
	if err != nil {
		a.vorteil.Alert(s, Error, "blah", "SQL query issue: "+err.Error(), a.Name(), nil)
	}
	defer rows.Close()

	var out []alertOutput

	for rows.Next() {

		var id, time int
		var severity, message, code string
		var owner, group, mode string

		err = rows.Scan(&id, &time, &severity, &message, &code, &owner, &group, &mode)
		if err != nil {
			a.vorteil.Alert(s, Error, "blah", "SQL query issue: "+err.Error(), a.Name(), nil)
		}

		//rules := privileges.NewRules(owner, group, mode)
		//if !s.CanRead(rules) {
		//	continue
		//}

		ao := alertOutput{
			Time:    int64(time),
			Message: message,
			Code:    code,
			Args:    make(map[string]string),
		}

		argrows, err := a.db.Query("SELECT * FROM args WHERE id=?", id)
		if err != nil {
			a.vorteil.Alert(s, Error, "blah", "SQL query issue: "+err.Error(), a.Name(), nil)
		}
		defer argrows.Close()

		for argrows.Next() {
			var rid int
			var key, value string
			err = argrows.Scan(&rid, &key, &value)
			if err != nil {
				a.vorteil.Alert(s, Error, "blah", "SQL query issue: "+err.Error(), a.Name(), nil)
			}

			ao.Args[key] = value

		}

		out = append(out, ao)
		response := NewSuccessResponse(out)
		w.Write(response.JSON())

	}
}

func (m *Messages) setupTables() {
	_, err := m.db.Exec(
		"CREATE TABLE IF NOT EXISTS messages (" +
			"id INTEGER PRIMARY KEY, " +
			"time INTEGER NULL, " +
			"severity VARCHAR(32) NULL, " +
			"messagetext VARCHAR(128) NULL, " +
			"messagecode VARCHAR(128) NULL, " +
			"rulesowner VARCHAR(128) NULL, " +
			"rulesgroup VARCHAR(128) NULL, " +
			"rulesmode VARCHAR(4) NULL" +
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

func (m *Messages) postie(department shared.Severity) {
	customers := m.customers[department]
	for {
		select {
		case x := <-m.outbox[department]:
			for c := range customers {
				c <- x
			}
		}
	}
}

func (m *Messages) Post(tx *sql.Tx, log *shared.Log) {
	m.inbox <- log
	result, err := tx.Exec("INSERT INTO alerts(time, severity, messagetext, messagecode, rulesowner, rulesgroup, rulesmode) VALUES(?,?,?,?,?,?,?)", log.Time, log.Severity, log.Message, log.Code, log.Rules.Owner, log.Rules.Group, log.Rules.Mode)
	if err != nil {
		panic(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	if log.Args != nil {
		for key, value := range log.Args {
			_, err = tx.Exec("INSERT INTO args(id, key, value) VALUES(?,?,?)", id, key, value)
			if err != nil {
				panic(err)
			}
		}
	}
}
