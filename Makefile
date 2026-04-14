SHELL := /bin/bash

.PHONY: build clean deploy-all deploy-staging deploy-production format pre-deploy run test test-cached

all: clean tidy format build test

build: clean
	go build ./...

clean:
	rm -rfv ./bin

deps:
	go install github.com/seantcanavan/fresh/v2@latest

format:
	gofmt -s -w -l .

run:
	fresh

test:
	go test ./...

test-cached:
	go test ./... -count=1

tidy:
	go mod tidy
