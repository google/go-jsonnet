#!/usr/bin/env bash


if [ "$TRAVIS_PULL_REQUEST" != "false" ]; then
  echo -e "Build Pull Request #$TRAVIS_PULL_REQUEST => Branch [$TRAVIS_BRANCH]"
  $HOME/gopath/bin/goveralls -service=travis-ci
  ./tests.sh --skip-go-test
elif [ "$TRAVIS_PULL_REQUEST" == "false" ] && [ "$TRAVIS_TAG" != "" ]; then
  echo -e 'Build Branch for Release => Branch ['$TRAVIS_BRANCH']  Tag ['$TRAVIS_TAG']'
  env VERSION=$TRAVIS_TAG ./release.sh
else
  echo -e 'Unknown build command for PR? ('$TRAVIS_PULL') Branch ['$TRAVIS_BRANCH']  Tag ['$TRAVIS_TAG']'
fi

