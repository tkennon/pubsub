package pubsub

// Publisher is an object that sends messages to all Subscribers that have
// subscribed to the Publisher's topic.
type Publisher struct {
	keys []string
	hub  *Hub
}

// NewPublisher returns a new object that will publish its messages to all
// Subscribers subscribed to topic.
func (h *Hub) NewPublisher(keys ...string) *Publisher {
	return &Publisher{
		keys: keys,
		hub:  h,
	}
}

// Publish publishes the message to all Subscribers that have subscribed to the
// Publisher's topic. Note that the message is sent "as is" to the Subscribers.
// If the message is a pointer variable then the Subscribers must ensure they
// synchronise any writes.
func (p *Publisher) Publish(msg interface{}) {
	p.hub.publish(msg, p.keys...)
}

// HasSubscribers returns true if there are any Subscribers currently subscribed
// to the Publisher's topic.
func (p *Publisher) HasSubscribers() bool {
	return p.hub.hasSubscribers(p.keys...)
}

// Topic returns the topic that the Publisher publishes about.
func (p *Publisher) Topic() []string {
	return p.keys
}
