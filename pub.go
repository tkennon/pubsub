package pubsub

// Publisher is an object that sends messages to all Subscribers that have
// subscribed to the Publisher's topic, or subtopics.
type Publisher struct {
	keys []string
	hub  *Hub
}

// NewPublisher returns a new object that will publish its messages to all
// Subscribers subscribed to the topic (i.e. the list of keys). The Publisher
// may be created for subtopics by specifiying multiple strings. e.g.
// `NewPublisher("alice", "bob")` will publish messages to Subscribers that have
// subscribed to `["alice"]`, or `["alice", "bob"]`, but not `["alice", "bob",
// "charlie"]`.
func (h *Hub) NewPublisher(keys ...string) *Publisher {
	return &Publisher{
		keys: keys,
		hub:  h,
	}
}

// Publish publishes the message to all Subscribers that have subscribed to the
// Publisher's topic. Note that the message is sent "as is" to the Subscribers.
// If the message is a pointer variable then the subscribering goroutines must
// ensure they synchronise any writes. The publishing will occur synchronously,
// and the Subscribers will be notified in an unspecified order. If a Subscriber
// is not ready to receive the message then the message may be dropped (if the
// Subscriber was created with `WithAllowDrop()`) to allow other Subscribers to
// receive the message.
func (p *Publisher) Publish(msg interface{}) {
	p.hub.publish(msg, p.keys...)
}

// HasSubscribers returns true if there are any Subscribers currently subscribed
// to the Publisher's topic, or any subtopics.
func (p *Publisher) HasSubscribers() bool {
	return p.hub.hasSubscribers(p.keys...)
}

// Topic returns the topic that the Publisher publishes about.
func (p *Publisher) Topic() []string {
	return p.keys
}
