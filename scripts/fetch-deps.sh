#!/bin/sh

RES=$( { go get github.com/golang/lint/golint; } 2>&1 )

if [[ "${RES}" != *"unrecognized"* ]] && [["${RES}" != *"fatal"* ]]; then
  echo "Success to get to github.com/golang/lint."
else
  echo "Failed to get to github.com/golang/lint, trying golint git repo now..."
  mkdir -p $GOPATH/src/golang.org/x \
    && git clone https://github.com/golang/lint.git $GOPATH/src/golang.org/x/lint \
    && go get -u golang.org/x/lint/golint
fi
