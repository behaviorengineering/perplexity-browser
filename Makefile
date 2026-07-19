.PHONY: bootstrap build test smoke install tidy

BIN ?= bin/perplexity-browser-mcp
SMOKE_BIN ?= bin/smoke
GO ?= GOWORK=off go

bootstrap:
	$(GO) mod tidy
	$(GO) run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium

tidy:
	$(GO) mod tidy

build:
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/perplexity-browser-mcp

test:
	$(GO) test ./...

smoke: build
	mkdir -p bin
	$(GO) build -o $(SMOKE_BIN) ./cmd/smoke
	$(SMOKE_BIN)

install: build
	install -m 755 $(BIN) "$${GOBIN:-$${HOME}/go/bin}/perplexity-browser-mcp"
