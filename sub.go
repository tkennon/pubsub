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

// NewSubscriber returns a new object that will receive messages about topics it
// subscribes to.
func (h *Hub) NewSubscriber(opts ...SubscriberOption) *Subscriber {
	s := &Subscriber{
		C:         make(chan interface{}, defaultCapacity),
		hub:       h,
		allowDrop: true,
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
// read before they are either dropped or Publishers block while publishing
// (depending on whether WithoutDrop() has been called at Subscriber creation).
func WithCapacity(cap int) SubscriberOption {
	return func(s *Subscriber) {
		s.C = make(chan interface{}, cap)
	}
}

// WithoutDrop does not allow the Subscriber to drop messages if its channel is
// full. Use with care: this could lead to Publishers being blocked.
func WithoutDrop() SubscriberOption {
	return func(s *Subscriber) {
		s.allowDrop = false
	}
}

// Subscribe adds the topics to the set of topics the Subscriber will be
// notified about. Messages published on these topics will be sent to the
// Subscriber's channel. It is important that this channel is read from to
// prevent data loss, or Publishers blocking on calls to Publish(). It returns
// the Subscriber for convenience of chaining functions.
func (s *Subscriber) Subscribe(keys ...string) *Subscriber {
	s.hub.addSubscriber(s, keys...)
	s.topicsMtx.Lock()
	s.topics = append(s.topics, keys)
	s.topicsMtx.Unlock()
	return s
}

// Unsubscribe removes the topics from the set of topics that the Subscriber is
// notified about. If the Subscriber is not subscribed to a topic, then this
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
		for range s.C {
			<-s.C
		}
	})
}

// NumSent returns a count of the number of messages sent to the Subscriber.
func (s *Subscriber) NumSent() uint64 {
	return atomic.LoadUint64(&s.sent)
}

// NumDropped returns a count of the number of messages dropped by the
// Subscriber because its channel was full.
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
