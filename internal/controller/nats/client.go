package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type Client interface {
	// initialize NATS connection
	Init() error
	// check if any stream exists in NATS JetStream
	StreamExists() (bool, error)
	// close NATS connection
	Close()
}

//go:generate mockery --name=Client --outpkg=mocks --case=underscore

type natsConfig struct {
	URL           string
	MaxReconnects int
	ReconnectWait int
}

type natsClient struct {
	Config *natsConfig
	conn   NatsConn
}

func NewNatsClient(natsConfig *natsConfig) Client {
	return &natsClient{Config: natsConfig}
}

func (c *natsClient) Init() error {
	if c.conn == nil || c.conn.Status() != nats.CONNECTED {
		natsOptions := []nats.Option{
			nats.RetryOnFailedConnect(true),
			nats.Name("NATS Controller"),
		}
		conn, err := nats.Connect(c.Config.URL, natsOptions...)
		if err != nil || !conn.IsConnected() {
			return fmt.Errorf("failed to connect to NATS server: %w", err)
		}
		c.conn = &natsConn{conn: conn}
	}
	return nil
}

func (c *natsClient) StreamExists() (bool, error) {
	// get JetStream context
	jetStreamCtx, err := c.conn.JetStream()
	if err != nil {
		return false, fmt.Errorf("failed to get JetStream: %w", err)
	}
	// get all streams and check if any exists
	streams := jetStreamCtx.Streams()
	// if it has no streams, it will return false
	_, ok := <-streams
	if !ok {
		return false, nil
	}

	return true, nil
}

func (c *natsClient) Close() {
	c.conn.Close()
}

type NatsConn interface {
	Status() nats.Status
	JetStream() (nats.JetStreamContext, error)
	IsConnected() bool
	Close()
}

type natsConn struct {
	conn *nats.Conn
}

func (c *natsConn) Status() nats.Status {
	return c.conn.Status()
}

func (c *natsConn) JetStream() (nats.JetStreamContext, error) {
	return c.conn.JetStream()
}

func (c *natsConn) IsConnected() bool {
	return c.conn.IsConnected()
}

func (c *natsConn) Close() {
	c.conn.Close()
}
