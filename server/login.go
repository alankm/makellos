package server

import (
	"encoding/json"
	"net/http"
)

var (
	ResponseBadLogin     = NewFailResponse(3000, "username or password invalid")
	ResponseBadLoginBody = NewFailResponse(3000, "body of the login request was invalid")
)

func (s *Server) handlerLogin(w http.ResponseWriter, r *http.Request) {
	defer func(s *Server, w http.ResponseWriter) {
		r := recover()
		if r != nil {
			w.Write(ResponseVorteilInternal.JSON())
			s.log.Error("unexpected panic during login attempt")
		}
	}(s, w)

	dec := json.NewDecoder(r.Body)
	val := make(map[string]string)
	err := dec.Decode(&val)
	if err != nil {
		w.Write(ResponseBadLoginBody.JSON())
		return
	}

	username := val["username"]
	password := val["password"]
	_, err = s.access.Login(username, password)
	if err != nil {
		w.Write(ResponseBadLogin.JSON())
		return
	}
	enc, err := s.web.cookie.Encode("vorteil", val)
	if err != nil {
		panic(CodeInternal)
	}
	cookie := &http.Cookie{
		Name:  "vorteil",
		Value: enc,
		Path:  "/",
	}
	http.SetCookie(w, cookie)
	s.log.Debug("user successfully logged in")
	w.Write(NewSuccessResponse(nil).JSON())
}
