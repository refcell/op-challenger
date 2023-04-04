.PHONY: clean tidy format lint build run test

all: clean tidy format golangci lint build run

ci: clean tidy format golangci lint build test

clean:
	rm -rf bin/op-challenger
	go clean -cache
	go clean -modcache

golangci:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

format:
	gofmt -s -w -l .

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint -e "errors.As" -e "errors.Is"

build:
	env GO111MODULE=on go build -o bin/op-challenger ./cmd

run:
	bin/op-challenger

test:
	go test -v ./...

tidy:
	go mod tidy
