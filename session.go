package main

type Session struct {
	staffer *Client
	user    *Client
}

func NewSession(staffer *Client, user *Client) *Session {
	return &Session{staffer: staffer, user: user}
}
