package server

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/inconshreveable/log15.v2"
)

type Images struct {
	s       *Server
	log     log15.Logger
	Storage storage
}

func (i *Images) setup(s *Server, log log15.Logger) error {
	log.Debug("HAI")
	i.s = s
	i.log = log
	var err error
	i.Storage, err = initStorage(i.s.conf.Storage)
	if err != nil {
		i.log.Debug("HAI")
		return err
	}
	i.log.Debug("images setup")
}

func (i *Images) setupRoutes(r *mux.Router) {
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.postOW}).Methods("POST").Queries("overwrite", "true")
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.post}).Methods("POST")
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.putAttr}).Methods("PUT").Queries("attributes", "true")
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.putOW}).Methods("PUT")
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.readAttr}).Methods("GET").Queries("attributes", "true")
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.read}).Methods("GET")
	r.Handle("/{path:.*}", &ProtectedHandler{i.s, i.delete}).Methods("DELETE")
}

func (i *Images) load(s *Session, r *http.Request) string {
	// Load body of request and commit to storage.
	file, err := ioutil.TempFile(i.vorteil.config.Data+"/tmp", "img")
	if err != nil {
		i.vorteil.log.Error("vimages couldn't create temporary file")
		panic(CodeInternal)
	}

	sha := sha256.New()
	mw := io.MultiWriter(sha, file)

	_, err = io.Copy(mw, r.Body)
	if err != nil {
		i.vorteil.log.Error("vimages couldn't copy file")
		panic(CodeInternal)
	}

	file.Close()
	checksum := hex.EncodeToString(sha.Sum(nil))

	// Commit loaded file to storage
	var uniq = true
	list, err := i.images.Storage.List()
	if err != nil {
		i.vorteil.log.Error("vimages failed to find files in storage")
		panic(CodeInternal)
	}

	for _, str := range list {
		if str == checksum {
			uniq = false
			break
		}
	}

	err = i.images.Storage.Put(checksum, file.Name())
	if err != nil {
		i.vorteil.log.Error("vimages failed to commit file to storage")
		panic(CodeInternal)
	}

	return checksum
}

type uploadData struct {
	Owner       string
	Group       string
	Mode        uint16
	Target      string
	Author      string
	AuthSet     bool
	Description string
	DescSet     bool
	Time        uint64
	TimeSet     bool
	Checksum    string
}

func (i *Images) makeUploadStruct(s *Session, r *http.Request) *uploadData {
	ret := new(uploadData)
	ret.Owner = s.User.Name()
	ret.Group = s.User.PrimaryGroup()
	ret.Mode = s.Mode()
	ret.Target = strings.TrimPrefix(r.URL.Path, i.s.servicesVersionString())
	if val, ok := r.Header["Author"]; ok {
		ret.AuthSet = true
		ret.Author = val
	}
	if val, ok := r.Header["Description"]; ok {
		ret.AuthSet = true
		ret.Author = val
	}
	if val, ok := r.Header["Time"]; ok {
		ret.AuthSet = true
		ret.Author = val
	}
	ret.Checksum = i.load(s, r)
	if err != nil {
		return nil
	}
	return ret
}

type uploadRet struct {
	Ok     bool
	Delete string
}

func (i *Images) uploadOWFSM(data []byte) interface{} {
	args := new(uploadData)
	ret := new(uploadRet)
	err := decode(data, args)
	if err != nil {
		return ret
	}
	ret.Ok, ret.Delete = i.s.data.imagesOverwrite(args)
	return ret
}

func (i *Images) uploadFSM(data []byte) interface{} {
	args := new(uploadData)
	ret := new(uploadRet)
	err := decode(data, args)
	if err != nil {
		return ret
	}
	ret.Ok, ret.Delete = i.s.data.imagesAdd(args)
	return ret
}

func (i *Images) attributesFSM(data []byte) interface{} {
	args := new(uploadData)
	ret := new(uploadRet)
	err := decode(data, args)
	if err != nil {
		return ret
	}
	ret.Ok = i.s.data.imagesAttr(args)
	return ret
}

func (i *Images) postOW(s *Session, w http.ResponseWriter, r *http.Request) {
	ul := i.makeUploadStruct(s, r)
	ret := i.s.sync("imagesUploadOW", ul)
	r := ret.(uploadRet)
	r := ret.(imgDeleteRet)
	//
	if !r.Ok {
		w.Write(NewFailResponse(0, "").JSON())
		return
	}
	//
	if r.Delete != "" {
		err := i.Storage.Delete(r.Delete)
		if err != nil {
			panic(err)
		}
	}

	w.Write(NewSuccessResponse(nil).JSON())
}

func (i *Images) post(s *Session, w http.ResponseWriter, r *http.Request) {
	ul := i.makeUploadStruct(s, r)
	ret := i.s.sync("imagesUploadOW", ul)
	r := ret.(uploadRet)
	r := ret.(imgDeleteRet)
	//
	if !r.Ok {
		w.Write(NewFailResponse(0, "").JSON())
		return
	}
	w.Write(NewSuccessResponse(nil).JSON())
}

