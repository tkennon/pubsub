package pubsub

import (
	"sync"
	"sync/atomic"
)

const (
	// The minimum capacity to allow to async messages being sent on new
	// Subscribers.
	defaultCapacity = 1
)

// Subscriber is the type that receives notifications.
type Subscriber struct {
	// These fields are written once at creation, and then read from thereafter.
	// They require no synchronisation.
	C         chan interface{}
	closeOnce sync.Once
	hub       *Hub
	allowDrop bool

	// These fields require locking.
	topicsMtx sync.RWMutex
	topics    [][]string

	// These fields are written to/read from using atomic synchronisation.
	sent    uint64
	dropped uint64
}

// NewSubscriber returns a new object that will receive messages about a topic
// it is subscribed to. A Subscriber may subscribe to multiple topics. By
// default, if the Subscriber is not ready to receive a message when it is
// published, then the publishing goroutine wait for it to be ready. To add a
// non-zero buffer to the Subscriber's channel, use the WithCapacity()
// SubscriberOption. To allow dropping of messages if a Subscriber channel is
// full then use WithAllowDrop() SubscriberOption.
func (h *Hub) NewSubscriber(opts ...SubscriberOption) *Subscriber {
	s := &Subscriber{
		C:   make(chan interface{}, defaultCapacity),
		hub: h,
	}
	for _, f := range opts {
		f(s)
	}
	return s
}

// SubscriberOption specifies a function that can modify a Subscriber on
// creation.
type SubscriberOption func(*Subscriber)

// WithCapacity sets the capacity of the Subscriber channel. This will be the
// number of messages that can be published to the Subscriber without them being
// read before they are either dropped or Publisher blocks while publishing
// (depending on whether WithAllowDrop() has also been called at Subscriber
// creation).
func WithCapacity(cap int) SubscriberOption {
	return func(s *Subscriber) {
		s.C = make(chan interface{}, cap)
	}
}

// WithAllowDrop allows the Subscriber to drop messages if its channel is
// full. If all Subscribers are created with this option then publishing
// goroutines will never block, however message delivery to the Subscriber is
// not guaranteed.
func WithAllowDrop() SubscriberOption {
	return func(s *Subscriber) {
		s.allowDrop = true
	}
}

// Subscribe adds the topic to the set of topics the Subscriber is subscribed
// to, and will therefore receive messages about. Messages published on this
// topic will be sent to the Subscriber's channel. It is important that this
// channel is read from to prevent data loss, or Publishers blocking on calls to
// Publish(). This method returns the Subscriber for convenience of chaining
// functions, e.g.:
//    s := NewSubscriber().Subscribe("alice").Subscribe("bob")
// creates a new Subscriber that is subscribed to both "alice" and "bob" topics.
// To Subscribe to a subtopic, simply specify multiple strings in a call to
// Subscribe, e.g.:
//    s := NewSubscriber().Subscribe("alice", "bob")
// creates a new Subscriber that is notified about messages published to the
// ["alice", "bob"] subtopic, but not the ["alice"] topic. This feature allows
// fine grained subscriptions to be made.
func (s *Subscriber) Subscribe(keys ...string) *Subscriber {
	s.hub.addSubscriber(s, keys...)
	s.topicsMtx.Lock()
	s.topics = append(s.topics, keys)
	s.topicsMtx.Unlock()
	return s
}

// Unsubscribe removes the topic from the set of topics that the Subscriber is
// subscribed to. If the Subscriber is not subscribed to the topic, then this
// method is a no-op.
func (s *Subscriber) Unsubscribe(keys ...string) *Subscriber {
	s.topicsMtx.Lock()
	for i, topic := range s.topics {
		if equalTopics(topic, keys) {
			s.topics = append(s.topics[:i], s.topics[i+1:]...)
		}
	}
	s.topicsMtx.Unlock()
	s.hub.removeSubscriber(s, keys...)
	return s
}

// Topics returns all the topics the Subscriber is currently subcribed to.
func (s *Subscriber) Topics() [][]string {
	s.topicsMtx.RLock()
	defer s.topicsMtx.RUnlock()
	return s.topics
}

// IsSubscribed returns true if the Subscriber is currently subcribed to the
// topic.
func (s *Subscriber) IsSubscribed(keys ...string) bool {
	s.topicsMtx.RLock()
	defer s.topicsMtx.RUnlock()
	for _, topic := range s.topics {
		if equalTopics(topic, keys) {
			return true
		}
	}
	return false
}

// Close unsubscribes the Subscriber from receiving future messages on all
// topics, and closes and drains the its channel. It should be called when the
// Subscriber is no longer needed. Subscribers will be unable to receive any
// messages after Close() has been called.
func (s *Subscriber) Close() error {
	s.hub.removeSubscriber(s)
	s.closeAndDrain()
	return nil
}

// closeAndDrain closes and drains the Subscriber's channel.
func (s *Subscriber) closeAndDrain() {
	s.closeOnce.Do(func() {
		close(s.C)
	})
}

// NumSent returns a count of the number of messages sent to the Subscriber
// across all topic subscriptions.
func (s *Subscriber) NumSent() uint64 {
	return atomic.LoadUint64(&s.sent)
}

// NumDropped returns a count of the number of messages dropped by the
// Subscriber across all topics because its channel was full at the time the
// message was published.
func (s *Subscriber) NumDropped() uint64 {
	return atomic.LoadUint64(&s.dropped)
}

// send sends the message on the Subscriber's channel.
func (s *Subscriber) send(msg interface{}) {
	if s.allowDrop {
		select {
		case s.C <- msg:
			atomic.AddUint64(&s.sent, 1)
		default:
			atomic.AddUint64(&s.dropped, 1)
		}
	} else {
		s.C <- msg
		atomic.AddUint64(&s.sent, 1)
	}
}

// equalTopics is a helper function to compare two topics (string slices).
func equalTopics(a, b []string) bool {
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
