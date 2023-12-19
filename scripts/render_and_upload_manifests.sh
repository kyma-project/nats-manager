#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   BASE_REF - name of the tag
#   GITHUB_TOKEN - github token used to upload the template yaml

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

echo "BASE_REF ${BASE_REF}"

MODULE_VERSION=${BASE_REF} make render-manifest

echo "Generated nats-manager.yaml:"
cat nats-manager.yaml

echo "Updating github release with nats-manager.yaml"

echo "Finding release id for: ${BASE_REF}"
CURL_RESPONSE=$(curl -w "%{http_code}" -sL \
	-H "Accept: application/vnd.github+json" \
	-H "Authorization: Bearer $GITHUB_TOKEN" \
	https://api.github.com/repos/kyma-project/nats-manager/releases)
JSON_RESPONSE=$(sed '$ d' <<<"${CURL_RESPONSE}")
HTTP_CODE=$(tail -n1 <<<"${CURL_RESPONSE}")
if [[ "${HTTP_CODE}" != "200" ]]; then
	echo "${JSON_RESPONSE}" && exit 1
fi

RELEASE_ID=$(jq <<<${JSON_RESPONSE} --arg tag "${BASE_REF}" '.[] | select(.tag_name == $ARGS.named.tag) | .id')

if [ -z "${RELEASE_ID}" ]; then
	echo "No release with tag = ${BASE_REF}"
	exit 1
fi

UPLOAD_URL="https://uploads.github.com/repos/kyma-project/nats-manager/releases/${RELEASE_ID}/assets"

uploadFile "nats-manager.yaml" "${UPLOAD_URL}?name=nats-manager.yaml"
uploadFile "config/samples/default.yaml" "${UPLOAD_URL}?name=nats_default_cr.yaml"
