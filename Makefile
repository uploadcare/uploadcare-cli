VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

BINARY  := uploadcare
GOFILES := $(shell find . -name '*.go' -not -path './dist/*' -not -path './bin/*')

.PHONY: build test lint fmt fmt-check clean release-snapshot

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/uploadcare

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	@files="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$files" ]; then \
		echo "gofmt required for:"; \
		echo "$$files"; \
		exit 1; \
	fi

clean:
	rm -rf bin/ dist/

release-snapshot:
	goreleaser release --snapshot --clean
