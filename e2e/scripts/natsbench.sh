#!/bin/bash

kubectl -n kyma-system port-forward svc/eventing-nats 4222:4222 &
PID=$!
sleep 1

# The following will run `bench` with the subject 'testsubject', 5 publishers 5 subscribers,
# 16 byte size per message, 1000 messages using JetStream.
nats bench testsubject --pub 5 --sub 5 --size 16 --msgs 1000 --js --replicas=3
# Forcefully purge the stream.
nats stream purge benchstream -f
# Forcefully delete the stream.
nats stream rm benchstream -f

# Kill the port-forwarding
kill $PID
