// Package pubsub provides a Publisher-Subscriber infrastructure for message
// passing within a single process.
package pubsub

import (
	"sync"
)

var (
	// Provide an optional global Hub if clients want one.
	globalHubOnce sync.Once
	globalHub     *Hub
)

// Hub is the hub of the publisher-subscriber model. All messages pass through
// it, and all new Publishers and Subscribers are created from it.
type Hub struct {
	sync.RWMutex
	subsByTopics map[string]*subscriberSet
}

// subscriberSet is a set of Subscribers.
type subscriberSet struct {
	sync.RWMutex
	subs map[*Subscriber]bool
}

// newSubscriberSet returns a new (empty) set of Subscribers.
func newSubscriberSet() *subscriberSet {
	return &subscriberSet{subs: make(map[*Subscriber]bool)}
}

// NewHub returns a new Hub that can be used to pubslish and subscribe to
// messages.
func NewHub() *Hub {
	return &Hub{
		subsByTopics: make(map[string]*subscriberSet),
	}
}

// GlobalHub returns the global instance of a Hub. It is a utility function so
// that if a global Hub is to be used then all client code using it can simply
// call this function rather than have to be passed the Hub explicitly. If not
// called, then no resources are allocated.
func GlobalHub() *Hub {
	globalHubOnce.Do(func() {
		globalHub = NewHub()
	})
	return globalHub
}

// Close closes the Hub. All Subscribers will be closed, and no further messages
// sent. Publishing after a Hub is closed will panic.
func (h *Hub) Close() {
	h.Lock()
	defer h.Unlock()
	for _, subsByTopic := range h.subsByTopics {
		subsByTopic.Lock()
		for sub := range subsByTopic.subs {
			sub.closeAndDrain()
		}
		subsByTopic.subs = nil
		subsByTopic.Unlock()
	}
	h.subsByTopics = nil
}
