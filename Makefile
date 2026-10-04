.PHONY: help bootstrap build test smoke install tidy ci hooks-install

.DEFAULT_GOAL := help

BIN ?= bin/perplexity-browser-mcp
SMOKE_BIN ?= bin/smoke
GO ?= GOWORK=off go

help:
	@echo "perplexity-browser — Perplexity Pro research via Playwright MCP"
	@echo ""
	@echo "  make bootstrap   go mod tidy + install Playwright Chromium"
	@echo "  make build       Build $(BIN)"
	@echo "  make test        go test ./..."
	@echo "  make ci          tidy + gofmt + vet + race tests + build"
	@echo "  make smoke       Build + run session status smoke"
	@echo "  make install     Copy $(BIN) to \$${GOBIN:-\$${HOME}/go/bin}"
	@echo "  make tidy        go mod tidy"
	@echo "  make hooks-install  Install Lefthook git hooks (once per clone)"

bootstrap:
	$(GO) mod tidy
	$(GO) run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium

tidy:
	$(GO) mod tidy

build:
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/perplexity-browser-mcp

test:
	$(GO) test ./...

ci:
	@cp go.mod go.mod.bak && cp go.sum go.sum.bak
	$(GO) mod tidy
	@diff -u go.mod.bak go.mod && diff -u go.sum.bak go.sum
	@rm -f go.mod.bak go.sum.bak
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:" && gofmt -l . && exit 1)
	$(GO) vet ./...
	$(GO) test -race -count=1 ./...
	$(GO) build ./...

smoke: build
	mkdir -p bin
	$(GO) build -o $(SMOKE_BIN) ./cmd/smoke
	$(SMOKE_BIN)

install: build
	install -m 755 $(BIN) "$${GOBIN:-$${HOME}/go/bin}/perplexity-browser-mcp"

hooks-install:
	@command -v lefthook >/dev/null 2>&1 || { \
		if command -v brew >/dev/null 2>&1; then brew install lefthook; \
		else go install github.com/evilmartians/lefthook@latest; fi; }
	@command -v lefthook >/dev/null 2>&1 || { echo "lefthook not on PATH; add $$(go env GOPATH)/bin"; exit 1; }
	lefthook install
