#!/usr/bin/env bash
set -e

# for our tests we need to port-forward the Pods.
kubectl -n kyma-system port-forward eventing-nats-0 8222:8222 &
PID1=$!
kubectl -n kyma-system port-forward eventing-nats-1 8223:8222 &
PID2=$!
kubectl -n kyma-system port-forward eventing-nats-2 8224:8222 &
PID3=$!

# This will will all the port-forwarding. We need this to be in a function so we can even call it, if our tests fails
# since `set -e` would stop the script in case of an failing test.
function kill_port_forward() {
  kill ${PID1}
  kill ${PID2}
  kill ${PID3}
}
# This kills the port-forwards even if the test fails.
trap kill_port_forward ERR

go test ./e2e/natsserver/natsserver_test.go --tags=e2e

kill_port_forward
