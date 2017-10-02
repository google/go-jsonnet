#!/bin/bash

set -e

(cd jsonnet; go build)
source tests_path.source
export DISABLE_LIB_TESTS=true
export DISABLE_FMT_TESTS=true
export DISABLE_ERROR_TESTS=true
export JSONNET_BIN="$PWD/jsonnet/jsonnet"
cd "$TESTS_PATH"
./tests.sh
