#!/bin/bash

build() {
	# get version information
	VERSION=$(cat version)
	COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
	BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
	GO_VERSION=$(go version | awk '{print $3}')

	go build -ldflags "\
    -X 'device-go/src.Version=$VERSION' \
    -X 'device-go/src.Commit=$COMMIT' \
    -X 'device-go/src.BuildTime=$BUILD_TIME' \
    -X 'device-go/src.GoVersion=$GO_VERSION'" \
	-o output/device
}

env(){
	# use `go tool dist list``
	export GOOS=linux
	export GOARCH=arm
	export GOARM=7
	export CGO_ENABLED=0
	# export CC=arm-linux-gnueabi-gcc
}

env_ci(){
	# use `go tool dist list``
	export GOOS=linux
	# export GOARCH=arm
	# export GOARM=7
	export CGO_ENABLED=0
	# export CC=arm-linux-gnueabi-gcc
}

if [ $# -eq 0 ]; then
    env
elif [ $# -eq 1 ]; then
    case "$1" in
        "default")
            env
            ;;
        "ci")
            env_ci
            ;;
        *)
            echo "Unknown args"
            exit 1
    esac
else
	echo "Invalid args length"
	exit 1
fi

build
