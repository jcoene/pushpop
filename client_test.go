package pushpop

import (
	"bytes"
	"database/sql"
	"os"
	"sync"

	"testing"

	"github.com/google/uuid"
)

var postgresUrl string

func init() {
	postgresUrl = os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		postgresUrl = "postgres://postgres:@127.0.0.1:5432/pushpop_test?sslmode=disable"
	}
}

func TestPushpopClient(t *testing.T) {
	mustReset()

	c, err := NewClient(postgresUrl)
	if err != nil {
		t.Fatal(err)
	}

	expectPopNoMessage := func() {
		if _, err := c.Pop("widgets"); err != ErrNoMessage {
			t.Fatalf("expectPopNoMessage: unexpected error: %s", err)
		}
	}

	expectPushOk := func(payload []byte) *Message {
		msg := c.NewMessage("widgets", payload)
		if err := msg.Push(); err != nil {
			t.Fatalf("expectPushOk: unexpected error: %s", err)
		}
		return msg
	}

	expectPopPayload := func(payload []byte) *Message {
		msg, err := c.Pop("widgets")
		if err != nil {
			t.Fatalf("expectPopPayload: unexpected error: %s", err)
		}
		if !bytes.Equal(msg.Payload, payload) {
			t.Fatalf("expectPopPayload: payloads not equal: '%s' != '%s'", payload, msg.Payload)
		}
		return msg
	}

	expectPopMessage := func() *Message {
		msg, err := c.Pop("widgets")
		if err != nil {
			t.Fatalf("expectPopPayload: unexpected error: %s", err)
		}
		return msg
	}

	expectCompleteOk := func(msg *Message) {
		if err := msg.Complete(); err != nil {
			t.Fatalf("expectCompleteOk: unexpected error: %s", err)
		}
	}

	expectSameId := func(m1 *Message, m2 *Message) {
		if m1 == nil {
			t.Fatalf("expectSameId: nil m1")
		}
		if m2 == nil {
			t.Fatalf("expectSameId: nil m2")
		}
		if m1.Id != m2.Id {
			t.Fatalf("expectSameId: %s != %s", m1.Id, m2.Id)
		}
	}

	// Queue is empty, pop nothing
	expectPopNoMessage()

	// Push a message, read it back and compare
	m1a := expectPushOk([]byte("1"))
	m1b := expectPopPayload([]byte("1"))
	expectSameId(m1a, m1b)

	// Queue is empty, pop nothing
	expectPopNoMessage()
	expectPopNoMessage()
	expectPopNoMessage()

	// Complete m1, queue should still be empty
	expectCompleteOk(m1b)
	expectPopNoMessage()

	// Push messages, read back in order, complete too.
	expectPushOk([]byte("2"))
	expectPushOk([]byte("3"))
	expectPushOk([]byte("4"))
	m2 := expectPopPayload([]byte("2"))
	expectCompleteOk(m2)
	m3 := expectPopPayload([]byte("3"))
	expectCompleteOk(m3)
	m4 := expectPopPayload([]byte("4"))
	expectCompleteOk(m4)

	// Queue is empty, pop nothing
	expectPopNoMessage()

	// Put 1000 messages with unique values and read them back, making sure
	// each value received is unique and there's nothing left in the queue
	// afterward.
	wg := sync.WaitGroup{}
	vm := map[string]bool{}
	mu := sync.Mutex{}
	for n := 0; n < 10; n++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				x := []byte(uuid.New().String())
				expectPushOk(x)
			}

			for i := 0; i < 1000; i++ {
				msg := expectPopMessage()
				if err := msg.Complete(); err != nil {
					t.Fatal(err)
				}

				mu.Lock()
				if vm[msg.Id] {
					t.Fatalf("duplicate id: %s", msg.Id)
				}
				vm[msg.Id] = true
				mu.Unlock()
			}

			wg.Done()
		}()
	}
	wg.Wait()
	if len(vm) != 10000 {
		t.Fatalf("expected %d unique ids, got %d", 10000, len(vm))
	}

	// Queue is empty, pop nothing.
	expectPopNoMessage()
}

func mustReset() {
	db, err := sql.Open("postgres", postgresUrl)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err := db.Exec("drop table if exists pushpop_messages"); err != nil {
		panic(err)
	}
}
