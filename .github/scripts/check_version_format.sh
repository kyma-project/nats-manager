#!/usr/bin/env bash

set -eu

# This script checks that the VERSION arg does follow the pattern x.y.z where x, y and z are integers.

TAG="$1"

if [[ $TAG =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Version format is valid"
else
	echo "Version format is invalid"
	exit 1
fi
