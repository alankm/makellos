package shared

type Core interface {
	Register(Module)
	LoggingHook(*LoggingFunctions)
	AuthorizationHook(*AuthorizationFunctions)
	Log(*Log)
	Login()
	HashedLogin()
	GetRules()
	Read()
	Write()
	Exec()
	Mkdir()
	Mkfile()
	Delete()
	Chown()
	Chgrp()
	Chmod()
}

type Module interface {
	Setup(Core) error
}

type Rules struct {
	Owner [32]byte
	Group [32]byte
	Mode  [2]byte
}

type Severity uint8

const (
	Debug = iota
	Info
	Warn
	Error
	Crit
	Alert
	All
)

type Log struct {
	Severity Severity
	Rules    Rules
	Code     string
	Message  string
	Args     map[string]string
}

type LoggingFunctions struct {
	Post func(*Log)
}

type AuthorizationFunctions struct {
	Login       func()
	HashedLogin func()
	GetRules    func()
	Read        func()
	Write       func()
	Exec        func()
	Mkdir       func()
	Mkfile      func()
	Delete      func()
	Chown       func()
	Chgrp       func()
	Chmod       func()
}
