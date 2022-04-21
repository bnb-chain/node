package pubsub

type Topic string

type Event interface {
	GetTopic() Topic
}

type Handler func(Event)
