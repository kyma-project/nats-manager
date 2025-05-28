#!/bin/bash

# Check for required commands.
for cmd in kubectl nats; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "Error: $cmd not found in PATH."
    exit 1
  }
done

# Function to clean up resources.
cleanup() {
  # Forcefully purge the stream.
  nats stream purge benchstream -f || true
  # Forcefully delete the stream.
  nats stream rm benchstream -f || true

  if [[ -n "$PID" ]]; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
}

# Ensure cleanup runs on script exit or error.
trap cleanup EXIT

# Start port-forwarding in the background.
kubectl port-forward -n kyma-system svc/eventing-nats 4222:4222 &
PID=$!
sleep 1

# Create the stream if it doesn't exist.
nats stream add benchstream --subjects "testsubject" --storage memory --max-msgs 1000 || true

CLUSTER_SIZE=$(kubectl get nats -n kyma-system eventing-nats -ojsonpath='{.spec.cluster.size}')

# Run `bench` with the subject 'testsubject', 5 publishers, 5 subscribers,
# 16-byte size per message, 100 messages using JetStream.
nats bench testsubject --js --replicas="$CLUSTER_SIZE" --pub 5 --sub 5 --size 16 --msgs 100 --no-progress
