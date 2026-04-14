APP_NAME    := clashc
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run clean install vet fmt tidy release-snapshot

build:
	go build $(BUILD_FLAGS) -o $(APP_NAME) ./cmd/clashc/

run: build
	./$(APP_NAME)

install:
	go install $(BUILD_FLAGS) ./cmd/clashc/

clean:
	rm -f $(APP_NAME)
	rm -rf dist/

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

# Local goreleaser snapshot (no publish, no tag required)
release-snapshot:
	goreleaser release --snapshot --clean --skip=publish
