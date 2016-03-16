package shared

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
)

type Core interface {
	Register(Module)
	LoggingHook(*LoggingFunctions)
	AuthorizationHook(*AuthorizationFunctions)
	Database() *sql.DB
	// General
	IsLeader() bool
	Leader() string
	ValidateCookie(*http.Cookie) Session
	ServiceRouter(string) *mux.Router
	// Logging
	Log(*sql.Tx, *Log)
	// Access
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
	Groups() []string
	GID() string
	Mode() uint16
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
	Post func(*sql.Tx, *Log)
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
