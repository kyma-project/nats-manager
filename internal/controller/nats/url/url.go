package url

import (
	"fmt"
)

const (
	format   = "%s://%s.%s.svc.cluster.local:%d"
	protocol = "nats"
	port     = 4222
)

func Format(name, namespace string) string {
	return fmt.Sprintf(format, protocol, name, namespace, port)
}
