/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msgbus

import (
	"sync"
)

const (
	defaultMessageBufferSize = 10240
)

// MessageBus provides a pub-sub interface for cross-module communication
type MessageBus interface {
	// A subscriber register on s specific topic.
	// When a message on this topic is published, the subscriber's OnMessage() is called.
	Register(topic Topic, sub Subscriber)
	// Used to publish a message on this message bus to notify subscribers.
	Publish(topic Topic, payload interface{})
	// Used to publish a message on this message bus to notify subscribers.
	// Safe mode, make sure the subscriber's OnMessage() is called with the order of a message on this topic is published.
	PublishSafe(topic Topic, payload interface{})
	// Close the message bus, all publishes are ignored.
	Close()
}

// Subscriber should implement these methods,
type Subscriber interface {
	// When a message with topic A is published on the message bus,
	// all the subscribers's OnMessage() methods of topic A are called.
	OnMessage(*Message)

	// When the message bus is shutting down,
	OnQuit()
}

type messageBusImpl struct {
	mu sync.RWMutex

	// topic -> subscriber list
	// topicMap map[Topic]*list.List
	topicMap   sync.Map
	once       sync.Once
	channelMap sync.Map
	// messageC     chan *Message
	safeMessageC chan *Message
	quitC        chan struct{}
	closed       bool
}

func NewMessageBus() MessageBus {
	return &messageBusImpl{
		mu: sync.RWMutex{},
		// topicMap:     make(map[Topic]*list.List),
		topicMap:   sync.Map{},
		once:       sync.Once{},
		channelMap: sync.Map{},
		// messageC:     make(chan *Message, defaultMessageBufferSize),
		safeMessageC: make(chan *Message, defaultMessageBufferSize),
		quitC:        make(chan struct{}),
		closed:       false,
	}
}

type Message struct {
	Topic   Topic
	Payload interface{}
}

// Register topic for subscriber
func (b *messageBusImpl) Register(topic Topic, sub Subscriber) {
	b.once.Do(func() {
		b.handleMessageLooping()
	})

	// Lock to protect b.closed & slice of subscribers
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	r, _ := b.topicMap.Load(topic)
	if r == nil {
		// The topic has never been registed, make a new array to for subscribers
		b.topicMap.Store(topic, make([]Subscriber, 0))
		// make one channal for each topic
		msgC := make(chan *Message, defaultMessageBufferSize)
		b.channelMap.Store(topic, msgC)
		// start listening channel message for topic
		go func() {
			for {
				select {
				case <-b.quitC:
					return
				case m := <-msgC:
					go b.notify(m, false)
				}
			}
		}()
	}
	subs, _ := b.topicMap.Load(topic)

	s := subs.([]Subscriber)
	// If the same subscriber instance has exist, then return to avoid redundancy
	if isRedundant(s, sub) {
		return
	}
	s = append(s, sub)
	b.topicMap.Store(topic, s)
}

func (b *messageBusImpl) Publish(topic Topic, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}
	// fetch the channel for topic & push message into channel
	c, _ := b.channelMap.Load(topic)
	if c == nil {
		//fmt.Printf("WARN: topic %s is not registed", topic.String()) // stop logging to console
		return
	}
	channel := c.(chan *Message)
	channel <- &Message{Topic: topic, Payload: payload}
}

func (b *messageBusImpl) PublishSafe(topic Topic, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	b.safeMessageC <- &Message{Topic: topic, Payload: payload}
}

func (b *messageBusImpl) Close() {
	// support multiple Close()
	select {
	case <-b.quitC:
	default:
		close(b.quitC)
		b.channelMap.Range(func(_, v interface{}) bool {
			channel := v.(chan *Message)
			close(channel)
			return true
		})
		// close(b.messageC)
		close(b.safeMessageC)
	}
}

func (b *messageBusImpl) handleMessageLooping() {
	go func() {
		for {
			select {
			case m := <-b.safeMessageC:
				b.notify(m, true)
			case <-b.quitC:
				b.mu.Lock()
				// for each top, notify subscribes that message bus is quiting now
				b.topicMap.Range(func(_, v interface{}) bool {
					s := v.([]Subscriber)
					length := len(s)
					for i := 0; i < length; i++ {
						s := s[i].(Subscriber)
						go s.OnQuit()
					}
					return true
				})
				b.closed = true
				b.mu.Unlock()
				return
			}
		}
	}()
}

// notify subscribers when msg comes
func (b *messageBusImpl) notify(m *Message, isSafe bool) {
	if m.Topic <= 0 {
		return
	}

	// fetch the subscribers for topic
	subs, _ := b.topicMap.Load(m.Topic)
	if subs == nil {
		return
	}
	s := subs.([]Subscriber)
	length := len(s)

	// notify each subscriber one by one
	for i := 0; i < length; i++ {
		s := s[i].(Subscriber)
		if isSafe {
			s.OnMessage(m) // notify in order
		} else {
			go s.OnMessage(m)
		}
	}
}

func isRedundant(subs []Subscriber, sub Subscriber) bool {
	for _, s := range subs {
		if s == sub {
			return true
		}
	}
	return false
}

type DefaultSubscriber struct{}

func (DefaultSubscriber) OnMessage(*Message) {
	// just for mock
}
func (DefaultSubscriber) OnQuit() {
	// just for mock
}
