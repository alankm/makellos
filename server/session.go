package server

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/alankm/simplicity/server/access"
)

var (
	ResponseAuthentication = NewFailResponse(0, "bad cookie")
)

type Session struct {
	User access.User
	SU   bool
}

func (s *Session) Mode() uint16 {
	return 0755
}

func (s *Session) CanRead(r *access.Rules) bool {
	return false
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
