.PHONY: build build-release build-multiplatform install clean test run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY := torgo

LDFLAGS = -s -w -X main.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

build-release:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

build-multiplatform:
	@mkdir -p dist
	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/torgo-linux-amd64 .
	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/torgo-linux-arm64 .
	@echo ""
	@echo "Done! Binaries in dist/:"
	@ls -lh dist/

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -f $(BINARY)
	rm -rf dist

test:
	go test ./...

run:
	go run .
