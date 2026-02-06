BINARY := dctl
BUILD_DIR := bin
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/raviatluri/dctl/cmd.Version=$(VERSION)"

.DEFAULT_GOAL := build

.PHONY: build
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

.PHONY: install
install: build
	@echo "Installing $(BINARY)..."
	@cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

.PHONY: test
test:
	@go test ./pkg/... ./cmd/...

.PHONY: test-unit
test-unit:
	@go test -v ./pkg/...

.PHONY: test-e2e
test-e2e:
	@go test -tags e2e -v -timeout 300s ./e2e/...

.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: vet
vet:
	@go vet ./...
