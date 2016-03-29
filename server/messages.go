package server

import (
	"net/http"
	"strconv"
	"time"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Severity int

const (
	Debug = iota
	Info
	Warn
	Error
	Crit
	Alert
	All
)

var SeverityCodes = map[string]Severity{
	"debug":    Debug,
	"info":     Info,
	"warning":  Warn,
	"error":    Error,
	"critical": Crit,
}

var SeverityStrings = map[Severity]string{
	Debug: "debug",
	Info:  "info",
	Warn:  "warning",
	Error: "error",
	Crit:  "critical",
}

type Log struct {
	Severity Severity
	Time     int64
	Rules    Rules
	Code     string
	Message  string
	Args     map[string]string
}

/*
Messages implements Vorteil's 'Module' interface. It exists to add a powerful
and advanced logging & messaging system for use by other modules to Vorteil. It
is also a Vorteil service, providing web-accessible interfaces to access and
interact with its data.
*/
type Messages struct {
	log     log15.Logger
	s       *Server
	inbox   chan *Log
	outbox  map[Severity](chan *Log)
	clients map[Severity](map[chan *Log]bool)
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
Setup is required to implement Core's Module interface. It initializes
appropriate tables within Core's database, spawns all of the goroutines
required to make websockets work, hooks into Core's logging system to provide
access to other modules, and then establishes its web services.
*/
func (m *Messages) setup(s *Server, log log15.Logger) {
	m.s = s
	m.log = log
	m.inbox = make(chan *Log)
	m.outbox = make(map[Severity](chan *Log))
	m.clients = make(map[Severity](map[chan *Log]bool))
	go m.dispatch()
	for i := Debug; i <= All; i++ {
		m.outbox[Severity(i)] = make(chan *Log)
		m.clients[Severity(i)] = make(map[chan *Log]bool)
		go m.postie(Severity(i))
	}
	m.log.Debug("messages setup")
}

func (m *Messages) setupRoutes(r *mux.Router) {
	r.Handle("/", &ProtectedHandler{m.s, m.post}).Methods("POST")
	r.Handle("/ws/debug", &ProtectedHandler{m.s, m.wsDebug})
	r.Handle("/ws/info", &ProtectedHandler{m.s, m.wsInfo})
	r.Handle("/ws/warning", &ProtectedHandler{m.s, m.wsWarn})
	r.Handle("/ws/error", &ProtectedHandler{m.s, m.wsError})
	r.Handle("/ws/critical", &ProtectedHandler{m.s, m.wsCrit})
	r.Handle("/ws/alert", &ProtectedHandler{m.s, m.wsAlert})
	r.Handle("/ws/all", &ProtectedHandler{m.s, m.wsAll})
	r.Handle("/debug", &ProtectedHandler{m.s, m.httpDebug}).Methods("GET")
	r.Handle("/info", &ProtectedHandler{m.s, m.httpInfo}).Methods("GET")
	r.Handle("/warning", &ProtectedHandler{m.s, m.httpWarn}).Methods("GET")
	r.Handle("/error", &ProtectedHandler{m.s, m.httpError}).Methods("GET")
	r.Handle("/critical", &ProtectedHandler{m.s, m.httpCrit}).Methods("GET")
	r.Handle("/alert", &ProtectedHandler{m.s, m.httpAlert}).Methods("GET")
	r.Handle("/all", &ProtectedHandler{m.s, m.httpAll}).Methods("GET")
}

// dispatch receives all incoming messages and redistributes them based on type
// to various outboxes.
func (m *Messages) dispatch() {
	for {
		select {
		case x := <-m.inbox:
			m.outbox[x.Severity] <- x
			m.outbox[All] <- x
			if x.Severity < Warn {
				m.outbox[Alert] <- x
			}
		}
	}
}

// posties takes messages from dispatch's outboxes and pass them on to clients
// listening on a particular department.
func (m *Messages) postie(department Severity) {
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

func (m *Messages) postLog(log *Log) error {
	err := m.s.data.postMessagesLog(log)
	if err != nil {
		return err
	}
	m.inbox <- log
	return nil
}

func (m *Messages) httpDebug(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, Debug)
}

func (m *Messages) httpInfo(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, Info)
}

func (m *Messages) httpWarn(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, Warn)
}

func (m *Messages) httpError(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, Error)
}

func (m *Messages) httpCrit(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, Crit)
}

func (m *Messages) httpAlert(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, Alert)
}

func (m *Messages) httpAll(s *Session, w http.ResponseWriter, r *http.Request) {
	m.http(s, w, r, All)
}

func (m *Messages) wsDebug(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, Debug)
}

