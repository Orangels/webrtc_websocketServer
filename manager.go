package main

import "encoding/json"

type Manager struct {
	staffers map[string]*Client
	users    map[string]*Client
	sessions map[*Session]bool
	msgQueue chan *Message
}

var inst *Manager

func GetManager() *Manager {
	if inst == nil {
		inst = &Manager{
			staffers: make(map[string]*Client),
			users:    make(map[string]*Client),
			sessions: make(map[*Session]bool),
			msgQueue: make(chan *Message, 16),
		}
	}
	return inst
}

func (mgr *Manager) loop() {
	for {
		select {
		case msg := <-mgr.msgQueue:
			mgr.processMessage(msg)
		}
	}
}

func (mgr *Manager) Run() {
	go mgr.loop()
}

func (mgr *Manager) OnChannelClose(channel *Channel) {
	r := RequestHeader{
		Cmd: CMD_CHANNEL_BROKEN,
	}
	data, _ := json.Marshal(&r)

	msg := NewMessage(channel, data)
	mgr.PostMessage(msg)
}

func (mgr *Manager) PostMessage(msg *Message) {
	mgr.msgQueue <- msg
}

func (mgr *Manager) processMessage(msg *Message) error {
	var rh RequestHeader
	err := json.Unmarshal(msg.data, &rh)
	if err != nil {
		return err
	}

	if rh.Cmd == CMD_STAFFER_LOGIN {
		mgr.onStafferLogin(msg)
	} else if rh.Cmd == CMD_USER_LOGIN {
		mgr.onUserLogin(msg)
	} else if rh.Cmd == CMD_CALL {
		mgr.onCall(msg)
	} else if rh.Cmd == CMD_CHAT {
		mgr.onChat(msg)
	} else if rh.Cmd == CMD_WEBRTC {
		mgr.onWebRTC(msg)
	} else if rh.Cmd == CMD_HANGUP {
		mgr.onHangup(msg)
	} else if rh.Cmd == CMD_CHANNEL_BROKEN {
		mgr.onChannelBroken(msg.channel)
	}

	return nil
}

func (mgr *Manager) onStafferLogin(msg *Message) error {
	var r StafferLoginRequest
	err := json.Unmarshal(msg.data, &r)
	if err != nil {
		return err
	}

	if len(r.Username) == 0 {
		// 用户名为空
		res := StafferLoginResponse{
			Cmd:   CMD_STAFFER_LOGIN_ACK,
			Error: ERROR_USERNAME,
		}
		data, _ := json.Marshal(&res)
		msg.channel.Write(data)
		return nil
	}

	if _, ok := mgr.staffers[r.Username]; ok {
		// 已登录
		res := StafferLoginResponse{
			Cmd:   CMD_STAFFER_LOGIN_ACK,
			Error: ERROR_ALREADY_LOGIN,
		}
		data, _ := json.Marshal(&res)
		msg.channel.Write(data)
		return nil
	}

	client := &Client{
		channel: msg.channel,
		account: r.Username,
		agent:   r.Agent,
		state:   STATE_IDLE,
	}
	mgr.staffers[r.Username] = client

	res := StafferLoginResponse{
		Cmd:   CMD_STAFFER_LOGIN_ACK,
		Error: ERROR_SUCCESS,
	}
	data, _ := json.Marshal(&res)
	msg.channel.Write(data)

	return nil
}

func (mgr *Manager) onUserLogin(msg *Message) error {
	var r UserLoginRequest
	err := json.Unmarshal(msg.data, &r)
	if err != nil {
		return err
	}

	if len(r.Username) == 0 {
		// 用户名为空
		res := StafferLoginResponse{
			Cmd:   CMD_STAFFER_LOGIN_ACK,
			Error: ERROR_USERNAME,
		}
		data, _ := json.Marshal(&res)
		msg.channel.Write(data)
		return nil
	}

	if _, ok := mgr.users[r.Username]; ok {
		// 已登录
		res := StafferLoginResponse{
			Cmd:   CMD_STAFFER_LOGIN_ACK,
			Error: ERROR_ALREADY_LOGIN,
		}
		data, _ := json.Marshal(&res)
		msg.channel.Write(data)
		return nil
	}

	client := &Client{
		channel: msg.channel,
		account: r.Username,
		agent:   r.Agent,
		state:   STATE_IDLE,
	}
	mgr.users[r.Username] = client

	res := UserLoginResponse{
		Cmd:   CMD_USER_LOGIN_ACK,
		Error: ERROR_SUCCESS,
	}
	data, _ := json.Marshal(&res)
	msg.channel.Write(data)

	return nil
}

