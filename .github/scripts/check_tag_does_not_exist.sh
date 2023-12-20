#!/usr/bin/env bash

# Error handling:
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script checks that the TAG arg does not exist, already.

TAG="$1"

if [ $(git tag -l $TAG) ]; then
	echo "Error; tag $TAG already exists"
	exit 1
else
	echo "tag $TAG does not exist"
fi
