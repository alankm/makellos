package server

import (
	"gopkg.in/inconshreveable/log15.v2"

	"github.com/alankm/simplicity/server/access"
)

type Server struct {
	err     error
	started bool
	log     log15.Logger
	fail    chan bool
	conf    configuration
	data    Data
	web     web
	access  access.Access
	journal Messages
	raft    consensus
}

func (s *Server) Setup(configPath string) {
	defer func() {
		recover()
	}()
	s.log = log15.Root()
	s.fail = make(chan bool)
	s.failOnError(s.conf.load(configPath), "loading config file")
	s.failOnError(s.data.Setup(s.conf.Base, s.conf.Database, s.log.New("module", "data")), "initializing database")
	s.access, s.err = access.New(s.conf.Modules["access"], s.log.New("module", "access"), s.data.Database())
	s.failOnError(s.err, "setting up access")
	s.web.setup(&s.conf)
	s.failOnError(s.raft.setup(s, s.conf.Advertise, &s.conf.Raft), "setting up raft")
	s.journal.setup(s, s.log.New("module", "journal"))
	s.setupRoutes()
}

func (s *Server) Start() {
	if s.err == nil && !s.started {
		s.started = true
		s.failOnError(s.raft.start(), "starting raft server")
		s.failOnError(s.web.start(), "starting web server")
		s.log.Info("Vorteil started")
	}
}

func (s *Server) Stop() {
	if s.started {
		s.started = false
		s.log.Info("Vorteil stopped safely")
	}
}

func (s *Server) setupRoutes() {
	s.web.mux.HandleFunc(s.servicesVersionString()+"/login", s.handlerLogin).Methods("POST")
	s.journal.setupRoutes(s.web.mux.PathPrefix(s.servicesVersionString() + "/messages").Subrouter())
}

func (s *Server) servicesVersionString() string {
	return "/services/" + s.conf.Version
}

func (s *Server) failOnError(err error, message string) {
	if err != nil {
		s.err = err
		s.log.Crit(message + ":\n\t\t\t" + err.Error())
		go func(s *Server) {
			s.fail <- true
		}(s)
		panic(nil)
	}
}

func (s *Server) FailChannel() <-chan bool {
	return s.fail
}

func (s *Server) NotifyLeaderChange(ch *chan bool) {

}

func (s *Server) UnsubscribeLeaderChange(ch *chan bool) {

}
