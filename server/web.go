package server

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/sisatech/multiserver"
)

type web struct {
	server   *multiserver.HTTPServer
	mux      *mux.Router
	cookie   *securecookie.SecureCookie
	services []string
}

func (w *web) setup(config *configuration) {
	w.mux = mux.NewRouter()
	w.server = multiserver.NewHTTPServer(config.Bind, w.mux, nil)
	w.cookie = securecookie.New(securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
}

func (w *web) start() error {
	return w.server.Start()
}
