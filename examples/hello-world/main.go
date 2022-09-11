package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/hyperonym/ratus"
)

func main() {

	// Parse command-line flags to get the origin and create a client instance.
	origin := flag.String("origin", "http://127.0.0.1:80", "origin of the Ratus instance")
	flag.Parse()
	client, err := ratus.NewClient(&ratus.ClientOptions{Origin: *origin})
	if err != nil {
		log.Fatal(err)
	}

	// Insert a new task to the "example" topic.
	// This will return an error if a task with the same ID already exists.
	// Use client.UpsertTask or client.UpsertTasks instead to upsert tasks.
	if _, err := client.InsertTask(context.TODO(), &ratus.Task{
		ID:      "1",
		Topic:   "example",
		Payload: "hello world",
	}); err != nil {
		log.Fatal(err)
	}

	// Claim and execute the next available task in the "example" topic.
	// An error wrapping ratus.ErrNotFound will be returned if the topic is empty,
	// or if no task in the topic has reached its scheduled time of execution.
	ctx, err := client.Poll(context.TODO(), "example", &ratus.Promise{Timeout: "30s"})
	if err != nil {
		log.Fatal(err)
	}

	// Print the payload of the acquired task.
	// In real-world applications, now its time to execute the task.
	fmt.Println(ctx.Task.Payload)

	// Remember to acknowledge Ratus that the task is now completed.
	if err := ctx.Commit(); err != nil {
		log.Fatal(err)
	}
}
