#!/bin/bash
export GOPATH=$(pwd)/.go
export GOCACHE=$(pwd)/.cache
export GOTMPDIR=$(pwd)/.tmp

go run main.go
