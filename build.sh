#!/bin/bash

# go tool dist list
export GOOS=linux
export GOARCH=arm64
export GOARM=8
export CGO_ENABLED=0
# export CC=arm-linux-gnueabi-gcc

go build -o output/device
