#!/bin/bash

set -e -x

go fmt github.com/k14s/ytt/...

GOCACHE=off go test -v `go list ./...|grep -v yaml.v2` "$@"

echo UNIT SUCCESS
