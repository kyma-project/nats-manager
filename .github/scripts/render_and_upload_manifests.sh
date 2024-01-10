#!/usr/bin/env bash

# This script will render the latest manifests and it will uploaded them to the release on github.com.

# Error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

RELEASE_TAG=${1}
MODULE_NAME=${2}
GITHUB_TOKEN=${3}

# uploadFile uploads the rendered assets to the github release.
uploadFile() {
	filePath=${1}
	ghAsset=${2}

	response=$(curl -s -o output.txt -w "%{http_code}" \
		--request POST --data-binary @"$filePath" \
		-H "Authorization: token $GITHUB_TOKEN" \
		-H "Content-Type: text/yaml" \
		$ghAsset)
	if [[ "$response" != "201" ]]; then
		echo "Unable to upload the asset ($filePath): "
		echo "HTTP Status: $response"
		cat output.txt
		exit 1
	else
		echo "$filePath uploaded"
	fi
}

# Render the nats-manager.yaml.
echo "RELEASE_TAG: ${RELEASE_TAG}"
IMG="europe-docker.pkg.dev/kyma-project/prod/${MODULE_NAME}-manager:${RELEASE_TAG}" make render-manifest
echo "Generated ${MODULE_NAME}-manager.yaml:"
cat ${MODULE_NAME}-manager.yaml

# Find the release on github.com via the release tag.
echo -e "\n Updating github release with ${MODULE_NAME}-manager.yaml"
echo "Finding release id for: ${RELEASE_TAG}"
CURL_RESPONSE=$(curl -w "%{http_code}" -sL \
	-H "Accept: application/vnd.github+json" \
	-H "Authorization: Bearer $GITHUB_TOKEN" \
	https://api.github.com/repos/kyma-project/${MODULE_NAME}-manager/releases)
JSON_RESPONSE=$(sed '$ d' <<<"${CURL_RESPONSE}")
HTTP_CODE=$(tail -n1 <<<"${CURL_RESPONSE}")
if [[ "${HTTP_CODE}" != "200" ]]; then
	echo "${JSON_RESPONSE}" && exit 1
fi

# Extract the release id out of the github.com response.
RELEASE_ID=$(jq <<<${JSON_RESPONSE} --arg tag "${RELEASE_TAG}" '.[] | select(.tag_name == $ARGS.named.tag) | .id')
if [ -z "${RELEASE_ID}" ]; then
	echo "No release with tag = ${RELEASE_TAG}"
	exit 1
fi

# With the id of the release we can build the URL to upload the assets.
UPLOAD_URL="https://uploads.github.com/repos/kyma-project/${MODULE_NAME}-manager/releases/${RELEASE_ID}/assets"

# Finally we will upload the manager.yaml and the default.yaml.
uploadFile "nats-manager.yaml" "${UPLOAD_URL}?name=${MODULE_NAME}-manager.yaml"
uploadFile "config/samples/default.yaml" "${UPLOAD_URL}?name=${MODULE_NAME}-default-cr.yaml"
