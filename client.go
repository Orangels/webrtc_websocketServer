package main

const (
	STATE_UNKNOWN = 0
	STATE_OFFLINE = 1
	STATE_IDLE    = 2
	STATE_BUSY    = 3
)

type Client struct {
	channel *Channel
	account string
	agent   string
	state   int
}
