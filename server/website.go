package server

import (
	"net/http"
	"strings"

	"github.com/alankm/makellos/website"
)

func (s *Server) websiteHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add redirect here if not raft leader
	path := r.URL.Path[1:]
	// if data not found, return 404
	data, err := website.Asset(path)
	if err != nil {
		s.log.Debug("website object not found", "module", "website", "object", path)
		return
	}
	// detect content type and set it
	// if it is text/plain, try to get it by file ending
	ct := http.DetectContentType(data)
	if strings.HasPrefix(ct, "text/plain") {
		pathSplit := strings.Split(path, ".")
		switch pathSplit[len(pathSplit)-1] {
		case "js":
			w.Header().Set("Content-Type", "text/javascript")
		case "css":
			w.Header().Set("Content-Type", "text/css")
		case "json":
			w.Header().Set("Content-Type", "application/json")
		default:
			w.Header().Set("Content-Type", ct)
		}
	} else {
		w.Header().Set("Content-Type", ct)
	}
	w.Write(data)
}
