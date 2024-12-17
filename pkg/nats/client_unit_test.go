package nats

import (
	"errors"
	"fmt"
	"testing"

	nmnatsmocks "github.com/kyma-project/nats-manager/pkg/nats/mocks"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

var ErrJetStreamErrorMsg = errors.New("JetStream error")

func Test_StreamExists(t *testing.T) {
	fakeError := ErrJetStreamErrorMsg
	tests := []struct {
		name                 string
		createMockNatsClient func() *natsClient
		streams              []*natsgo.StreamInfo
		expected             bool
		err                  error
	}{
		{
			name: "should no stream exist",
			createMockNatsClient: func() *natsClient {
				mockNatsConn := &nmnatsmocks.Conn{}
				jsCtx := &nmnatsmocks.JetStreamContext{}
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
				mockNatsConn := &nmnatsmocks.Conn{}
				jsCtx := &nmnatsmocks.JetStreamContext{}
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
				mockNatsConn := &nmnatsmocks.Conn{}
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
				require.Equal(t, tt.expected, actual)
			}

			if err != nil && err.Error() != tt.err.Error() {
				require.Equal(t, tt.err, err)
			}
		})
	}
}

func returnStreams() <-chan *natsgo.StreamInfo {
	ch := make(chan *natsgo.StreamInfo)
	go func() {
		defer close(ch)
		ch <- &natsgo.StreamInfo{
			Config: natsgo.StreamConfig{
				Name:     "test-stream",
				Subjects: []string{"test-subject"},
			},
		}
		ch <- &natsgo.StreamInfo{
			Config: natsgo.StreamConfig{
				Name:     "test-stream-2",
				Subjects: []string{"test-subject-2"},
			},
		}
	}()
	return ch
}

func returnEmptyStream() <-chan *natsgo.StreamInfo {
	ch := make(chan *natsgo.StreamInfo)
	go func() {
		defer close(ch)
	}()
	return ch
}
