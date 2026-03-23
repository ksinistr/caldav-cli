.PHONY: build test

build:
	go build -o caldav-cli ./cmd/caldav

test:
	go test ./...
