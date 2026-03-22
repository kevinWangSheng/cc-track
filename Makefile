VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/shenghuikevin/cc-track/cmd.Version=$(VERSION)"

.PHONY: build test lint vet clean

build:
	go build $(LDFLAGS) -o cc-track .

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

clean:
	rm -f cc-track

all: lint test build
