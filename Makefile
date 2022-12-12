TARGET := bin/github-contribs

build:
	CGO_ENABLED=0 go build -o $(TARGET)
.PHONY: build

lint:
	golangci-lint run .
.PHONY: lint
