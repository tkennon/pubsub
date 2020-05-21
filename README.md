# pubsub

An in-memory publisher-subscriber framework for single processes. Allows message
passing of any type from many Publishers to many Subscribers.

## Topics

The notion of a "topic" is key to `pubsub`. A topic is a list of strings that
uniquely identifies a the type of message being sent. For example
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
s.Subscribe("button_pressed", "accept") // receive messages from p1
s.Subscribe("button_pressed", "closed") // receive messages from p2
```
or more simply
```
s := h.NewSubscriber().Subscribe("button_pressed") // receive messages from both p1 and p2
```
Subscriptions therefore subscribe to the topic given and implicitly all subtopics.

Publishers may only ever publish to the topic they are created for. Subscribers
may subscribe to as many topics as they wish, and also unsubscribe from them.
Many Publishers and many Subscribers may be created for the same topic.

## Publishers

Publishers are very simply created with `NewPublisher(...)` which accepts a
variadic number of strings. Note that the Publisher created with no topic
(`NewPublisher()`) will publish to every Subscriber, since every Subscriber
subcribes to a subtopic of something.

## Subscribers

Subscribers are created with `NewSubscriber()`, and can subscribe or unsubscrive
to topics through the methods `Subscribe()` and `Unsubscribe()`. Note that both
methods return the Subscriber object so that these methods may be chained:
```
s := h.NewSubscriber().
    Subscribe("foo").
    Subscribe("bar", "baz").
    UnSubscribe("bill")
```

## Deadlocking

Calls to `(*Publisher).Publish()` will block until the message has been sent to
all Subscribers. If a Subscriber is not currently waiting on its message channel
this can cause the publishing goroutine to block. As such, by default, if a
message cannot be sent to a Subscriber it is dropped. To add buffering to the
Subscriber message channel, use the `WithCapacity()` option when creating the
Subscriber. If you wish to never drop a message then use the `WithoutDrop()`
option. Users of this option must ensure that their code is always listening on
the Subscriber's message channel.
