package core

import (
	"reflect"

	"github.com/alankm/makellos/core/shared"
	"gopkg.in/inconshreveable/log15.v2"
)

func (c *Core) Log(log *shared.Log) {
	c.logging.Post(log)
}

func (c *Core) SubLogger(m shared.Module) log15.Logger {
	return c.log.New("module", reflect.TypeOf(m).Name())
}
