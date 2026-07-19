.PHONY: bootstrap build test smoke install tidy

BIN ?= bin/perplexity-browser-mcp
SMOKE_BIN ?= bin/smoke

bootstrap:
	go mod tidy
	go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium

tidy:
	go mod tidy

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/perplexity-browser-mcp

test:
	go test ./...

smoke: build
	mkdir -p bin
	go build -o $(SMOKE_BIN) ./cmd/smoke
	$(SMOKE_BIN)

install: build
	install -m 755 $(BIN) "$${GOBIN:-$${HOME}/go/bin}/perplexity-browser-mcp"
