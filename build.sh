#!/bin/bash
set -e
stencil -input Dockerfile | docker build -t gsc-build:latest -
docker run -e GITHUB_TOKEN=$GITHUB_TOKEN -v $GOPATH:/project gsc-build:latest
result=$?
if [ $result -ne 0 ]; then
    echo "failed with code $result"
    exit $result
fi
