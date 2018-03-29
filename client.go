package pushpop

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"

	"github.com/google/uuid"
)

// ErrNoMessage is returned when a Message cannot be found. It will be returned
// when you try to find a Message by id and it does not exist, or when you try
// to Pop from an empty topic.
var ErrNoMessage = sql.ErrNoRows

// Client represents a PushPop client with a connection pool to a backend
// PostgreSQL database which will be used for message persistence.
type Client struct {
	db *sql.DB
}

// HandlerFunc is a function that takes a Message and performs some work. In case
// of an error, the message will be reqeueued.
type HandlerFunc func(*Message) error

// NewClient creates a new Client for the given PostgreSQL database connection
// url. It will automatically create or update tables or indices as necessary.
func NewClient(url string) (*Client, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	c := &Client{
		db: db,
	}
	if err := c.setup(); err != nil {
		c.db.Close()
		return nil, err
	}
	return c, nil
}

// Close closes any open database connections and renders the Client unusable.
func (c *Client) Close() error {
	return c.db.Close()
}

// NewMessage creates (but does not enqueue) a new Message with the
// given topic and optional payload.
func (c *Client) NewMessage(topic string, payload ...[]byte) *Message {
	m := &Message{
		Id:    uuid.New().String(),
		Topic: topic,
		c:     c,
	}
	if len(payload) == 1 {
		m.Payload = payload[0]
	}
	return m
}

// FindMessage finds a message by its primary identifier, a string
// representation of a 16-byte UUID. The ErrNoMessage error will be
// returned if the message does not exist.
func (c *Client) FindMessage(id string) (*Message, error) {
	msg := &Message{}
	if err := c.db.QueryRow(sqlFindMessage, id).Scan(&msg.Id, &msg.Topic, &msg.State, &msg.StateTime, &msg.Payload); err != nil {
		return nil, err
	}
	return msg, nil
}

// Pop returns the next available message from the queue of the given topic,
// if one exists. The message will be transitioned to the "pending" state, and
// the receiver becomes responsible for transitioning the state of the message
// using Complete, Discard, or Defer.
func (c *Client) Pop(topic string) (*Message, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msg := &Message{}
	if err := tx.QueryRow(sqlPopMessage, topic).Scan(&msg.Id, &msg.Topic, &msg.State, &msg.StateTime, &msg.Payload); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	msg.c = c
	return msg, nil
}

// Work creates a given number of workers for the supplied topic, managing the
// flow of messages and ensuring orderly shutdown of workers.
func (c *Client) Work(topic string, n int, fn HandlerFunc) {
	// TODO
	panic("not implemented")
}

// setup creates the required table and indices for pushpop to function.
func (c *Client) setup() error {
	ok := false
	if err := c.db.QueryRow(`select exists (select 1 from information_schema.tables where table_name = 'pushpop_messages');`).Scan(&ok); err != nil {
		return err
	}
	if ok {
		return nil
	}
	if _, err := c.db.Exec(sqlCreateMessages); err != nil {
		return err
	}
	if _, err := c.db.Exec(sqlIndexMessagesReady); err != nil {
		return err
	}
	if _, err := c.db.Exec(sqlIndexMessagesPending); err != nil {
		return err
	}
	return nil
}

// push pushes the given message to the queue.
func (c *Client) push(msg *Message, delay time.Duration) error {
	if msg.Id == "" {
		msg.Id = uuid.New().String()
	}
	if msg.c == nil {
		msg.c = c
	}
	msg.State = State_READY
	msg.StateTime = time.Now().Add(delay).UTC()
	_, err := c.db.Exec(sqlPushMessage, msg.Id, msg.Topic, msg.State, msg.StateTime, msg.Payload)
	return err
}
