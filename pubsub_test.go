package pubsub_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tkennon/pubsub"
)

func TestOneToOne(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p := h.NewPublisher("topic")
	s := h.NewSubscriber().Subscribe("topic")
	defer s.Close()
	p.Publish("hello")
	assert.Equal(t, "hello", <-s.C)
}

func TestOneToMany(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p := h.NewPublisher("topic")
	s1 := h.NewSubscriber().Subscribe("topic")
	defer s1.Close()
	s2 := h.NewSubscriber().Subscribe("topic")
	defer s2.Close()
	p.Publish("hello")
	assert.Equal(t, "hello", <-s1.C)
	assert.Equal(t, "hello", <-s2.C)
}

func TestManyToOne(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p1 := h.NewPublisher("foo")
	p2 := h.NewPublisher("bar")
	s := h.NewSubscriber().Subscribe("foo", "bar")
	defer s.Close()
	p1.Publish("hello")
	assert.Equal(t, "hello", <-s.C)
	p2.Publish("world")
	assert.Equal(t, "world", <-s.C)
}

func TestManyToMany(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p1 := h.NewPublisher("foo")
	p2 := h.NewPublisher("bar")
	s1 := h.NewSubscriber().Subscribe("foo", "bar")
	defer s1.Close()
	s2 := h.NewSubscriber().Subscribe("foo", "bar")
	defer s2.Close()
	p1.Publish("hello")
	assert.Equal(t, "hello", <-s1.C)
	assert.Equal(t, "hello", <-s2.C)
	p2.Publish("world")
	assert.Equal(t, "world", <-s1.C)
	assert.Equal(t, "world", <-s2.C)
}

func TestSubscribe(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p := h.NewPublisher("topic")
	assert.Equal(t, "topic", p.Topic())
	assert.False(t, p.HasSubscribers())
	s := h.NewSubscriber()
	defer s.Close()
	assert.False(t, s.IsSubscribed("topic"))
	assert.Zero(t, len(s.Topics()))
	s.Subscribe("topic")
	assert.True(t, s.IsSubscribed("topic"))
	assert.Equal(t, []string{"topic"}, s.Topics())
	assert.True(t, p.HasSubscribers())
	s.Unsubscribe("does not exist")
	assert.True(t, s.IsSubscribed("topic"))
	assert.Equal(t, []string{"topic"}, s.Topics())
	assert.True(t, p.HasSubscribers())
	s.Unsubscribe("topic")
	assert.False(t, s.IsSubscribed("topic"))
	assert.Zero(t, len(s.Topics()))
	assert.False(t, p.HasSubscribers())
}

func TestBacklog(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p := h.NewPublisher("topic")
	s := h.NewSubscriber(pubsub.WithCapacity(2)).Subscribe("topic")
	defer s.Close()
	for i := 0; i < 5; i++ {
		p.Publish(i)
	}
	assert.Equal(t, uint64(2), s.NumSent())
	assert.Equal(t, uint64(3), s.NumDropped())
}

func TestBlocking(t *testing.T) {
	h := pubsub.NewHub()
	defer h.Close()
	p := h.NewPublisher("topic")
	s := h.NewSubscriber(pubsub.WithCapacity(1), pubsub.WithoutDrop()).Subscribe("topic")
	defer s.Close()

	p.Publish("hello") // will send async
	go func() {
		p.Publish("world") // will block until "hello" is read on s.
	}()

	// Wait an check that nothing is dropped.
	<-time.After(time.Millisecond)
	assert.Equal(t, uint64(1), s.NumSent())
	assert.Zero(t, s.NumDropped())
	<-s.C
	<-s.C
	<-time.After(time.Millisecond) // a small grace to allow the counter to update.
	assert.Equal(t, uint64(2), s.NumSent())
	assert.Zero(t, s.NumDropped())
}

func TestGlobalHub(t *testing.T) {
	p := pubsub.GlobalHub().NewPublisher("topic")
	s := pubsub.GlobalHub().NewSubscriber().Subscribe("topic")
	defer s.Close()
	p.Publish("hello")
	assert.Equal(t, "hello", <-s.C)
}

func TestLazyClose(t *testing.T) {
	h := pubsub.NewHub()
	p := h.NewPublisher("topic")
	s := h.NewSubscriber().Subscribe("topic")
	p.Publish("foo")
	h.Close()
	// Check that the subscriber channel returns the zero value immediately.
	assert.Zero(t, <-s.C)
	assert.Zero(t, <-s.C)
	assert.Zero(t, <-s.C)
}
