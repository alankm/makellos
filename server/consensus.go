package server

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/sisatech/raft"
)

func (s *Server) sync(fn string, arg interface{}) interface{} {
	s.log.Debug("syncing: "+fn, "module", "fsm")
	data := encode(arg)
	ld := logData{
		Fn:  fn,
		Gob: data,
	}
	ldata := encode(ld)
	cmd := &raft.RequestClientCmd{
		Data: ldata,
	}
	resp, err := s.raft.client.SendCmdRequest(s.raft.config.Bind, cmd)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Msg)
	fmt.Println(resp)
	return resp.Msg
}

type fsm struct {
	actions map[string]func([]byte) interface{}
	log     log15.Logger
}

func (f *fsm) setup(s *Server) {
	f.log = s.log.New("module", "fsm")
	f.actions = make(map[string]func([]byte) interface{})
	f.actions["message"] = s.journal.postFSM
	f.actions["imagesDelete"] = s.images.deleteFSM
	f.actions["imagesUploadOW"] = s.images.uploadOWFSM
	f.actions["imagesUpload"] = s.images.uploadFSM
	f.actions["imagesAttrOW"] = s.images.attributesFSM
}

func encode(obj interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(obj)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func decode(data []byte, obj interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(obj)
}

type logData struct {
	Fn  string
	Gob []byte
}

func (f *fsm) Apply(log *raft.Rlog) interface{} {
	defer func() {
		r := recover()
		if r != nil {
			fmt.Println("PANIC")
		}
	}()
	f.log.Debug("committing")
	ld := new(logData)
	err := decode(log.Data, ld)
	if err != nil {
		return err
	}
	f.log.Debug(ld.Fn)
	fn := f.actions[ld.Fn]
	if fn == nil {
		return errors.New("no such function")
	}
	return fn(ld.Gob)
}

func (f *fsm) Snapshot() ([]byte, error) {
	return nil, nil // TODO: snapshot
}

func (f *fsm) Restore(snapshot []byte) error {
	return nil // TODO: restore
}

type consensus struct {
	config *raft.Config
	client *raft.Client
	server *raft.Raft
	fsm    *fsm
}

func (c *consensus) setup(s *Server, advertise string, config *raft.Config) error {
	var err error
	c.config = config
	c.fsm = new(fsm)
	c.fsm.setup(s)
	c.config.FillDefaults()
	c.config.StateMachine = c.fsm
	c.client = raft.NewClient(nil)
	c.server, err = raft.NewRaft(c.config)
	if err != nil {
		return err
	}
	properties := make(map[string]string)
	properties["HTTP"] = advertise
	c.server.PropertiesSet(properties)
	return nil
}

func (c *consensus) start() error {
	return c.server.Start()
}
