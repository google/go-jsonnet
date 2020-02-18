#!/usr/bin/env bash

run_tests() {
  $GOPATH/bin/goveralls -service=travis-ci
  SKIP_GO_TESTS=1 ./tests.sh
}

run_tests

