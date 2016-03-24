package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/alankm/simplicity/server"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: vorteil config_file\n")
		return
	}
	vorteil := new(server.Server)
	vorteil.Setup(os.Args[1])
	vorteil.Start()
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	select {
	case <-ch:
	case <-vorteil.FailChannel():
	}
	vorteil.Stop()
}
