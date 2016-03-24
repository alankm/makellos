package server

import (
	"encoding/json"
)

const (
	CodeInternal = 3000 + iota
	CodeDatabase
)

var (
	Success                 = NewSuccessResponse(nil)
	ResponseVorteilInternal = NewFailResponse(3000, "internal error")
	ResponseLeader          = NewFailResponse(3001, "server isn't current raft leader")
)

type Response interface {
	JSON() []byte
}

type ResponseWrapper struct {
	Code    int         `json:"status_code"`
	Msg     string      `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
}

func NewSuccessResponse(payload interface{}) *ResponseWrapper {
	wrapper := new(ResponseWrapper)
	wrapper.Code = 200
	wrapper.Msg = "success"
	wrapper.Payload = payload
	return wrapper
}

type ErrorResponse struct {
	wrapper *ResponseWrapper
	Code    int               `json:"code"`
	Msg     string            `json:"status"`
	Info    map[string]string `json:"info,omitempty"`
}

func NewFailResponse(code int, msg string) *ErrorResponse {
	wrapper := new(ResponseWrapper)
	wrapper.Code = 500
	wrapper.Msg = "fail"
	resp := &ErrorResponse{
		wrapper: wrapper,
		Code:    code,
		Msg:     msg,
	}
	wrapper.Payload = resp
	return resp
}

func (r *ErrorResponse) JSON() []byte {
	a, _ := json.Marshal(r.wrapper)
	return a
}

func (r *ResponseWrapper) JSON() []byte {
	a, _ := json.Marshal(r)
	return a
}
