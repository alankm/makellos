package messages

import (
	"database/sql"
	"fmt"
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

type messagesList struct {
	Length int           `json:"length"`
	List   []alertOutput `json:"list"`
}

type alertOutput struct {
	Severity string            `json:"severity"`
	Time     int64             `json:"timestamp"`
	Message  string            `json:"message"`
	Code     string            `json:"code"`
	Args     map[string]string `json:"info"`
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
	router.NewRoute().Handler(&shared.ProtectedHandler{m.core, m.post}).Path("/").Methods("POST")
	router.Handle("/ws/debug", &shared.ProtectedHandler{m.core, m.wsDebug})
	router.Handle("/debug", &shared.ProtectedHandler{m.core, m.httpDebug}).Methods("GET")
	router.Handle("/ws/info", &shared.ProtectedHandler{m.core, m.wsInfo})
	router.Handle("/info", &shared.ProtectedHandler{m.core, m.httpInfo}).Methods("GET")
	router.Handle("/ws/warning", &shared.ProtectedHandler{m.core, m.wsWarn})
	router.Handle("/warning", &shared.ProtectedHandler{m.core, m.httpWarn}).Methods("GET")
	router.Handle("/ws/error", &shared.ProtectedHandler{m.core, m.wsError})
	router.Handle("/error", &shared.ProtectedHandler{m.core, m.httpError}).Methods("GET")
	router.Handle("/ws/critical", &shared.ProtectedHandler{m.core, m.wsCrit})
	router.Handle("/critical", &shared.ProtectedHandler{m.core, m.httpCrit}).Methods("GET")
	router.Handle("/ws/alert", &shared.ProtectedHandler{m.core, m.wsAlert})
	router.Handle("/alert", &shared.ProtectedHandler{m.core, m.httpAlert}).Methods("GET")
	router.Handle("/ws/all", &shared.ProtectedHandler{m.core, m.wsAll})
	router.Handle("/all", &shared.ProtectedHandler{m.core, m.httpAll}).Methods("GET")
}

func (m *Messages) post(s shared.Session, w http.ResponseWriter, r *http.Request) {

	fmt.Println("POST REQUEST")

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
	fmt.Println(sev)
	if val, ok := shared.SeverityCodes[sev]; !ok {
		w.Write(shared.NewFailResponse(0, "missing or bad severity header").JSON())
		return
	} else {
		sevno = val
	}

	var code string
	if val, ok := r.Header["Code"]; ok {
		code = val[0]
	} else {
		w.Write(shared.NewFailResponse(0, "missing or bad code header").JSON())
		return
	}
	if code == "" {
		w.Write(shared.NewFailResponse(0, "missing or bad code header").JSON())
		return
	}

	var message string
	if val, ok := r.Header["Message"]; ok {
		message = val[0]
	} else {
		w.Write(shared.NewFailResponse(0, "missing or bad message header").JSON())
		return
	}
	if message == "" {
		w.Write(shared.NewFailResponse(0, "missing or bad message header").JSON())
		return
	}

	var args = make(map[string]string)
	var argsArray = r.Header["Args"]
	if len(args)%2 == 1 {
		w.Write(shared.NewFailResponse(0, "bad args header").JSON())
		return
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
	err = tx.Commit()
	if err != nil {
		w.Write(shared.ResponseVorteilInternal.JSON())
	} else {
		w.Write(shared.Success.JSON())
	}
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

	fmt.Println("WS REQUEST")

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
			ao := alertOutput{
				Time:     x.Time,
				Message:  x.Message,
				Code:     x.Code,
				Args:     x.Args,
				Severity: shared.SeverityStrings[x.Severity],
			}
			err := conn.WriteJSON(ao)
			if err != nil {
				return
			}
		case <-connMonitor:
			return
		}
	}
}

