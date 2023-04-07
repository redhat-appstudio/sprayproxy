.PHONY: all build test run

CONTAINER_ENGINE ?= "podman"
IMAGE ?= "sprayproxy"
TAG ?= "latest"

all: build

build:
	mkdir -p bin
	go build -o bin/sprayproxy main.go

test:
	go test -count=1 ./...

run:
	go run main.go server --host localhost --port 8080

container:
	${CONTAINER_ENGINE} build -t ${IMAGE}:${TAG} .
