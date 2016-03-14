package core

import (
	"os"
	"reflect"

	"github.com/alankm/makellos/core/shared"

	"gopkg.in/inconshreveable/log15.v2"
)

type Core struct {
	err     error
	log     log15.Logger
	conf    config
	logging *shared.LoggingFunctions
	access  *shared.AuthorizationFunctions
}

func New(config string) *Core {
	c := new(Core)
	c.err = c.conf.load(config)
	c.init()
	return nil
}

func (c *Core) init() {
	if c.err == nil {
		c.err = os.MkdirAll(c.conf.Base, 0775)
		c.log = log15.New("module", reflect.TypeOf(c).Name())
	}
}

func (c *Core) Start() error {
	if c.err == nil {
		c.log.Info("Starting...")
	}
	return c.err
}

func (c *Core) Stop() {

}
