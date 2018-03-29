# PushPop

[![Build Status](https://secure.travis-ci.org/jcoene/pushpop.png?branch=master)](http://travis-ci.org/jcoene/pushpop) [![GoDoc](https://godoc.org/github.com/jcoene/pushpop?status.svg)](http://godoc.org/github.com/jcoene/pushpop)

PushPop is a simple Postgres backed message queue for Go. It's designed to perform work asynchronously and provides at-least-once delivery. PushPop requires PostgreSQL version 9.5 or higher.

## How it works

1.  `Push` a message to a topic, queueing it up to be popped later.
2.  `Pop` a message from the same topic, taking ownership of it.
3.  Do some work to satisfy the purpose of the message, and:
    * `Complete` it when you're done, marking it as handled and safe for deletion.
    * `Discard` it if the message is not useful, marking it safe for deletion.
    * `Defer` it if you want to try processing again later.
    * `Extend` it if you need more than 5 minutes to process the message.

## Features

* **Probably flexible enough:**
  * Messages are queued by topic, an arbitrary string.
  * Each topic provides a separate message queue.
  * Messages can be looked up by id to retrieve the status of a pending or recently processed message.
* **Probably safe enough:**
  * Messages are stored in PostgreSQL, a durable store.
  * Messages are re-queued in the event that a popped message is not acknowledged, ensuring messages are not lost due to worker restart or death.
* **Probably fast enough:**
  * PushPop is simlpe and fast enough to handle millions of messages a day with relatively low queue depths.
  * It may not fare so well past that. Don't build Uber dispatch on this.
* **Probably scalable enough:**
  * Multiple PushPop clients can be run separately without any coordination.
  * Works great for CPU or IO heavy workloads.
* **Simple**
  * PushPop only has one dependency: PostgreSQL.
  * PushPop is easy to integrate and deploy along side your existing code.
  * PushPop data can live alongside your other tables in the same database if you like.
* **Language agnostic:**
  * PushPop involves no inter-process communication or language specific features.
  * It's possible (and not very difficult) to provide alternate PushPop producer or consumer implementations in other languages.

## Limitations

* **Centralized:**
  * Data lives in a single PostgreSQL instance.
  * You can scale it up, but PushPop has no mechanisms for sharding, failover, or geo-replicating your messages.
* **Transactional:**
  * Because we're using some locking features of PostgreSQL to ensure correct and orderly delivery, there will be an upper bound on the number of messages per second.
  * Good rule of thumb: if you're even considering PushPop, you probably don't need to worry about this.
* **Single Consumer Group:**
  * PushPop provides no facilities to attach multiple consumer groups to a single topic.
  * This might be a feature that can be added in the future.

## Message Ordering and Failure Modes

* Messages are strictly ordered based on wall time plus an optional deferment duration.
* Messages are guaranteed to be delivered at least once, and may be delivered multiple times during certain failure modes. You won't regret building idempotent workers.
* No guarantees are made in the event of PostgreSQL database operating failure.

## TODO

PushPop is still alpha quality software. A few things aren't implemented yet:

* [ ] Add requeueing of pending messages whose deadline has expired
* [ ] Add deletion of completed or discarded messages after a certain grace period.
* [ ] Add worker manager with signal handling.
* [ ] Add observability features (instrumentation, stats api, etc)

## Usage Example

You can find a runnable code example in the _example_ directory. It looks like this:

```go
package main

import (
	"log"
	"time"

	"github.com/jcoene/pushpop"
)

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
```

## License

MIT License, see [LICENSE](https://github.com/jcoene/pushpop/blob/master/LICENSE)