func (m *Messages) wsInfo(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, Info)
}

func (m *Messages) wsWarn(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, Warn)
}

func (m *Messages) wsError(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, Error)
}

func (m *Messages) wsCrit(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, Crit)
}

func (m *Messages) wsAlert(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, Alert)
}

func (m *Messages) wsAll(s *Session, w http.ResponseWriter, r *http.Request) {
	m.ws(s, w, r, All)
}

func (m *Messages) post(s *Session, w http.ResponseWriter, r *http.Request) {
	var sev, code, message string
	var sevno Severity
	var ok bool
	// Severity
	if val, ok := r.Header["Severity"]; ok && len(val) == 1 {
		sev = val[0]
	}
	if sevno, ok = SeverityCodes[sev]; !ok {
		w.Write(NewFailResponse(0, "missing or bad severity header").JSON())
		return
	}
	// Code
	if val, ok := r.Header["Code"]; ok && len(val) == 1 && val[0] != "" {
		code = val[0]
	} else {
		w.Write(NewFailResponse(0, "missing or bad code header").JSON())
		return
	}
	// Message
	if val, ok := r.Header["Message"]; ok && len(val) == 1 && val[0] != "" {
		message = val[0]
	} else {
		w.Write(NewFailResponse(0, "missing or bad message header").JSON())
		return
	}
	// Args
	var args = make(map[string]string)
	var argsArray = r.Header["Args"]
	if len(args)%2 == 1 {
		w.Write(NewFailResponse(0, "bad args header").JSON())
		return
	}
	for i := 0; i < len(argsArray); i = i + 2 {
		args[argsArray[i]] = argsArray[i+1]
	}

	log := &Log{
		Severity: sevno,
		Time:     time.Now().Unix(),
		Rules: Rules{
			Owner: s.User.Name(),
			Group: s.User.PrimaryGroup(),
			Mode:  s.Mode(),
		},
		Code:    code,
		Message: message,
		Args:    args,
	}

	ret := m.s.sync("message", log)
	if ret == nil {
		m.log.Debug("nil fsm response!")
		w.Write(Success.JSON())
	}

	err, ok := ret.(error)
	if !ok {
		panic(nil)
	}
	if err != nil {
		w.Write(ResponseVorteilInternal.JSON())
		return
	}
}

func (m *Messages) postFSM(args []byte) interface{} {
	log := new(Log)
	err := decode(args, log)
	if err != nil {
		return err
	}
	return m.s.data.postMessagesLog(log)
}

func (m *Messages) ws(s *Session, w http.ResponseWriter, r *http.Request, severity Severity) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.Write(ResponseVorteilInternal.JSON())
		return
	}
	// goroutine to health-check websocket
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
	// Vorteil leader status health-check
	leaderMonitor := make(chan bool)
	m.s.NotifyLeaderChange(&leaderMonitor)
	defer m.s.UnsubscribeLeaderChange(&leaderMonitor)
	// listen for more messages
	monitor := make(chan *Log)
	m.clients[severity][monitor] = true
	defer delete(m.clients[severity], monitor)
	for {
		select {
		case <-leaderMonitor:
			return
		case x := <-monitor:
			if s.CanRead(&x.Rules) {
				ao := message{
					Time:     x.Time,
					Message:  x.Message,
					Code:     x.Code,
					Args:     x.Args,
					Severity: SeverityStrings[x.Severity],
				}
				err := conn.WriteJSON(ao)
				if err != nil {
					return
				}
			}
		case <-connMonitor:
			return
		}
	}
}

func (m *Messages) http(s *Session, w http.ResponseWriter, r *http.Request, severity Severity) {
	var offset, length int
	var start, end int64
	var sort string
	// offset
	vals := r.URL.Query()
	val, ok := vals["offset"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 32)
		offset = int(x)
	}
	// length
	val, ok = vals["length"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 32)
		length = int(x)
	} else {
		length = -1
	}
	// start
	val, ok = vals["start"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 64)
		start = int64(x)
	}
	// end
	val, ok = vals["end"]
	if ok && len(val) > 0 {
		x, _ := strconv.ParseUint(val[0], 10, 64)
		end = int64(x)
	} else {
		end = time.Now().Unix()
	}
	// sort
	val, ok = vals["sort"]
	if ok && len(val) > 0 && val[0] == "true" {
		sort = "severity DESC, id DESC"
	} else {
		sort = "id DESC"
	}

	count, out := m.s.data.getMessages(s, severity, sort, start, end, offset, length)
	response := NewSuccessResponse(&listPayload{count, out})
	w.Write(response.JSON())
}
