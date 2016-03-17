package core

import "github.com/alankm/makellos/core/shared"

func (c *Core) Login(username, password string) (shared.Session, error) {
	return c.access.Login(username, password)
}

func (c *Core) HashedLogin(username, hashword string) (shared.Session, error) {
	return c.access.HashedLogin(username, hashword)
}

func (c *Core) RegisterPath(path string) {

}
