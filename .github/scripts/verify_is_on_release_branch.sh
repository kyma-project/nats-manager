#!/usr/bin/env bash

# This script verifies, that the current branch name starts with 'release-'
#
# Error handling:
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" == release-* ]]; then
	echo "Branch name starts with 'release-'."
else
	echo "Branch name does not start with 'release-': ${CURRENT_BRANCH}"
	exit 1
fi
