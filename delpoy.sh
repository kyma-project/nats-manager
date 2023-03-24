docker build -t mfaizan21/nats-manager-dev:22032023 .
docker push mfaizan21/nats-manager-dev:22032023
kubectl delete po -n nats-manager -l control-plane=manager

sleep 5

kubectl logs -n nats-manager -l control-plane=manager -f
