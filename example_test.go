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

	// Create some Publishers. Note it does not matter whether Publishers or
	// Subscribers are created first.
	p1 := h.NewPublisher("adele")
	p2 := h.NewPublisher("bob")
	p3 := h.NewPublisher("bob", "marley")
	p4 := h.NewPublisher("cher")

	// Now create some Subscribers.
	s1 := h.NewSubscriber().Subscribe()                                  // Subscribe to everything.
	s2 := h.NewSubscriber().Subscribe("adele").Subscribe("bob")          // Subscribe to p1, p2, and p3.
	s3 := h.NewSubscriber().Subscribe("bob", "marley").Subscribe("cher") // Subscribe to p3 and p4.

	// Start goroutines to listen on publications.
	recv := make(chan string, 10)
	wg := sync.WaitGroup{}
	for i, s := range []*pubsub.Subscriber{s1, s2, s3} {
		wg.Add(1)
		go func(i int, s *pubsub.Subscriber) {
			defer wg.Done()
			for msg := range s.C {
				recv <- fmt.Sprintf("s%d: %s", i+1, msg.(string))
			}
		}(i, s)
	}

	// Publish messages.
	p1.Publish("hello")              // Will be sent to s1 and s2.
	p2.Publish("one love")           // Will be sent to s1 and s2.
	p3.Publish("three little birds") // Will be sent to s1, s2, and s3.
	p4.Publish("believe")            // Will be sent to s1 and s3.

	// Unsubscribe from one topic and see that s2 no longer receives
	// notifications.
	s2.Unsubscribe("bob", "marley")
	p2.Publish("i shot the sherriff") // Will be sent to s1, but not s2.

	// Close all the Subscribers so that the above goroutines return.
	if err := s1.Close(); err != nil {
		panic(err)
	}
	if err := s2.Close(); err != nil {
		panic(err)
	}
	if err := s3.Close(); err != nil {
		panic(err)
	}
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
