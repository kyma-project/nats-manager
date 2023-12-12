#!/usr/bin/env bash

# This script will check if the tag for the nats-manager image is the one we define as the semantinc release version.

EXPECTED_TAG=${1:-latest}

IMAGE_TO_CHECK=${2:-europe-docker.pkg.dev/kyma-project/prod/nats-manager}
BUMPED_IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

if [[ "$BUMPED_IMAGE_TAG" != "$EXPECTED_TAG" ]]; then
	echo "Tags are not correct: wanted $EXPECTED_TAG but got $BUMPED_IMAGE_TAG"
	exit 1
fi
echo "Tags are correct"
exit 0
