// Package pubsub provides a Publisher-Subscriber framework for message passing
// within a single process.
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
	// children stores a tree of child Hubs, each with subscribers and children
	// of their own.
	childrenMtx sync.RWMutex
	children    map[string]*Hub

	// subscribers holds the set of all subscribers that have subscribed to
	// messages at this 'level'.
	subscribersMtx sync.RWMutex
	subscribers    map[*Subscriber]bool
}

// NewHub returns a new Hub that can be used to publish, and subscribe to,
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

// publish sends the message to all subscribers at the current level before
// recursively walking the branch of the Hub tree described by keys. If the
// branch does not exist then the recursion ends.
//
// For example, if there are two Subscribers; one subscribed to `["alice"]` and
// another subscribed to `["alice", "bob"]`, then this function will publish
// messages for `["alice"]` to _only_ the first Subscriber, and messages to
// `["alice", "bob"]` to both Subscribers (since `["alice"]` captures
// `["alice", "bob"]` as a subtopic).
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

	// Otherwise notify all child Hubs.
	h.childrenMtx.RLock()
	child, ok := h.children[keys[0]]
	h.childrenMtx.RUnlock()
	if !ok {
		return
	}
	child.publish(msg, keys[1:]...)
}

// addSubscriber adds a subscriber to the Hub's children on the topic specified
// by the keys. The Subscriber will be sent all messages that are published to
// its topic and all sub-topics. If necessary, new child Hubs are created if
// they do not exist for the specified topic.
//
// For example, `addSubscriber(s, "alice", "bob")` will store the Subscriber in
// `h.child["alice"].child["bob"].subscribers`. Messages published to
// `["alice", "bob", ...]` will be sent to `s`, but messages published to
// `["alice"]` alone will not.
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

// removeSubscriber removes the subscriber from the topic of the Hub given by
// keys. If no more Subscribers remain for a child Hub then it is deleted.
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

// hasSubscribers checks if the topic specified by keys has any subscribers or
// any child Hubs (which indicate subscribers exist at deeper levels).
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
		sub.close()
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
