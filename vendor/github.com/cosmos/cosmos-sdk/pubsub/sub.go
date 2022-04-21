package pubsub

import (
	"fmt"
	"sync"

	"github.com/tendermint/tendermint/libs/log"
)

type ClientID string

type Subscriber struct {
	clientID ClientID
	server   *Server
	handlers map[Topic]Handler
	out      chan Event
	quit     chan struct{}
	wg       sync.WaitGroup
	Logger   log.Logger
}

func (server *Server) NewSubscriber(clientID ClientID, logger log.Logger) (*Subscriber, error) {
	server.mtx.Lock()
	defer server.mtx.Unlock()
	_, ok := server.subscribers[clientID]
	if ok {
		return nil, ErrDuplicateClientID
	}
	sub := &Subscriber{
		clientID: clientID,
		server:   server,
		handlers: make(map[Topic]Handler),
		out:      make(chan Event, 100),
		quit:     make(chan struct{}),
		Logger:   logger,
	}
	server.subscribers[clientID] = make(map[Topic]bool)

	go func() {
		for {
			select {
			case event := <-sub.out:
				sub.eventHandle(event)
				sub.wg.Done()
			case <-sub.quit:
				if sub.Logger != nil {
					sub.Logger.Info(fmt.Sprintf("Subscriber[%s] removed", sub.clientID))
				}
				return
			}
		}
	}()
	return sub, nil
}

func (s *Subscriber) eventHandle(event Event) {
	defer func() {
		if err := recover(); err != nil && s.Logger != nil {
			s.Logger.Error("event handle err: ", err)
		}
	}()
	handler, ok := s.handlers[event.GetTopic()]
	if ok {
		handler(event)
	}
}

func (s *Subscriber) Subscribe(topic Topic, handler Handler) error {
	if handler == nil {
		return ErrNilHandler
	}
	s.server.mtx.RLock()
	subscribers, ok := s.server.subscribers[s.clientID]
	if ok {
		_, ok = subscribers[topic]
	}
	s.server.mtx.RUnlock()
	if ok {
		return ErrAlreadySubscribed
	}

	s.handlers[topic] = handler

	select {
	case s.server.cmds <- cmd{op: sub, topic: topic, subscriber: s, clientID: s.clientID}:
		s.server.mtx.Lock()
		if _, ok := s.server.subscribers[s.clientID]; !ok {
			s.server.subscribers[s.clientID] = make(map[Topic]bool)
		}
		s.server.subscribers[s.clientID][topic] = true
		s.server.mtx.Unlock()
		return nil
	case <-s.server.Quit():
		return nil
	}
}

func (s *Subscriber) Unsubscribe(topic Topic) error {
	s.server.mtx.RLock()
	subscribers, ok := s.server.subscribers[s.clientID]
	if ok {
		_, ok = subscribers[topic]
	}
	s.server.mtx.RUnlock()
	if !ok {
		return ErrSubscriptionNotFound
	}
	select {
	case s.server.cmds <- cmd{op: unsub, clientID: s.clientID, topic: topic}:
		s.server.mtx.Lock()
		delete(s.server.subscribers[s.clientID], topic)
		close(s.quit)
		s.server.mtx.Unlock()
		return nil
	case <-s.server.Quit():
		return nil
	}
}

func (s *Subscriber) UnsubscribeAll() error {
	s.server.mtx.RLock()
	_, ok := s.server.subscribers[s.clientID]
	s.server.mtx.RUnlock()
	if !ok {
		return ErrSubscriptionNotFound
	}
	select {
	case s.server.cmds <- cmd{op: unsub, clientID: s.clientID}:
		s.server.mtx.Lock()
		delete(s.server.subscribers, s.clientID)
		s.server.mtx.RUnlock()
		return nil
	case <-s.server.Quit():
		return nil
	}
}

func (s *Subscriber) Wait() {
	s.server.wg.Wait()
	s.wg.Wait()
}
