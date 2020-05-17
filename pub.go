package pubsub

// Publisher is an object that sends messages to all Subscribers that have
// subscribed to the Publisher's topic.
type Publisher struct {
	topic string
	hub   *Hub
}

// NewPublisher returns a new object that will publish its messages to all
// Subscribers subscribed to topic.
func (h *Hub) NewPublisher(topic string) *Publisher {
	return &Publisher{
		topic: topic,
		hub:   h,
	}
}

// Publish publishes the message to all Subscribers that have subscribed to the
// Publisher's topic. Note that the message is sent "as is" to the Subscribers.
// If the message is a pointer variable then the Subscribers must ensure they
// synchronise any writes.
func (p *Publisher) Publish(msg interface{}) {
	p.hub.RLock()
	subsByTopic := p.hub.subsByTopics[p.topic]
	p.hub.RUnlock()
	subsByTopic.RLock()
	defer subsByTopic.RUnlock()
	for sub := range subsByTopic.subs {
		sub.send(msg)
	}
}

// HasSubscribers returns true if there are any Subscribers currently subscribed
// to the Publisher's topic.
func (p *Publisher) HasSubscribers() bool {
	p.hub.RLock()
	defer p.hub.RUnlock()
	subsByTopic, ok := p.hub.subsByTopics[p.topic]
	if !ok {
		return false
	}
	subsByTopic.RLock()
	defer subsByTopic.RUnlock()
	return len(subsByTopic.subs) > 0
}

// Topic returns the topic that the Publisher publishes about.
func (p *Publisher) Topic() string {
	return p.topic
}
