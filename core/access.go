package core

func (c *Core) Login() {
	c.access.Login()
}

func (c *Core) HashedLogin() {
	c.access.HashedLogin()
}

func (c *Core) GetRules() {
	c.access.GetRules()
}

func (c *Core) Read() {
	c.access.Read()
}

func (c *Core) Write() {
	c.access.Write()
}

func (c *Core) Exec() {
	c.access.Exec()
}

func (c *Core) Mkdir() {
	c.access.Mkdir()
}

func (c *Core) Mkfile() {
	c.access.Mkfile()
}

func (c *Core) Delete() {
	c.access.Delete()
}

func (c *Core) Chown() {
	c.access.Chown()
}

func (c *Core) Chgrp() {
	c.access.Chgrp()
}

func (c *Core) Chmod() {
	c.access.Chmod()
}
