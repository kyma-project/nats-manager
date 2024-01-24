#!/bin/bash

## This script requires the following env variables:
# PR_NUMBER (optional, int, If not set then will run for main branch. e.g. 82)
# COMMIT_STATUS_JSON (required, json)
# PROJECT_ROOT (required, string, e.g. "../")

# Example of `COMMIT_STATUS_JSON`
# {
# "url": "https://api.github.com/repos/kyma-project/nats-manager/statuses/12345678765432345676543",
# "avatar_url": "https://avatars.githubusercontent.com/u/123456",
# "id": 123456789,
# "node_id": "SC_kwDOJBeAG123456789",
# "state": "success",
# "description": "Job succeeded.",
# "target_url": "https://status.build.kyma-project.io/view/gs/kyma-prow-logs/post/kyma-project_nats-manager/81/post-nats-module-build/123456789",
# "context": "post-nats-module-build",
# "created_at": "2023-07-18T11:39:23Z",
# "updated_at": "2023-07-18T11:39:23Z"
# }

## check if required ENVs are provided.
if [[ -z "${COMMIT_STATUS_JSON}" ]]; then
  echo "ERROR: COMMIT_STATUS_JSON is not set!"
  exit 1
fi

if [[ -z "${PROJECT_ROOT}" ]]; then
  echo "ERROR: PROJECT_ROOT is not set!"
  exit 1
fi

## define variables
MODULE_TEMPLATE_FILE="${PROJECT_ROOT}/module-template.yaml"

# set links for artifacts of pull requests.
TEMPLATE_FILE_BASE_URL="https://gcsweb.build.kyma-project.io/gcs/kyma-prow-logs/pr-logs/pull/kyma-project_nats-manager"
TEMPLATE_FILE_BASE_URL="https://gcsweb.build.kyma-project.io/gcs/kyma-prow-logs/logs/post-nats-module-build"

## Extract the prow job ID.
echo "Extracting prow job Id from: ${COMMIT_STATUS_JSON}"
TARGET_URL=$(echo ${COMMIT_STATUS_JSON} | jq -r '.target_url')
PROW_JOB_ID=$(echo ${TARGET_URL##*/})
echo "Prow Job ID: ${PROW_JOB_ID}, Link: ${TARGET_URL}"

## Download the module-template.yaml from the build job.
TEMPLATE_FILE_URL="${TEMPLATE_FILE_BASE_URL}/${PROW_JOB_ID}/artifacts/module-template.yaml"
echo "Downloading ${MODULE_TEMPLATE_FILE} from: ${TEMPLATE_FILE_URL}"
curl -s -L -o ${MODULE_TEMPLATE_FILE} ${TEMPLATE_FILE_URL}

## print the module-template.yaml
echo "~~~~~~~~~~~~BEGINNING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
cat ${MODULE_TEMPLATE_FILE}
echo "~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"
