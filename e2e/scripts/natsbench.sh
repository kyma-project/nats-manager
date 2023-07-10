#!/bin/bash
set -e

kubectl port-forward -n kyma-system svc/eventing-nats 4222:4222 &
PID=$!
sleep 1

# This will will all the port-forwarding and delete the stream. We need this to be in a function so we can even call it,
# if our tests fails since `set -e` would stop the script in case of an failing test.
function cleanup() {
  kill ${PID}
  # Forcefully purge the stream.
  nats stream purge benchstream -f
  # Forcefully delete the stream.
  nats stream rm benchstream -f
}

# This kills the port-forwards even if the test fails.
trap cleanup ERR
CLUSTER_SIZE=$(kubectl get nats -n kyma-system eventing-nats -ojsonpath='{.spec.cluster.size}')
# The following will run `bench` with the subject 'testsubject', 5 publishers 5 subscribers,
# 16 byte size per message, 1000 messages using JetStream.
nats bench testsubject --js --replicas=${CLUSTER_SIZE} --pub 5 --sub 5 --size 10 --msgs 5 --no-progress

# Kill the port-forwarding.
cleanup
