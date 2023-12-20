#!/usr/bin/env bash

# This script creates a draft release and returns its id .

# Error handling:
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

RELEASE_TAG=$1

REPOSITORY=${REPOSITORY:-kyma-project/nats-manager}
GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
GITHUB_AUTH_HEADER="Authorization: Bearer ${GITHUB_TOKEN}"
CHANGELOG_FILE=$(cat CHANGELOG.md)

# Create the json payload to create a draft release.
JSON_PAYLOAD=$(jq -n \
	--arg tag_name "$RELEASE_TAG" \
	--arg name "$RELEASE_TAG" \
	--arg body "$CHANGELOG_FILE" \
	'{
    "tag_name": $tag_name,
    "name": $name,
    "body": $body,
    "draft": true
  }')

# Send the payload to github to create the draft release. The response contains the id of the release.
CURL_RESPONSE=$(curl -L \
	-X POST \
	-H "Accept: application/vnd.github+json" \
	-H "${GITHUB_AUTH_HEADER}" \
	-H "X-GitHub-Api-Version: 2022-11-28" \
	${GITHUB_URL}/releases \
	-d "$JSON_PAYLOAD")

# Return the draft release id.
echo "$(echo $CURL_RESPONSE | jq -r ".id")"
