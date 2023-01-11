.PHONY: all, build, test, run

all: build

build:
	go build -o bin/sprayproxy main.go

test:
	go test ./...

run:
	go run main.go server --host localhost --port 8080