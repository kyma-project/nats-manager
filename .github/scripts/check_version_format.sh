#!/usr/bin/env bash

# Error handling:
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script checks that the RELEASE_TAG does follow the pattern x.y.z where x, y and z are integers.

RELEASE_TAG="$1"

if [[ $RELEASE_TAG =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Version format is valid."
else
	echo "Version format is invalid: ${RELEASE_TAG}"
	echo "Version should follow pattern x.y.z, where x, y and z are integers."
	echo "(e.g. 1.2.3)"
	exit 1
fi
