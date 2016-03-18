package core

import (
	"bytes"
	"encoding/gob"
	"reflect"

	"github.com/alankm/makellos/core/shared"
	"github.com/sisatech/raft"
)

type stateMachine struct {
	c          *Core
	components map[string]func([]byte) []byte
}

func (c *Core) RegisterSyncFunction(module shared.Module, fn func([]byte) []byte) {
	c.raft.fsm.components[reflect.TypeOf(module).Name()] = fn
}

func (c *Core) Encode(obj interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(obj)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (c *Core) Decode(data []byte, obj interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(obj)
}

func (c *Core) Sync(module shared.Module, data []byte) interface{} {
	m := make(map[string][]byte)
	m[reflect.TypeOf(module).Name()] = data
	command := c.Encode(m)
	cmd := &raft.RequestClientCmd{
		Data: command,
	}
	resp, err := c.raft.client.SendCmdRequest(c.raft.config.Bind, cmd)
	if err != nil || resp.Msg == nil {
		panic(err)
	}
	return resp.Msg
}

func (s *stateMachine) Apply(log *raft.Rlog) interface{} {
	m := make(map[string][]byte)
	s.c.Decode(log.Data, &m)
	for key, val := range m {
		return s.components[key](val)
	}
	return nil
}

func (s *stateMachine) Snapshot() ([]byte, error) {
	return nil, nil
}

func (s *stateMachine) Restore([]byte) error {
	return nil
}
