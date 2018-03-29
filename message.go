package pushpop

import (
	"encoding/json"
	"time"
)

// State represents the status of a message. A message can only be in one state
// at a time.
type State uint16

const (
	// READY: The message is enqueued and ready for processing.
	State_READY State = 0

	// PENDING: The message was popped and is in-flight with a consumer.
	State_PENDING State = 1

	// COMPLETED: The message was popped and successfully processed. At this
	// point the message is safe to delete.
	State_COMPLETED State = 2

	// DISCARDED: The message was popped and otherwise processed. At this point
	// the message is safe to delete.
	State_DISCARDED State = 3
)

// Message represents a single message that can be pushed or popped by a
// Client. While in posession of a client, Messages are stateless.
type Message struct {
	// Id is the UUID of the message, useful for looking up by id.
	Id string

	// Topic is the name of the topic the message has or will be queued in.
	Topic string

	// State represent the state of the Message.
	State State

	// StateTime represents a timestamp to be interpreted based on the State:
	// State_READY: The earliest time at which the message can be popped.
	// State_PENDING: The deadline by which the consumer must check-in, either
	// completing, discarding, deferring, or extending.
	// State_COMPLETED: The time the message was completed.
	// State_DISCARDED: The time the message was discarded.
	StateTime time.Time

	// Payload is a byte slice payload containing the message contents.
	Payload []byte

	c *Client
}

// Push pushes the message to the topic queue, making it available to be popped
// immediately by the next available consumer.
func (m *Message) Push() error {
	return m.c.push(m, 0)
}

// PushDelay pushes the message to the topic queue, making it available to be
// popped after the given duration of time.
func (m *Message) PushDelay(dur time.Duration) error {
	return m.c.push(m, dur)
}

// Complete marks the message as complete, meaning all work has been performed
// and the message can be safely deleted.
func (m *Message) Complete() error {
	return m.transitionAfter(State_COMPLETED, 0)
}

// Discard marks the message as discarded, meaning the work was not performed
// but the message has no further value and can be safely deleted.
func (m *Message) Discard() error {
	return m.transitionAfter(State_DISCARDED, 0)
}

// Defer re-queues the message to be retried at a later time.
func (m *Message) Defer(dur time.Duration) error {
	return m.transitionAfter(State_READY, dur)
}

// Extend extends the deadline of the pending message.
func (m *Message) Extend(dur time.Duration) error {
	return m.transitionAfter(State_PENDING, dur)
}

// EncodePayload encodes the given object as JSON and stores it in the Payload
// field of the Message.
func (m *Message) EncodePayload(v interface{}) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.Payload = buf
	return nil
}

// DecodePayload decodes the JSON payload of the message into the given object.
func (m *Message) DecodePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// transitionAfter updates the Message to the given state, and state_time to
// the current time plus any given duration.
func (m *Message) transitionAfter(state State, dur time.Duration) error {
	m.State = state
	m.StateTime = time.Now().Add(dur).UTC()
	_, err := m.c.db.Exec(sqlTransitionMessage, m.Id, m.State, m.StateTime)
	return err
}
