#!/usr/bin/env bash
set -e

# doc: Run this script in docker to build agent binaries.

cd /workspace
scripts/build/build-using-go.sh
