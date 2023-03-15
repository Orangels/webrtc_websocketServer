package main

type ChannelObserver interface {
	OnChannelClose(channel *Channel)
}
