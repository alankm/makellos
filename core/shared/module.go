package shared

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/gorilla/mux"
)

var sharedLogger log15.Logger = log15.New("module", "shared")

type Core interface {
	Register(Module)
	LoggingHook(*LoggingFunctions)
	AuthorizationHook(*AuthorizationFunctions)
	Database() *sql.DB
	SubLogger(Module) log15.Logger
	// General
	IsLeader() bool
	Leader() string
	ValidateCookie(*http.Cookie) Session
	ServiceRouter(string) *mux.Router
	NotifyLeaderChange(*chan bool)
	UnsubscribeLeaderChange(*chan bool)
	RegisterSyncFunction(Module, func([]byte) []byte)
	Encode(interface{}) []byte
	Decode([]byte, interface{}) error
	Sync(Module, []byte) interface{}
	// Logging
	Log(*Log)
	// Access
	RegisterPath(string)
	Login(username, password string) (Session, error)
	HashedLogin(username, hashword string) (Session, error)
}

type Module interface {
	Setup(Core) error
}

type Rules struct {
	Owner string
	Group string
	Mode  uint16
}

type Session interface {
	Username() string
	Hashword() string
	Groups() []string
	GID() string
	Mode() uint16
	SU() bool
	CanRead(*Rules) bool
}

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

type LoggingFunctions struct {
	Post func(*Log)
}

type AuthorizationFunctions struct {
	Login       func(username, password string) (Session, error)
	HashedLogin func(username, hashword string) (Session, error)
}

type ProtectedHandler struct {
	Core    Core
	Handler func(Session, http.ResponseWriter, *http.Request)
}

func (h *ProtectedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func(c Core, w http.ResponseWriter) {
		rec := recover()
		if r == nil {
			return
		}
		sharedLogger.Error("recovered from panic", r.Method+": "+r.URL.String(), rec)
		fmt.Fprintf(os.Stderr, "stack: \n%v\n", string(debug.Stack()))
		w.Write(ResponseVorteilInternal.JSON())
	}(h.Core, w)
	if !h.Core.IsLeader() {
		w.Write(ResponseLeader.SetInfo("leader", h.Core.Leader()).JSON())
		return
	}
	s := h.Login(r)
	if s == nil {
		w.Write(ResponseAuthentication.JSON())
		return
	}
	h.Handler(s, w, r)
}

func (h *ProtectedHandler) Login(r *http.Request) Session {
	if cookie, err := r.Cookie("vorteil"); err == nil {
		return h.Core.ValidateCookie(cookie)
	}
	return nil
}
