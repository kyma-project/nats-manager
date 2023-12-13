#!/bin/bash

# This script checks with a timeout, if a defined image with a defined can be found.

TIMEOUT=${1:-600}
INTERVAL=${2:-20}
TAG=${3:-latest}
IMAGE=${4:-europe-docker.pkg.dev/kyma-project/prod/nats-manager}

start_time=$(date +%s)
end_time=$((start_time + TIMEOUT))

while true; do
	docker manifest inspect $IMAGE:$TAG
	# Check the exit status
	if [ $? -eq 0 ]; then
		echo "Image found."
		exit 0
	else
		echo "Image not found, yet."
	fi

	current_time=$(date +%s)

	# Check if the TIMEOUT has been reached
	if [ $current_time -ge $end_time ]; then
		echo "TIMEOUT reached. Image was not found in time."
		exit 1
	fi

	sleep $INTERVAL
done
