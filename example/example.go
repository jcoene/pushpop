package main

import (
	"log"
	"time"

	"github.com/jcoene/pushpop"
)

// In order to run this example you will need to create the "pushpop_example" database:
//
// > psql -c "create database pushpop_example"
// > go run example.go
//
// Expected output:
//
// > go run example.go
// 2018/03/29 15:42:41 got other message: birds are the best
// 2018/03/29 15:42:41 got other message: cats are the best
// 2018/03/29 15:42:41 got our message: dogs are the best

func main() {
	// Create a new pushpop client
	client, err := pushpop.NewClient("postgres://postgres:@127.0.0.1:5432/pushpop_example?sslmode=disable")
	if err != nil {
		log.Fatalln("unable to connect:", err)
	}
	defer client.Close() // Close any open database connections when we're done

	// Push a few messages to the "mytopic" topic.
	if err := client.NewMessage("mytopic", []byte("birds are the best")).Push(); err != nil {
		log.Fatalln("unable to push:", err)
	}
	if err := client.NewMessage("mytopic", []byte("cats are the best")).Push(); err != nil {
		log.Fatalln("unable to push:", err)
	}
	if err := client.NewMessage("mytopic", []byte("dogs are the best")).Push(); err != nil {
		log.Fatalln("unable to push:", err)
	}

	// We'll iterate over the messages in the queue until we find the one we want.
	for {
		// Pop off the next message
		msg, err := client.Pop("mytopic")
		if err != nil {
			// ErrNoMessage means that no message is available. Let's wait and try again.
			if err == pushpop.ErrNoMessage {
				time.Sleep(1 * time.Second)
				continue
			}

			// An unexpected error occured, not good!
			log.Fatalln("unable to pop:", err)
		}

		// We got a message. Let's verify that this is the message we want.
		if string(msg.Payload) == "dogs are the best" {
			log.Println("got our message:", string(msg.Payload))
			msg.Complete()
			return
		}

		// We got a different message. Let's push it back to the queue and let a cat
		// person deal with it later.
		log.Println("got other message:", string(msg.Payload))
		msg.Defer(5 * time.Second)
	}
}