func (mgr *Manager) onCall(msg *Message) error {
	var r CallRequest
	err := json.Unmarshal(msg.data, &r)
	if err != nil {
		return err
	}

	if _, ok := mgr.users[r.Caller]; !ok {
		// 未登录
		res := CallResponse{
			Cmd:   CMD_CALL_ACK,
			Error: ERROR_USER_NOT_LOGIN,
		}
		data, _ := json.Marshal(&res)
		msg.channel.Write(data)
		return nil
	}
	user := mgr.users[r.Caller]

	var staffer *Client
	if len(r.Callee) == 0 {
		// 没有指定具体的客服，找到一个空闲的客服即可
		for _, v := range mgr.staffers {
			if v.state == STATE_IDLE && v.agent == user.agent {
				staffer = v
				break
			}
		}
		if staffer != nil {
			staffer.state = STATE_BUSY
		} else {
			res := CallResponse{
				Cmd:   CMD_CALL_ACK,
				Error: ERROR_ALL_BUSY,
			}
			data, _ := json.Marshal(&res)
			msg.channel.Write(data)
			return nil
		}
	} else {
		// 具体指定了客服
		if _, ok := mgr.staffers[r.Callee]; !ok {
			// 未登录
			res := CallResponse{
				Cmd:   CMD_CALL_ACK,
				Error: ERROR_STAFFER_NOT_LOGIN,
			}
			data, _ := json.Marshal(&res)
			msg.channel.Write(data)
			return nil
		}
		if mgr.staffers[r.Callee].state == STATE_BUSY || mgr.staffers[r.Callee].agent != user.agent {
			// 客服忙
			res := CallResponse{
				Cmd:   CMD_CALL_ACK,
				Error: ERROR_ALL_BUSY,
			}
			data, _ := json.Marshal(&res)
			msg.channel.Write(data)
			return nil
		}
		staffer = mgr.staffers[r.Callee]
		staffer.state = STATE_BUSY
	}

	session := NewSession(staffer, user)
	mgr.sessions[session] = true

	iceServer := IceServer{
		Urls:       "turn:43.143.227.135:3478",
		Username:   "ghb",
		Credential: "moonshine",
	}

	res := SessionBeginResponse{
		Cmd:        CMD_SESSION_BEGIN,
		Staffer:    staffer.account,
		User:       user.account,
		IceServers: make([]IceServer, 0),
	}
	res.IceServers = append(res.IceServers, iceServer)

	data, _ := json.Marshal(&res)
	staffer.channel.Write(data)
	user.channel.Write(data)

	return nil
}

func (mgr *Manager) onChat(msg *Message) error {
	var r ChatRequest
	err := json.Unmarshal(msg.data, &r)
	if err != nil {
		return err
	}

	for session, _ := range mgr.sessions {
		if session.staffer.channel == msg.channel {
			session.user.channel.Write(msg.data)
			break
		} else if session.user.channel == msg.channel {
			session.staffer.channel.Write(msg.data)
			break
		}
	}

	return nil
}

func (mgr *Manager) onWebRTC(msg *Message) error {
	for session, _ := range mgr.sessions {
		if session.staffer.channel == msg.channel {
			session.user.channel.Write(msg.data)
			break
		} else if session.user.channel == msg.channel {
			session.staffer.channel.Write(msg.data)
			break
		}
	}

	return nil
}

func (mgr *Manager) onHangup(msg *Message) error {
	for session, _ := range mgr.sessions {
		if session.staffer.channel == msg.channel || session.user.channel == msg.channel {
			session.staffer.state = STATE_IDLE

			r := SessionEndResponse{Cmd: CMD_SESSION_END, Error: ERROR_SUCCESS}
			data, _ := json.Marshal(&r)
			session.user.channel.Write(data)
			session.staffer.channel.Write(data)
			delete(mgr.sessions, session)
			break
		}
	}
	return nil
}

func (mgr *Manager) onChannelBroken(channel *Channel) {
	for session, _ := range mgr.sessions {
		if session.staffer.channel == channel {
			// 员工掉线
			r := SessionEndResponse{Cmd: CMD_SESSION_END, Error: ERROR_PEER_OFFLINE}
			data, _ := json.Marshal(&r)
			session.user.channel.Write(data)
			delete(mgr.sessions, session)
			break
		} else if session.user.channel == channel {
			// 用户掉线
			session.staffer.state = STATE_IDLE
			r := SessionEndResponse{Cmd: CMD_SESSION_END, Error: ERROR_PEER_OFFLINE}
			data, _ := json.Marshal(&r)
			session.staffer.channel.Write(data)
			delete(mgr.sessions, session)
			break
		}
	}

	for k, v := range mgr.staffers {
		if v.channel == channel {
			delete(mgr.staffers, k)
			return
		}
	}

	for k, v := range mgr.users {
		if v.channel == channel {
			delete(mgr.users, k)
			return
		}
	}
}
