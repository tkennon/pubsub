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
	subscribersMtx sync.RWMutex
	subscribers    map[*Subscriber]bool

	childrenMtx sync.RWMutex
	children    map[string]*Hub
}

// NewHub returns a new Hub that can be used to pubslish and subscribe to
// messages.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[*Subscriber]bool),
		children:    make(map[string]*Hub),
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

func (h *Hub) publish(msg interface{}, keys ...string) {
	// Send message to all subscribers at this level.
	h.subscribersMtx.RLock()
	for sub := range h.subscribers {
		sub.send(msg)
	}
	h.subscribersMtx.RUnlock()

	// End the recursion.
	if len(keys) == 0 {
		return
	}

	// Otherwise notify all child hubs.
	h.childrenMtx.RLock()
	child, ok := h.children[keys[0]]
	h.childrenMtx.RUnlock()
	if !ok {
		return
	}
	child.publish(msg, keys[1:]...)
}

func (h *Hub) addSubscriber(s *Subscriber, keys ...string) {
	if len(keys) == 0 {
		h.subscribersMtx.Lock()
		h.subscribers[s] = true
		h.subscribersMtx.Unlock()
		return
	}

	h.childrenMtx.Lock()
	child, ok := h.children[keys[0]]
	if !ok {
		child = NewHub()
		h.children[keys[0]] = child
	}
	h.childrenMtx.Unlock()
	child.addSubscriber(s, keys[1:]...)
}

func (h *Hub) removeSubscriber(s *Subscriber, keys ...string) {
	if len(keys) == 0 {
		h.subscribersMtx.Lock()
		delete(h.subscribers, s)
		h.subscribersMtx.Unlock()
		return
	}

	h.childrenMtx.Lock()
	child, ok := h.children[keys[0]]
	h.childrenMtx.Unlock()
	if ok {
		child.removeSubscriber(s, keys[1:]...)
	}
	if !h.hasSubscribers(keys[1:]...) {
		h.childrenMtx.Lock()
		delete(h.children, keys[0])
		h.childrenMtx.Unlock()
	}
}

func (h *Hub) hasSubscribers(keys ...string) bool {
	if len(keys) == 0 {
		h.subscribersMtx.RLock()
		hasSubs := len(h.subscribers) > 0
		h.subscribersMtx.RUnlock()
		if hasSubs {
			return true
		}

		h.childrenMtx.RLock()
		hasSubs = len(h.children) > 0
		h.childrenMtx.RUnlock()
		return hasSubs
	}

	h.childrenMtx.RLock()
	child, ok := h.children[keys[0]]
	h.childrenMtx.RUnlock()
	return ok && child.hasSubscribers(keys[1:]...)
}

// Close closes the Hub. All Subscribers will be closed, and no further messages
// sent. Publishing after a Hub is closed will panic.
func (h *Hub) Close() error {
	h.subscribersMtx.Lock()
	defer h.subscribersMtx.Unlock()
	for sub := range h.subscribers {
		sub.closeAndDrain()
	}
	h.subscribers = nil

	h.childrenMtx.Lock()
	defer h.childrenMtx.Unlock()
	for _, child := range h.children {
		if err := child.Close(); err != nil {
			// Should never reach this.
			return err
		}
	}
	h.children = nil

	return nil
}
