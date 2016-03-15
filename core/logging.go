package core

import (
	"database/sql"

	"github.com/alankm/makellos/core/shared"
)

func (c *Core) Log(tx *sql.Tx, log *shared.Log) {
	c.logging.Post(tx, log)
}
