default: clean tidy format build test

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
