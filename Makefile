.PHONY: clean format lint build run test

all: clean format lint build run test

clean:
	rm -rf bin/op-challenger

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
