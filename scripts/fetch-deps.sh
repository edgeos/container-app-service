#!/bin/sh

RES=$( { go get github.com/golang/lint/golint; } 2>&1 )

if [ -z "`echo "${RES}" | grep unrecognized`" ] && [ -z "`echo "${RES}" | grep fatal`" ]; then
  echo "Success to get to github.com/golang/lint."
else
  echo "Failed to get to github.com/golang/lint, trying golint git repo now..."
  mkdir -p $GOPATH/src/golang.org/x
  rm -rf $GOPATH/src/golang.org/x/lint
  git clone https://go.googlesource.com/lint $GOPATH/src/golang.org/x/lint
  go get -u golang.org/x/lint/golint
fi