func (i *Images) putAttr(s *Session, w http.ResponseWriter, r *http.Request) {
	ul := i.makeUploadStruct(s, r)
	ret := i.s.sync("imagesAttrOW", ul)
	r := ret.(uploadRet)
	r := ret.(imgDeleteRet)
	//
	if !r.Ok {
		w.Write(NewFailResponse(0, "").JSON())
		return
	}
	w.Write(NewSuccessResponse(nil).JSON())
}

func (i *Images) putOW(s *Session, w http.ResponseWriter, r *http.Request) {
	ul := i.makeUploadStruct(s, r)
	ret := i.s.sync("imagesUploadOW", ul)
	r := ret.(uploadRet)
	r := ret.(imgDeleteRet)
	//
	if !r.Ok {
		w.Write(NewFailResponse(0, "").JSON())
		return
	}
	//
	if r.Delete != "" {
		err := i.Storage.Delete(r.Delete)
		if err != nil {
			panic(err)
		}
	}

	w.Write(NewSuccessResponse(nil).JSON())
}

type attrPL struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Checksum    string `json:"checksum"`
	Date        uint64 `json:"date"`
}

func (i *Images) readAttr(s *Session, w http.ResponseWriter, r *http.Request) {
	pl := new(attrPL)
	path, name := splitPath(strings.TrimPrefix(r.URL.Path, i.s.servicesVersionString()))
	pl.Name = name
	pl.Author, pl.Description, pl.Date, pl.Checksum, _ = i.s.data.imagesGetAttributes(path, name)
	w.Write(NewSuccessResponse(pl).JSON())
}

func (i *Images) read(s *Session, w http.ResponseWriter, r *http.Request) {
	path, name := splitPath(strings.TrimPrefix(r.URL.Path, i.s.servicesVersionString()))
	chk := i.s.data.imagesGetInfo(path, name)
	if chk == "folder" {
		// return list of children
		i.readFolder(s, w, r)
	} else {
		// return image file
		file, err := i.Storage.Get(chk)
		if err != nil {
			panic(err)
		}
		io.Copy(w, file)
		return
	}
}

type folderPL struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type shortPL struct {
	Length int        `json:"length"`
	List   []folderPL `json:"list"`
}

func (i *Images) readFolder(s *Session, w http.ResponseWriter, r *http.Request) {
	var err error
	path, name := splitPath(strings.TrimPrefix(r.URL.Path, i.s.servicesVersionString()))
	off := 0
	if val, ok := r.URL.Query()["offset"]; ok {
		off, err = strconv.ParseInt(val, 10, 32)
		if err != nil {
			off = 0
		}
	}

	len := -1
	if val, ok := r.URL.Query()["length"]; ok {
		len, err = strconv.ParseInt(val, 10, 32)
		if err != nil {
			len = -1
		}
	}

	fltr := ""
	if val, ok := r.URL.Query()["filter"]; ok {
		fltr = val
	}

	srt := ""
	if val, ok := r.URL.Query()["sort"]; ok {
		// TODO: alternate sortings
	}

	ord := "DESC"
	if val, ok := r.URL.Query()["order"]; ok && val == "ASC" {
		ord = val
	}

	pl := i.s.data.imagesGetFolder(path, name, fltr, srt, ord, off, len)
	w.Write(NewSuccessResponse(pl).JSON())

}

type imgDeleteArgs struct {
	Target string
}

type imgDeleteRet struct {
	Ok       bool
	Delete   string
	Response string
}

func (i *Images) delete(s *Session, w http.ResponseWriter, r *http.Request) {
	target := strings.TrimPrefix(r.URL.Path, i.s.servicesVersionString())
	ret := i.s.sync("imagesDelete", &imgDeleteArgs{target})
	r := ret.(imgDeleteRet)
	//
	if !r.Ok {
		w.Write(NewFailResponse(0, "").JSON())
		return
	}
	//
	if r.Delete != "" {
		err := i.Storage.Delete(r.Delete)
		if err != nil {
			panic(err)
		}
	}
	//
	switch r.Response {
	case "SUCCESS":
		w.Write(NewSuccessResponse(nil).JSON())
	case "RECURSION":
		w.Write(NewFailResponse(0, "can't delete without recursion").JSON())
	default:
		panic(err)
	}
}

func (i *Images) deleteFSM(data []byte) interface{} {
	ret := new(imgDeleteRet)
	args := new(imgDeleteArgs)
	err := decode(data, args)
	if err != nil {
		return ret
	}

	path, name := splitPath(args.Target)
	if i.s.data.hasChildren(path, name) {
		ret.Ok = true
		ret.Response = "RECURSION"
		return ret
	}

	if ok, del := i.s.data.imagesDelete(path, name); ok {
		ret.Ok = true
		ret.Delete = del
		ret.Response = "SUCCESS"
		return ret
	}

	return ret

}
