package shared

import (
	"encoding/json"

	"github.com/sisatech/vorteil/privileges"
)

const (
	CodeInternal = 3000 + iota
	CodeDatabase
)

var (
	Success = &ResponseWrapper{200, "success", nil}

	ResponsePrivilegesInternal = NewFailResponse(1000, "internal error")
	ResponsePrivilegesDatabase = NewFailResponse(1001, "unexpected database error")
	ResponseBadLogin           = NewFailResponse(1002, "invalid username or password")
	ResponseAccessDenied       = NewFailResponse(1003, "access denied")

	ResponseVimagesInternal = NewFailResponse(2000, "internal error")
	ResponseVimagesDatabase = NewFailResponse(2001, "unexpected database error")

	ResponseVorteilInternal = NewFailResponse(3000, "internal error")
	ResponseLeader          = NewFailResponse(3001, "server isn't current raft leader")
	ResponseAuthentication  = NewFailResponse(3002, "not logged into server")
	ResponseBadLoginBody    = NewFailResponse(3003, "bad login body")
	ResponseBadAdminCommand = NewFailResponse(3004, "bad admin command")
	ResponseMissingUsername = NewFailResponse(3005, "missing username header")
	ResponseMissingGroup    = NewFailResponse(3006, "missing group header")
	ResponseMissingPassword = NewFailResponse(3007, "missing password header")

	ResponseCHMOD = NewFailResponse(4000, "chmod placeholder")
	ResponseCHOWN = NewFailResponse(4001, "chown placeholder")
	ResponseCHGRP = NewFailResponse(4002, "chgrp placeholder")

	ResponseUserExists  = NewFailResponse(privileges.CodeUserExists, privileges.ErrUserExists.Error())
	ResponseGroupExists = NewFailResponse(privileges.CodeGroupExists, privileges.ErrGroupExists.Error())
)

type Response interface {
	OK() bool
	JSON() []byte
}

