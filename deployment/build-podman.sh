#!/bin/bash
set -x

project_root=$(cd ..; pwd)
git_path=github.com/project-codeflare/multi-cluster-app-dispatcher

container_id=$(podman run --privileged --rm -v "$project_root":/go/src/$git_path -d -w /go/src/$git_path/deployment golang:1.16.3-alpine3.13 ./build-inside-container.sh)

podman logs -f $container_id
