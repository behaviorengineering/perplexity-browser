.PHONY: help bootstrap build test smoke smoke-export install tidy ci hooks-install format lint vet

.DEFAULT_GOAL := help

BIN ?= bin/perplexity-browser-mcp
SMOKE_BIN ?= bin/smoke
GO ?= GOWORK=off go

help: ## List local-dev make verbs
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-24s %s\n", $$1, $$2}'
	@printf '\n  Quality: go tool task lint / go tool task test\n'

bootstrap: ## go mod tidy plus Playwright Chromium
	$(GO) mod tidy
	$(GO) run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium

tidy: ## go mod tidy
	$(GO) mod tidy

build: ## Build perplexity-browser-mcp
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/perplexity-browser-mcp

test: ## Run unit tests
	$(GO) test ./...

format: ## Format with gofumpt
	go tool gofumpt -w ./cmd ./internal

lint: ## Run golangci-lint
	go tool golangci-lint run --timeout 5m ./cmd/... ./internal/...

vet: ## Run go vet
	$(GO) vet ./...

smoke: ## Session status smoke (need_login is OK)
	$(GO) build -o $(BIN) ./cmd/perplexity-browser-mcp
	mkdir -p bin
	$(GO) build -o $(SMOKE_BIN) ./cmd/smoke
	$(SMOKE_BIN)

smoke-export: ## Headed export extract smoke
	$(GO) build -o $(SMOKE_BIN) ./cmd/smoke
	$(SMOKE_BIN) export

install: ## Install binary to GOBIN
	$(MAKE) build
	install -m 755 $(BIN) "$${GOBIN:-$${HOME}/go/bin}/perplexity-browser-mcp"

hooks-install: ## Install Lefthook git hooks
	@command -v lefthook >/dev/null 2>&1 || { \
		if command -v brew >/dev/null 2>&1; then brew install lefthook; \
		else go install github.com/evilmartians/lefthook@latest; fi; }
	@command -v lefthook >/dev/null 2>&1 || { echo "lefthook not on PATH; add $$(go env GOPATH)/bin"; exit 1; }
	lefthook install

ci:
	@cp go.mod go.mod.bak && cp go.sum go.sum.bak
	$(GO) mod tidy
	@diff -u go.mod.bak go.mod && diff -u go.sum.bak go.sum
	@rm -f go.mod.bak go.sum.bak
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:" && gofmt -l . && exit 1)
	$(GO) vet ./...
	$(GO) test -race -count=1 ./...
	$(GO) build ./...
