#!/bin/bash

version=$(go run ./cmd/execapi -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

echo version=$version

docker build \
    --no-cache \
    -t udhos/execapi:latest \
    -t udhos/execapi:$version \
    -f docker/Dockerfile .
