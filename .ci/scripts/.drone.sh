#!/bin/sh

set -e
set -x

COMMIT="-X main.commit=${GITHUB_SHA}"
VERSION="-X main.version=${DRONE_TAG=latest}"

go build \
    -ldflags "-extldflags \"-static\" $COMMIT $VERSION"   \
	-o release/linux/amd64/drone-autoscaler \
	"github.com/${GITHUB_REPOSITORY}/cmd/drone-autoscaler"