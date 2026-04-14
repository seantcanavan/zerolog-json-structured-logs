set shell := ["bash", "-c"]

# Run the full pipeline: clean, tidy, format, build, test
default: clean tidy format build test

# Build the project
build: clean
    go build ./...

# Remove build artifacts
clean:
    rm -rfv ./bin

# Install development dependencies
deps:
    go install github.com/seantcanavan/fresh/v2@latest

# Format source files
format:
    gofmt -s -w -l .

# Run the development server via fresh
run:
    fresh

# Run all tests
test:
    go test ./...

# Run all tests bypassing the cache
test-cached:
    go test ./... -count=1

# Tidy go modules
tidy:
    go mod tidy
