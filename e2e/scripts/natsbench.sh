#!/bin/bash
set -e

kubectl port-forward -n kyma-system svc/eventing-nats 4222:4222 &
PID=$!
sleep 1

CLUSTER_SIZE=$(kubectl get nats -n kyma-system eventing-nats -ojsonpath='{.spec.cluster.size}')
# The following will run `bench` with the subject 'testsubject', 5 publishers 5 subscribers,
# 16 byte size per message, 1000 messages using JetStream.
nats bench testsubject --js --replicas=${CLUSTER_SIZE} --pub 5 --sub 5 --size 16 --msgs 1000 --no-progress
# Forcefully purge the stream.
nats stream purge benchstream -f
# Forcefully delete the stream.
nats stream rm benchstream -f

# Kill the port-forwarding
kill ${PID}
