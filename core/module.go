package core

import (
	"database/sql"

	"github.com/alankm/makellos/core/shared"
)

func (c *Core) Register(module shared.Module) {
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
