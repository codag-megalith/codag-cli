VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
  -X github.com/codag-megalith/codag-cli/cmd.Version=$(VERSION) \
  -X github.com/codag-megalith/codag-cli/cmd.Commit=$(COMMIT) \
  -X github.com/codag-megalith/codag-cli/cmd.BuildDate=$(DATE)

.PHONY: build install test clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/codag .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	rm -rf bin/
