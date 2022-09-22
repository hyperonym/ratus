package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hyperonym/ratus"
)

// Payload contains information of a crawl task.
type Payload struct {
	URL   string
	Trace struct {
		Referrer string
		Depth    int
	}
}

func main() {

	// Parse command-line flags to get the origin and create a client instance.
	origin := flag.String("origin", "http://127.0.0.1:80", "origin of the Ratus instance")
	flag.Parse()
	client, err := ratus.NewClient(&ratus.ClientOptions{Origin: *origin})
	if err != nil {
		log.Fatal(err)
	}

	// Insert two seed URLs: one of them will be crawled immediately, and the
	// other one will be crawled after 5 seconds.
	if _, err := client.InsertTasks(context.TODO(), []*ratus.Task{
		{
			ID:       "example.com",
			Topic:    "fresh",
			Producer: "mycrawler",
			Payload:  &Payload{URL: "https://example.com"},
		},
		{
			ID:       "foobar.com",
			Topic:    "fresh",
			Producer: "mycrawler",
			Payload:  &Payload{URL: "https://foobar.com"},
			Defer:    "5s",
		},
	}); err != nil {
		log.Fatal(err)
	}

	// Prepare to subscribe to two topics, where tasks found in "fresh" will be
	// moved to the "revisit" topic after their initial execution.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		// Subscribe to the "fresh" topic.
		client.Subscribe(context.TODO(), &ratus.SubscribeOptions{
			Promise: &ratus.Promise{
				Consumer: "mycrawler",
				Timeout:  "30s",
			},
			Topic:       "fresh",
			Concurrency: 5,
		}, func(ctx *ratus.Context, err error) {
			if err != nil {
				log.Println(err)
				return
			}

			// Decode the payload of the task.
			var p Payload
			if err := ctx.Task.Decode(&p); err != nil {
				log.Println(err)
				return
			}

			// Print the current progress and sleep 2 seconds to "crawl" the URL.
			log.Printf("[crawl]    %q (referrer=%q)\n", p.URL, p.Trace.Referrer)
			time.Sleep(2 * time.Second)

			// Create tasks for the "newly discovered" URLs.
			var ts []*ratus.Task
			for i := 0; i < 3; i++ {
				d := Payload{URL: fmt.Sprintf("%s/%d", p.URL, i)}
				d.Trace.Referrer = p.URL
				d.Trace.Depth = p.Trace.Depth + 1
				t := ratus.Task{
					ID:       fmt.Sprintf("%s/%d", ctx.Task.ID, i),
					Topic:    "fresh",
					Producer: "mycrawler",
					Payload:  &d,
				}
				ts = append(ts, &t)
				log.Printf("[discover] %q (depth=%d)\n", d.URL, d.Trace.Depth)
			}
			if _, err := client.InsertTasks(ctx.Context, ts); err != nil {
				log.Println(err)
			}

			// Move the task to the "revisit" topic and crawl it again after
			// 30 seconds.
			ctx.SetTopic("revisit").Retry("30s")
		})
	}()

	go func() {
		defer wg.Done()

		// Subscribe to the "revisit" topic.
		client.Subscribe(context.TODO(), &ratus.SubscribeOptions{
			Promise: &ratus.Promise{
				Consumer: "mycrawler",
				Timeout:  "30s",
			},
			Topic: "revisit",
		}, func(ctx *ratus.Context, err error) {
			if err != nil {
				log.Println(err)
				return
			}

			// Decode the payload of the task.
			var p Payload
			if err := ctx.Task.Decode(&p); err != nil {
				log.Println(err)
				return
			}

			// Print the current progress and sleep 2 seconds to "crawl" the URL.
			log.Printf("[revisit]  %q\n", p.URL)
			time.Sleep(2 * time.Second)

			// Revisit the URL after one minute.
			ctx.Retry("1m")
		})
	}()

	wg.Wait()
}
