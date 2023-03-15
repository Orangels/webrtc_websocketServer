package main

type Message struct {
	channel *Channel
	data    []byte
}

func NewMessage(channel *Channel, data []byte) *Message {
	return &Message{channel: channel, data: data}
}

