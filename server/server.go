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
	images  Images
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
	s.log.Debug("HAI")
	//s.failOnError(s.raft.setup(s, s.conf.Advertise, &s.conf.Raft), "setting up raft")
	s.log.Debug("HAI")
	s.journal.setup(s, s.log.New("module", "journal"))
	s.log.Debug("HAI")
	s.failOnError(s.images.setup(s, s.log.New("module", "images")), "setting up images")
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
	r := new(Rules)
	r.Owner = "server"
	r.Group = "server"
	r.Mode = 0777
	s.failOnError(s.data.insertFile("folder", "", "", r), "adding files to database")

	// login
	s.web.mux.HandleFunc(s.servicesVersionString()+"/login", s.handlerLogin).Methods("POST")

	// Messages
	s.journal.setupRoutes(s.web.mux.PathPrefix(s.servicesVersionString() + "/messages").Subrouter())
	s.failOnError(s.data.insertFile("service", "", "messages", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "debug", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "info", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "warning", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "error", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "critical", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "alert", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "all", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages", "ws", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "debug", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "info", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "warning", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "error", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "critical", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "alert", r), "adding files to database")
	s.failOnError(s.data.insertFile("service", "/messages/ws", "all", r), "adding files to database")

	// Images
	s.images.setupRoutes(s.web.mux.PathPrefix(s.servicesVersionString() + "/images").Subrouter())

	// website
	s.web.mux.HandleFunc("/{path:.*}", s.websiteHandler).Methods("GET")
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
