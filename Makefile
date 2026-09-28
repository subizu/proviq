.PHONY: build install test test-coverage clean run-mcp

GOCMD=go
BINARY_NAME=agent-proof
BIN_DIR=bin

build:
	$(GOCMD) build -v -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

install:
	$(GOCMD) install ./cmd/$(BINARY_NAME)

test:
	$(GOCMD) test -v ./...

test-coverage:
	$(GOCMD) test -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

run-mcp: build
	./$(BIN_DIR)/$(BINARY_NAME) mcp

clean:
	rm -rf $(BIN_DIR)
	rm -f *.key *.pub *.out *.html
