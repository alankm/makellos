package images

import (
	"database/sql"
	"strings"

	"github.com/alankm/makellos/core/shared"
	"github.com/gorilla/mux"
)

type Images struct {
	db      *sql.DB
	core    shared.Core
	storage storage
}

func (i *Images) Init(db *sql.DB) error {
	i.db = db
	i.setupTables()
	return nil
}

func (i *Images) Setup(core shared.Core) error {
	i.db = core.Database()
	i.core = core
	i.setupRoutes(core.ServiceRouter("images"))
	return i.Init(i.db)
}

func (i *Images) setupRoutes(r *mux.Router) {
	r.Handle("/{path:.*}", &shared.ProtectedHandler{i.core, i.httpGet}).Methods("GET")
	r.Handle("/{path:.*}", &shared.ProtectedHandler{i.core, i.httpPost}).Methods("POST")
	r.Handle("/{path:.*}", &shared.ProtectedHandler{i.core, i.httpPut}).Methods("PUT")
	r.Handle("/{path:.*}", &shared.ProtectedHandler{i.core, i.httpDelete}).Methods("DELETE")
}

func (i *Images) setupTables() {
	_, err := i.db.Exec("CREATE TABLE IF NOT EXISTS images (" +
		"name VARCHAR(64) NULL, " +
		"parent VARCHAR(1024) NULL, " +
		"directory BOOLEAN NULL, " +
		"checksum VARCHAR(64) NULL, " +
		"date INTEGER NULL, " +
		"author VARCHAR(64) NULL, " +
		"description VARCHAR(1024) NULL," +
		"PRIMARY KEY (name, parent)" +
		");")
	if err != nil {
		panic(err)
	}
	_, err = i.db.Exec("INSERT INTO vimages VALUES(?,?,?,?,?,?,?)", "", "", true, "", 0, "", "")
	if err != nil && !strings.HasPrefix(err.Error(), "UNIQUE") {
		panic(err)
	}
}
