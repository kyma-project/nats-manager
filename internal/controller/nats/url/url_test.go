package url

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	// given
	type args struct {
		name      string
		namespace string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return the correct nats url",
			args: args{
				name:      "test-name",
				namespace: "test-namespace",
			},
			want: "nats://test-name.test-namespace.svc.cluster.local:4222",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := Format(tt.args.name, tt.args.namespace)

			// then
			require.Equal(t, tt.want, got)
		})
	}
}
