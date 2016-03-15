package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/alankm/makellos/core"
	"github.com/alankm/makellos/core/shared"
	"github.com/alankm/makellos/modules/messages"
	"github.com/alankm/makellos/modules/privileges"
)

var modules = [...]shared.Module{
	&privileges.Privileges{},
	&messages.Messages{},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: vorteil config_file\n")
		return
	}
	vorteil := core.New(os.Args[1])
	for _, module := range modules {
		vorteil.Register(module)
	}
	err := vorteil.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		return
	}
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	vorteil.Stop()
}
