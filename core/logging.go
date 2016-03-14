package core

import "github.com/alankm/makellos/core/shared"

func (c *Core) Log(log *shared.Log) {
	c.logging.Post(log)
}
