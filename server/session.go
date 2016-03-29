package server

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alankm/simplicity/server/access"
)

var (
	ResponseAuthentication = NewFailResponse(0, "bad cookie")
	ResponseAccessDenied   = NewFailResponse(0, "access denied")
	ResponseBadMethod      = NewFailResponse(0, "bad HTTP method")
)

type Session struct {
	User access.User
	SU   bool
}

type Rules struct {
	Owner string
	Group string
	Mode  uint16
}

func (s *Session) Mode() uint16 {
	return 0755
}

type ProtectedHandler struct {
	s       *Server
	handler func(*Session, http.ResponseWriter, *http.Request)
}

func (p *ProtectedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func(s *Server, w http.ResponseWriter) {
		r := recover()
		if r != nil {
			w.Write(ResponseVorteilInternal.JSON())
			s.log.Error("unexpected panic")
			fmt.Fprintf(os.Stderr, "%v\n", r)
			fmt.Fprintf(os.Stderr, string(debug.Stack()))
		}
	}(p.s, w)

	// check if logged in
	s := p.HandlerLogin(r)
	if s == nil {
		w.Write(ResponseAuthentication.JSON())
		return
	}

	p.s.log("ACE")

	switch r.Method {
	case "POST":
		path, _ := splitPath(strings.TrimPrefix(r.URL.Path, p.s.servicesVersionString()))
		if !p.s.CanWriteFile(s, path) {
			w.Write(ResponseAccessDenied.JSON())
			return
		}
	case "PUT":
		if !p.s.CanWriteFile(s, strings.TrimPrefix(r.URL.Path, p.s.servicesVersionString())) {
			w.Write(ResponseAccessDenied.JSON())
			return
		}
	case "GET":
		if !p.s.CanReadFile(s, strings.TrimPrefix(r.URL.Path, p.s.servicesVersionString())) {
			w.Write(ResponseAccessDenied.JSON())
			return
		}
	case "DELETE":
		path, _ := splitPath(strings.TrimPrefix(r.URL.Path, p.s.servicesVersionString()))
		if !p.s.CanWriteFile(s, path) {
			w.Write(ResponseAccessDenied.JSON())
			return
		}
	default:
		w.Write(ResponseBadMethod.JSON())
		return
	}
	p.handler(s, w, r)
}

func (p *ProtectedHandler) HandlerLogin(r *http.Request) *Session {
	su := false
	if cookie, err := r.Cookie("vorteil"); err == nil {
		value := make(map[string]string)
		err = p.s.web.cookie.Decode("vorteil", cookie.Value, &value)
		if err == nil {
			username := value["username"]
			password := value["password"]
			user, err := p.s.access.Login(username, password)
			if err != nil {
				fmt.Println("A")
				return nil
			}
			return &Session{
				User: user,
				SU:   su,
			}
		}
		fmt.Println("B")
		fmt.Println(err)
	}
	fmt.Println("C")
	return nil
}

func (s *Session) CanRead(r *Rules) bool {
	if s.SU {
		return true
	}
	if r.Owner == s.User.Name() {
		if r.Mode&(1<<8) > 0 {
			return true
		}
		return false
	}
	for _, grp := range s.User.Groups() {
		if r.Group == grp {
			if r.Mode&(1<<5) > 0 {
				return true
			}
			return false
		}
	}
	if r.Mode&(1<<2) > 0 {
		return true
	}
	return false
}

func (s *Session) CanWrite(r *Rules) bool {
	if s.SU {
		return true
	}
	if r.Owner == s.User.Name() {
		if r.Mode&(1<<7) > 0 {
			return true
		}
		return false
	}
	for _, grp := range s.User.Groups() {
		if r.Group == grp {
			if r.Mode&(1<<4) > 0 {
				return true
			}
			return false
		}
	}
	if r.Mode&(1<<1) > 0 {
		return true
	}
	return false
}

func (s *Session) CanExec(r *Rules) bool {
	if s.SU {
		return true
	}
	if r.Owner == s.User.Name() {
		if r.Mode&(1<<6) > 0 {
			return true
		}
		return false
	}
	for _, grp := range s.User.Groups() {
		if r.Group == grp {
			if r.Mode&(1<<3) > 0 {
				return true
			}
			return false
		}
	}
	if r.Mode&(1<<0) > 0 {
		return true
	}
	return false
}

func splitPath(path string) (string, string) {
	p := strings.TrimSuffix(path, "/")
	if p[len(p)-1] == '/' {
		return "", ""
	}
	i := strings.LastIndex(p, "/")
	return path[:i], path[i+1:]
}

func (s *Server) CanReadFile(u *Session, path string) bool {
	r, err := s.data.getRules(splitPath(path))
	if err != nil {
		return false
	}
	return u.CanRead(r)
}

func (s *Server) CanWriteFile(u *Session, path string) bool {
	r, err := s.data.getRules(splitPath(path))
	if err != nil {
		return false
	}
	return u.CanWrite(r)
}

func (s *Server) CanExecFile(u *Session, path string) bool {
	r, err := s.data.getRules(splitPath(path))
	if err != nil {
		return false
	}
	return u.CanExec(r)
}

func (s *Server) CanRecursiveFile(u *Session, path string) bool {
	// TODO: make recursive operations work.
	return false
}