func (m *Messages) httpRequest(s shared.Session, w http.ResponseWriter, r *http.Request, severity shared.Severity) {

	fmt.Println("HTTP REQUEST 2")

	var offset, length int
	var start, end int64
	var sort string

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

	val, ok = vals["start"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 64)
		start = int64(x)
	}

	val, ok = vals["end"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 64)
		end = int64(x)
	} else {
		end = time.Now().Unix()
	}

	val, ok = vals["sort"]
	if ok && len(val) > 0 && val[0] == "true" {
		sort = "severity DESC, id DESC"
	} else {
		sort = "id DESC"
	}

	sevString := "("
	switch severity {
	case shared.Alert:
		sevString += strconv.Itoa(shared.Warn)
		sevString += ", "
		sevString += strconv.Itoa(shared.Error)
		sevString += ", "
		sevString += strconv.Itoa(shared.Crit)
	case shared.All:
		sevString += strconv.Itoa(shared.Debug)
		sevString += ", "
		sevString += strconv.Itoa(shared.Info)
		sevString += ", "
		sevString += strconv.Itoa(shared.Warn)
		sevString += ", "
		sevString += strconv.Itoa(shared.Error)
		sevString += ", "
		sevString += strconv.Itoa(shared.Crit)
	default:
		sevString += strconv.Itoa(int(severity))
	}
	sevString += ")"

	fmt.Printf("%v %v %v\n", offset, length, sort)

	groups := s.Groups()
	groupsString := "("
	for i, group := range groups {
		groupsString += "'" + group + "'"
		if i < len(groups)-1 {
			groupsString += ", "
		}
	}
	groupsString += ")"

	fmt.Println(sevString)
	fmt.Println(groupsString)
	rows, err := m.db.Query("SELECT * FROM messages WHERE (time BETWEEN ? AND ?) AND (severity IN "+sevString+") AND ((rulesowner = ? AND (rulesmode & 0x100) = 0x100) OR (rulesowner != ? AND rulesgroup IN "+groupsString+" AND (rulesmode & 0x020) = 0x020) OR (rulesowner != ? AND rulesgroup NOT IN "+groupsString+" AND (rulesmode & 0x004) = 0x004)) ORDER BY ? LIMIT ?,?", start, end, s.Username(), s.Username(), s.Username(), sort, offset, length)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var out []alertOutput
	for rows.Next() {
		var id, time, severity, mode int
		var message, code string
		var owner, group string
		err = rows.Scan(&id, &time, &severity, &message, &code, &owner, &group, &mode)
		if err != nil {
			panic(err)
		}

		ao := alertOutput{
			Time:     int64(time),
			Message:  message,
			Code:     code,
			Args:     make(map[string]string),
			Severity: shared.SeverityStrings[shared.Severity(severity)],
		}

		argrows, err := m.db.Query("SELECT * FROM args WHERE id=?", id)
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

	row := m.db.QueryRow("SELECT COUNT(*) FROM messages WHERE (time BETWEEN ? AND ?) AND (severity IN "+sevString+") AND ((rulesowner = ? AND (rulesmode & 0x100) = 0x100) OR (rulesowner != ? AND rulesgroup IN "+groupsString+" AND (rulesmode & 0x020) = 0x020) OR (rulesowner != ? AND rulesgroup NOT IN "+groupsString+" AND (rulesmode & 0x004) = 0x004))", start, end, s.Username(), s.Username(), s.Username())
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var count int
	err = row.Scan(&count)
	if err != nil {
		panic(err)
	}

	response := shared.NewSuccessResponse(&messagesList{count, out})
	w.Write(response.JSON())
}

func (m *Messages) setupTables() {
	_, err := m.db.Exec(
		"CREATE TABLE IF NOT EXISTS messages (" +
			"id INTEGER PRIMARY KEY, " +
			"time INTEGER NULL, " +
			"severity INTEGER NULL, " +
			"messagetext VARCHAR(128) NULL, " +
			"messagecode VARCHAR(128) NULL, " +
			"rulesowner VARCHAR(128) NULL, " +
			"rulesgroup VARCHAR(128) NULL, " +
			"rulesmode INTEGER NULL" +
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
	fmt.Printf("POST: %v\n", log.Severity)
	result, err := tx.Exec("INSERT INTO messages(time, severity, messagetext, messagecode, rulesowner, rulesgroup, rulesmode) VALUES(?,?,?,?,?,?,?)", log.Time, log.Severity, log.Message, log.Code, log.Rules.Owner, log.Rules.Group, log.Rules.Mode)
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
