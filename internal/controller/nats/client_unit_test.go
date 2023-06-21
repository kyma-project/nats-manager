package nats

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kyma-project/nats-manager/internal/controller/nats/mocks"
	"github.com/nats-io/nats.go"
)

func Test_StreamExists(t *testing.T) {
	fakeError := errors.New("JetStream error")
	tests := []struct {
		name                 string
		createMockNatsClient func() *natsClient
		streams              []*nats.StreamInfo
		expected             bool
		err                  error
	}{
		{
			name: "should no stream exist",
			createMockNatsClient: func() *natsClient {
				mockNatsConn := &mocks.NatsConn{}
				jsCtx := &mocks.JetStreamContext{}
				jsCtx.On("Streams").Return(returnEmptyStream())
				mockNatsConn.On("JetStream").Return(jsCtx, nil)
				return &natsClient{conn: mockNatsConn}
			},
			expected: false,
			err:      nil,
		},
		{
			name: "should streams exist",
			createMockNatsClient: func() *natsClient {
				mockNatsConn := &mocks.NatsConn{}
				jsCtx := &mocks.JetStreamContext{}
				jsCtx.On("Streams").Return(returnStreams())
				mockNatsConn.On("JetStream").Return(jsCtx, nil)
				return &natsClient{conn: mockNatsConn}
			},
			expected: true,
			err:      nil,
		},
		{
			name: "should fail getting JetStream context",
			createMockNatsClient: func() *natsClient {
				mockNatsConn := &mocks.NatsConn{}
				mockNatsConn.On("JetStream").Return(nil, fakeError)
				return &natsClient{conn: mockNatsConn}
			},
			err:      fmt.Errorf("failed to get JetStream: %w", fakeError),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			natsClient := tt.createMockNatsClient()

			actual, err := natsClient.StreamExists()

			if actual != tt.expected {
				t.Errorf("expected %v, but got %v", tt.expected, actual)
			}

			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("expected error %v, but got %v", tt.err, err)
			}
		})
	}
}

func returnStreams() <-chan *nats.StreamInfo {
	ch := make(chan *nats.StreamInfo)
	go func() {
		defer close(ch)
		ch <- &nats.StreamInfo{
			Config: nats.StreamConfig{
				Name:     "test-stream",
				Subjects: []string{"test-subject"},
			},
		}
		ch <- &nats.StreamInfo{
			Config: nats.StreamConfig{
				Name:     "test-stream-2",
				Subjects: []string{"test-subject-2"},
			},
		}
	}()
	return ch
}

func returnEmptyStream() <-chan *nats.StreamInfo {
	ch := make(chan *nats.StreamInfo)
	go func() {
		defer close(ch)
	}()
	return ch
}
