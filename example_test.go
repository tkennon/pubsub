package pubsub_test

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/tkennon/pubsub"
)

func Example() {
	h := pubsub.NewHub()

	// Subscribe to everything.
	s1 := h.NewSubscriber(pubsub.WithoutDrop()).Subscribe()
	// Subscribe to p1, p2, and p3.
	s2 := h.NewSubscriber(pubsub.WithoutDrop()).
		Subscribe("bob").
		Subscribe("adele")
	// Subscribe to p3 and p4.
	s3 := h.NewSubscriber(pubsub.WithoutDrop()).
		Subscribe("bob", "marley").
		Subscribe("cher")

	// Start goroutines to listen on publications.
	done := make(chan struct{})
	recv := make(chan string, 10)
	wg := sync.WaitGroup{}
	for i, s := range []*pubsub.Subscriber{s1, s2, s3} {
		wg.Add(1)
		go func(i int, s *pubsub.Subscriber) {
			defer wg.Done()
			for {
				select {
				case msg := <-s.C:
					recv <- fmt.Sprintf("s%d: %s", i+1, msg.(string))
				case <-done:
					return
				}
			}
		}(i, s)
	}

	// Now create some Publishers. Note it does not matter whether Publishers or
	// Subscribers are created first.
	p1 := h.NewPublisher("adele")
	p2 := h.NewPublisher("bob")
	p3 := h.NewPublisher("bob", "marley")
	p4 := h.NewPublisher("cher")

	p1.Publish("hello")
	p2.Publish("one love")
	p3.Publish("three little birds")
	p4.Publish("believe")

	// Unsubscribe from one topic and see that s2 no longer receives
	// notifications.
	s2.Unsubscribe("bob", "marley")
	p2.Publish("i shot the sherriff")

	close(done)
	wg.Wait()
	close(recv)

	// Collect all the results and sort them alphabetically (so that we can
	// deterministically print them).
	var results []string
	for r := range recv {
		results = append(results, r)
	}
	sort.Strings(results)

	fmt.Println(strings.Join(results, "\n"))
	// Output:
	// s1: believe
	// s1: hello
	// s1: i shot the sherriff
	// s1: one love
	// s1: three little birds
	// s2: hello
	// s2: one love
	// s2: three little birds
	// s3: believe
	// s3: three little birds
}
