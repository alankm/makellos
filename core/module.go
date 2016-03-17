package core

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"

	"github.com/alankm/makellos/core/shared"
)

func (c *Core) Register(module shared.Module) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		fmt.Fprintf(os.Stderr, "panic whilst registering '%v'\n%v\n",
			reflect.TypeOf(module).Name(), r)
		os.Exit(1)
	}()
	if c.err == nil {
		c.err = module.Setup(c)
	}
}

func (c *Core) AuthorizationHook(functions *shared.AuthorizationFunctions) {
	c.access = functions
}

func (c *Core) LoggingHook(functions *shared.LoggingFunctions) {
	c.logging = functions
}

func (c *Core) Database() *sql.DB {
	return c.db
}

func (c *Core) NotifyLeaderChange(monitor *chan bool) {

}

func (c *Core) UnsubscribeLeaderChange(monitor *chan bool) {

}
