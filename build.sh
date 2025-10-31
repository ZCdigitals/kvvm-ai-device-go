#!/bin/bash

# go tool dist list
export GOOS=linux
export GOARCH=arm
export GOARM=7
export CGO_ENABLED=0
# export CC=arm-linux-gnueabi-gcc

go build -o output/device
