.PHONY: build install clean deps

BINARY := pdf2context
VERSION := 1.0.0

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .

install:
	go install -ldflags="-s -w" .

clean:
	rm -f $(BINARY)

deps:
	go mod tidy

.DEFAULT_GOAL := build
