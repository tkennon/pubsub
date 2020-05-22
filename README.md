# pubsub

[![GoDoc](https://godoc.org/github.com/tkennon/pubsub?status.svg)](https://godoc.org/github.com/tkennon/pubsub)

An in-memory publisher-subscriber framework for single processes. Allows message
passing of any type from many Publishers to many Subscribers.

## Topics

The notion of a "topic" is key to `pubsub`. A topic is a list of strings that
uniquely identifies the type of message being sent. For example
`"button_pressed"` might define the topic of buttons being pressed. Publishers
may publish messages about buttons being pressed, and Subscribers may choose to
listen for them. Subtopics are easily defined by specifying a second, or third,
or any number of further strings in the topic list.

For example
```
h := pubsub.NewHub()
p1 := h.NewPublisher("button_pressed", "accept")
p2 := h.NewPublisher("button_pressed", "closed")
```
defines two publishers that broadcast messages about `accept` buttons being
pressed and `closed` buttons being pressed respectively. To subscribe to these
messages one could
```
s := h.NewSubscriber()
s.Subscribe("button_pressed", "accept") // Receive messages from p1.
s.Subscribe("button_pressed", "closed") // Receive messages from p2.
```
or more simply
```
s := h.NewSubscriber().Subscribe("button_pressed") // Receive messages from both p1 and p2.
```
Subscriptions therefore subscribe to the topic given and implicitly all
subtopics.

Publishers may only ever publish to the topic they are created for. Subscribers
may subscribe to as many topics as they wish, and also unsubscribe from them.
Many Publishers and many Subscribers may be created for the same topic.

## Publishers

Publishers are very simply created with `NewPublisher(...)` which accepts a
variadic number of strings describing the topic.

## Subscribers

Subscribers are created with `NewSubscriber()`, and can subscribe or unsubscribe
to topics through the methods `Subscribe()` and `Unsubscribe()` respectively.
Note that both methods return the Subscriber object allowing these methods to be
chained:
```
s := h.NewSubscriber().
    Subscribe("foo").
    Subscribe("bar", "baz").
    UnSubscribe("bill")
```
It is allowed to subcribe to an empty topic: `s := NewSubscriber().Subscribe()`.
This Subscriber will receive messages from all Publishers since all topics are a
subtopic of something.

## Deadlocking

Calls to `(*Publisher).Publish()` will block until the message has been sent to
all Subscribers. If a Subscriber is not currently waiting on its message channel
this can cause the publishing goroutine to block. Users of this package are
encouraged to always have code listening on Subscriber channels to avoid
deadlock. To add buffering to the Subscriber message channel, use the
`WithCapacity(cap int)` option when creating the Subscriber. This option allows
callers to specifiy the capacity of the Subscriber channel so that up to `cap`
messages may be queued without blocking the publishing goroutine. If a caller
does not require guaranteed message deilvery and can accept some messages being
dropped, then Subscribers may be created with the `WithAllowDrop()` option that
allows the message to be dropped if it cannot be sent on the Subscribers
channel. This ensures the publishing goroutine will never block waiting for a
subscribing goroutine.

## Messages

Publishers and Subscribers accept and return `interface{}` values as messages,
so it is up to the subscribing goroutine to cast the received message to the
concrete type that is expected.

## API stability

`v1.0.0` is tagged and should be considered stable.