type ResponseWrapper struct {
	Code    int         `json:"status_code"`
	Msg     string      `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
}

type ErrorResponse struct {
	wrapper *ResponseWrapper
	Code    int               `json:"code"`
	Msg     string            `json:"status"`
	Info    map[string]string `json:"info,omitempty"`
}

func NewFailResponse(code int, msg string, info ...string) *ErrorResponse {

	wrapper := new(ResponseWrapper)
	wrapper.Code = 500
	wrapper.Msg = "fail"

	resp := &ErrorResponse{
		wrapper: wrapper,
		Code:    code,
		Msg:     msg,
	}

	wrapper.Payload = resp

	resp.SetInfo(info...)

	return resp

}

type loginResponseStruct struct {
	services []string `json:"services"`
}

func NewSuccessResponse(payload interface{}) *ResponseWrapper {

	wrapper := new(ResponseWrapper)
	wrapper.Code = 200
	wrapper.Msg = "success"
	wrapper.Payload = payload
	return wrapper

}

type groupResponse struct {
	Group []string `json:"groups"`
}

type userResponse struct {
	Users []string `json:"users"`
}

func (r *ErrorResponse) AddInfo(info ...string) {

	if len(info)%2 != 0 {
		panic(CodeInternal)
	}

	for i := 0; i < len(info); i = i + 2 {
		r.Info[info[i]] = info[i+1]
	}

}

func (r *ErrorResponse) SetInfo(info ...string) *ErrorResponse {

	if len(info) == 0 {
		r.Info = nil
		return r
	}

	r.Info = make(map[string]string)
	r.AddInfo(info...)
	return r

}

func (r *ErrorResponse) OK() bool {
	return r.Code == 200
}

func (r *ErrorResponse) JSON() []byte {
	a, _ := json.Marshal(r.wrapper)
	return a
}

func (r *ResponseWrapper) OK() bool {
	return r.Code == 200
}

func (r *ResponseWrapper) JSON() []byte {
	a, _ := json.Marshal(r)
	return a
}

//func

/*

const (
	StatusMsgSuccess  = "success"
	StatusCodeSuccess = 200

	StatusMsgFail  = "error"
	StatusCodeFail = 500
)

var (
	Success = &GeneralResponse{
		Status: StatusMsgSuccess,
		Code:   StatusCodeSuccess,
	}
)

var (
	// Vorteil
	ErrNotLeader    = NewErrResponse(1000, "server isn't leader")
	ErrInternal     = NewErrResponse(1001, "an internal error occurred")
	ErrBadLoginBody = NewErrResponse(1002, "body of the login request was invalid")

	// Privileges
	ErrInvalidLogin          = NewErrResponse(2000, "username or password is invalid")
	ErrAccessDenied          = NewErrResponse(2001, "user does not have permissions to perform this action")

	ErrMissingGroupHeader    = NewErrResponse(2010, "command requires a group header argument")
	ErrMissingUsernameHeader = NewErrResponse(2011, "command requires a username header argument")
	ErrMissingPasswordHeader = NewErrResponse(2012, "command requires a password header argument")

	// Vimages
	ErrNoFileAtTarget = NewErrResponse(3000, "no image or folder exists at the specified target")
)

type Response interface {
	Response(string) []byte
}

type GeneralResponse struct {
	Status  string      `json:"status"`
	Code    int         `json:"status_code"`
	Leader  string      `json:"leader_info,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

func NewGeneralResponse(code int, msg string, payload interface{}) *GeneralResponse {
	r := new(GeneralResponse)
	r.Code = code
	r.Status = msg
	r.Payload = payload
	return r
}

func (r *GeneralResponse) Response(leader string) []byte {
	r.Leader = leader
	a, _ := json.Marshal(r)
	return a
}

type LoginResponse struct {
	Services []string `json:"services"`
	super    GeneralResponse
}

func NewLoginResponse(services []string) *LoginResponse {
	r := new(LoginResponse)
	r.super.Payload = r
	r.super.Status = StatusMsgSuccess
	r.super.Code = StatusCodeSuccess
	r.Services = services
	return r
}

func (r *LoginResponse) Response(leader string) []byte {
	a, _ := json.Marshal(r.super)
	return a
}

type ErrResponse struct {
	Code    int    `json:"error_code"`
	Message string `json:"error_message"`
	super   GeneralResponse
}

func NewErrResponse(code int, msg string) *ErrResponse {
	r := new(ErrResponse)
	r.super.Payload = r
	r.super.Status = StatusMsgFail
	r.super.Code = StatusCodeFail
	r.Code = code
	r.Message = msg
	return r
}

func (r *ErrResponse) Response(leader string) []byte {
	r.super.Leader = leader
	a, _ := json.Marshal(r.super)
	return a
}

type ListUsersResponse struct {
	Users []string `json:"users"`
	super GeneralResponse
}

func NewListUsersResponse(users []string) *ListUsersResponse {
	r := new(ListUsersResponse)
	r.super.Code = StatusCodeSuccess
	r.super.Status = StatusMsgSuccess
	r.super.Payload = r
	r.Users = users
	return r
}

func (r *ListUsersResponse) Response(leader string) []byte {
	r.super.Leader = leader
	a, _ := json.Marshal(r.super)
	return a
}

type ListGroupsResponse struct {
	Groups []string `json:"groups"`
	super  GeneralResponse
}

func NewListGroupsResponse(groups []string) *ListGroupsResponse {
	r := new(ListGroupsResponse)
	r.super.Code = StatusCodeSuccess
	r.super.Status = StatusMsgSuccess
	r.super.Payload = r
	r.Groups = groups
	return r
}

func (r *ListGroupsResponse) Response(leader string) []byte {
	r.super.Leader = leader
	a, _ := json.Marshal(r.super)
	return a
}

type GroupResponse struct {
	Group string `json:"group"`
	super GeneralResponse
}

func NewGroupResponse(group string) *GroupResponse {
	r := new(GroupResponse)
	r.super.Code = StatusCodeSuccess
	r.super.Status = StatusMsgSuccess
	r.super.Payload = r
	r.Group = group
	return r
}

func (r *GroupResponse) Response(leader string) []byte {
	r.super.Leader = leader
	a, _ := json.Marshal(r.super)
	return a
}

type UserInGroupResponse struct {
	Bool  bool `json:"user_is_in_group"`
	super GeneralResponse
}

func NewUserInGroupResponse(b bool) *UserInGroupResponse {
	r := new(UserInGroupResponse)
	r.super.Code = StatusCodeSuccess
	r.super.Status = StatusMsgSuccess
	r.super.Payload = r
	r.Bool = b
	return r
}

func (r *UserInGroupResponse) Response(leader string) []byte {
	r.super.Leader = leader
	a, _ := json.Marshal(r.super)
	return a
}
*/
