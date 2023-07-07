#!/usr/bin/env bash

kubectl -n kyma-system port-forward svc/eventing-nats 4222:4222 &
PID1=$!
kubectl -n kyma-system port-forward eventing-nats-0 8222:8222 &
PID2=$!
kubectl -n kyma-system port-forward eventing-nats-1 8223:8222 &
PID3=$!
kubectl -n kyma-system port-forward eventing-nats-2 8224:8222 &
PID4=$!

go test ./e2e/natsserver/natsserver_test.go --tags=e2e

kill $PID1
kill $PID2
kill $PID3
kill $PID4
