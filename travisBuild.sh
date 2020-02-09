#!/usr/bin/env bash

run_tests() {
  $GOPATH/bin/goveralls -service=travis-ci
  ./tests.sh --skip-go-test
}

run_tests

