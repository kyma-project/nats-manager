#!/bin/bash

TAG=${1:-latest}
IMAGE=${2:-europe-docker.pkg.dev/kyma-project/prod/nats-manager}

docker manifest inspect $IMAGE:$TAG

if [ $? -eq 0 ]; then
	echo "Error: image ${IMAGE}:${TAG} found."
	exit 1
else
	echo "Image not found."
	exit 0
fi
