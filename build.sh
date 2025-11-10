#!/bin/bash

# get version information
VERSION=$(cat version)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(go version | awk '{print $3}')

# use `go tool dist list``
export GOOS=linux
export GOARCH=arm
export GOARM=7
export CGO_ENABLED=0
# export CC=arm-linux-gnueabi-gcc

go build -ldflags "\
    -X 'device-go/src.Version=$VERSION' \
    -X 'device-go/src.Commit=$COMMIT' \
    -X 'device-go/src.BuildTime=$BUILD_TIME' \
    -X 'device-go/src.GoVersion=$GO_VERSION'" \
	-o output/device
