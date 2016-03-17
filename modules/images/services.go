package images

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/alankm/makellos/core/shared"
	"github.com/gorilla/mux"
)

type fileAttributes struct {
	Name        string `json:"name"`
	Date        uint64 `json:"date"`
	Checksum    string `json:"checksum"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

type listElement struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type listPayload struct {
	Length int           `json:"length"`
	List   []listElement `json:"list"`
}

func (i *Images) httpGet(s shared.Session, w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	rec := i.lookup(path)
	if rec == nil {
		w.Write(shared.NewFailResponse(0, "target not found").JSON())
		return
	}

	if rec.dir {
		// retrieve directory children
		length := -1
		offset := 0
		if vals, ok := r.URL.Query()["length"]; ok && len(vals) == 1 {
			length, err := strconv.ParseInt(val[0], 10, 32)
			if err != nil {
				w.Write(shared.NewFailResponse(0, "bad length parameter").JSON())
				return
			}
		}
		if vals, ok := r.URL.Query()["offset"]; ok && len(vals) == 1 {
			offset, err := strconv.ParseInt(val[0], 10, 32)
			if err != nil {
				w.Write(shared.NewFailResponse(0, "bad offset parameter").JSON())
				return
			}
		}

		rows, count := i.lookupChildren(path, length, offset)
		if rows == nil {
			panic(errors.New("rows is nil"))
		}
		defer rows.Close()
		var list listElement
		for rows.Next() {
			rec := recordFromRows(rows)
			tstr := "file"
			if rec.dir {
				tstr = "folder"
			}
			list = append(list, listElement{
				Name: rec.name,
				Type: tstr,
			})
		}
		w.Write(shared.NewSuccessResponse(&listPayload{
			Length: count,
			List:   list,
		}).JSON())
		return
	}

	attr, ok := r.URL.Query()["attributes"]
	if ok && len(attr) == 1 && attr[0] == "true" {
		// retrieve image attributes
		w.Write(shared.NewSuccessResponse(&fileAttributes{
			Name:        rec.name,
			Date:        rec.date,
			Author:      rec.auth,
			Checksum:    rec.chk,
			Description: rec.desc,
		}).JSON())
		return
	}

	// retrieve image itself
	rdr, err := i.storage.Get(rec.chk)
	if err != nil {
		panic(err)
	}
	io.Copy(w, rdr)
}

func (i *Images) httpPost(s shared.Session, w http.ResponseWriter, r *http.Request) {

}

func (i *Images) httpPut(s shared.Session, w http.ResponseWriter, r *http.Request) {

}

func (i *Images) httpDelete(s shared.Session, w http.ResponseWriter, r *http.Request) {

}
