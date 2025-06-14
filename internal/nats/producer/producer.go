package producer

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

type natsWrapper struct {
	conn    *nats.Conn
	subject string
}

func New(conn *nats.Conn, subject string) *natsWrapper {
	return &natsWrapper{
		conn:    conn,
		subject: subject,
	}
}

func (n *natsWrapper) Publish(subject string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal NATS message: %w", err)
	}

	if err = n.conn.Publish(subject, jsonData); err != nil {
		return fmt.Errorf("failed to publish NATS message: %w", err)
	}

	return nil
}
