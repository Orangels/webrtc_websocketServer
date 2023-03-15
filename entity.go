package main

const (
	CMD_STAFFER_LOGIN     = "CMD_STAFFER_LOGIN"
	CMD_STAFFER_LOGIN_ACK = "CMD_STAFFER_LOGIN_ACK"
	CMD_USER_LOGIN        = "CMD_USER_LOGIN"
	CMD_USER_LOGIN_ACK    = "CMD_USER_LOGIN_ACK"
	CMD_CALL              = "CMD_CALL"
	CMD_CALL_ACK          = "CMD_CALL_ACK"
	CMD_SESSION_BEGIN     = "CMD_SESSION_BEGIN"
	CMD_SESSION_END       = "CMD_SESSION_END"
	CMD_CHAT              = "CMD_CHAT"
	CMD_WEBRTC            = "CMD_WEBRTC"
	CMD_HANGUP            = "CMD_HANGUP"
	CMD_CHANNEL_BROKEN    = "CMD_CHANNEL_BROKEN"
)

const (
	ERROR_SUCCESS           = 0
	ERROR_USER_NOT_LOGIN    = 1
	ERROR_STAFFER_NOT_LOGIN = 2
	ERROR_ALREADY_LOGIN     = 3
	ERROR_USERNAME          = 4
	ERROR_PASSWORD          = 5
	ERROR_ALL_BUSY          = 6
	ERROR_PEER_OFFLINE      = 7
)

type RequestHeader struct {
	Cmd string `json:"cmd"`
}

type StafferLoginRequest struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Password string `json:"password"`
	Agent    string `json:"agent"`
}

type StafferLoginResponse struct {
	Cmd   string `json:"cmd"`
	Error int    `json:"error"`
}

type UserLoginRequest struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Password string `json:"password"`
	Agent    string `json:"agent"`
}

type UserLoginResponse struct {
	Cmd   string `json:"cmd"`
	Error int    `json:"error"`
}

type CallRequest struct {
	Cmd    string `json:"cmd"`
	Caller string `json:"caller"`
	Callee string `json:"callee"`
}

type CallResponse struct {
	Cmd   string `json:"cmd"`
	Error int    `json:"error"`
}

type IceServer struct {
	Urls       string `json:"urls"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
}

type SessionBeginResponse struct {
	Cmd        string      `json:"cmd"`
	Staffer    string      `json:"staffer"`
	User       string      `json:"user"`
	IceServers []IceServer `json:"iceServers"`
}

type SessionEndResponse struct {
	Cmd   string `json:"cmd"`
	Error int    `json:"error"`
}

type ChatRequest struct {
	Cmd  string `json:"cmd"`
	Text string `json:"text"`
}

type HangupRequest struct {
	Cmd string `json:"cmd"`
}
