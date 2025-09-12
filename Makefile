SHELL := /usr/bin/env bash -o pipefail -c
.DEFAULT_GOAL := help

# Load variables from .env if present
ifneq (,$(wildcard ./.env))
include .env
export
endif

# ------- Tooling -------
GO               ?= go
GOW              ?= gow
COMPOSE          ?= docker compose
VAULT_SERVICE    ?= vault

# ------- App config -------
APP              ?= vault-raft-backup-operator
SOURCE           ?=
TARGET           ?=
GO_MAIN          ?= ./cmd/operator
GO_PACKAGES      ?= ./...

# ------- Docker -------
IMAGE_REPO       ?= ghcr.io/chapsvision-dev/$(APP)
IMAGE_TAG        ?= dev
IMAGE            ?= $(IMAGE_REPO):$(IMAGE_TAG)

# ------- Helpers -------
define PRINT_HELP_PREAMBLE
Available targets (developer experience):
endef
export PRINT_HELP_PREAMBLE

help: ## Show this help
	@echo "$$PRINT_HELP_PREAMBLE"
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z0-9_.-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---------- Setup ----------
setup: ## Install tool versions via asdf (if available)
	@if command -v asdf >/dev/null 2>&1; then asdf install; else echo "asdf not installed (skipping)"; fi

go-tools: ## Install gow (live-reload) for dev loop
	@$(GO) install github.com/mitranim/gow@latest

# ---------- Vault (docker-compose) ----------
up: ## Start Vault (Raft) in docker compose
	$(COMPOSE) up -d $(VAULT_SERVICE)
	$(COMPOSE) ps

down: ## Stop Vault
	$(COMPOSE) down

remove: ## Stop and remove Vault data
	$(COMPOSE) down -v

logs: ## Tail Vault logs
	$(COMPOSE) logs -f $(VAULT_SERVICE)

init: ## Initialize Vault and auto-unseal (jq runs on host)
	@OUT=$$($(COMPOSE) exec -T $(VAULT_SERVICE) vault operator init -key-shares=1 -key-threshold=1 -format=json); \
	KEY=$$(printf '%s\n' "$$OUT" | jq -r '.unseal_keys_b64[0]'); \
	$(COMPOSE) exec -T $(VAULT_SERVICE) vault operator unseal "$$KEY"; \
	printf '%s\n' "$$OUT" | jq -r '.root_token'

status: ## Show Vault status
	@$(COMPOSE) exec -T $(VAULT_SERVICE) vault status

# ---------- Development ----------
dev: ## Live-reload dev loop (uses gow)
	$(GOW) run $(GO_MAIN)

run: ## Run once (no reload)
	$(GO) run $(GO_MAIN)

build: ## Build binary
	$(GO) build -o bin/$(APP) $(GO_MAIN)

test: ## Run unit tests
	$(GO) test -v $(GO_PACKAGES)

fmt: ## Format code
	$(GO) fmt $(GO_PACKAGES)

vet: ## Static checks with go vet
	$(GO) vet $(GO_PACKAGES)

tidy: ## Sync go.mod/go.sum
	$(GO) mod tidy

lint: vet ## Lint alias (extend with golangci-lint if available)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed (skipping)"

# ---------- Backup / Restore ----------
# For Vault Raft, SOURCE is the Vault base URL (e.g., http://127.0.0.1:8200)
# TARGET is your destination URI (e.g., azure://container/path/snap-<ts>.snap)
backup: ## Run backup: make backup SOURCE=<vault-url> TARGET=<uri>
	$(GOW) run $(GO_MAIN) backup $(SOURCE) $(TARGET)

restore: ## Run restore: make restore SOURCE=<uri> TARGET=<vault-url>
	$(GOW) run $(GO_MAIN) restore $(SOURCE) $(TARGET)

# ---------- Docker ----------
docker-build: ## Build docker image (IMAGE, IMAGE_TAG)
	docker build -t $(IMAGE) .

docker-push: ## Push docker image
	docker push $(IMAGE)

# ---------- Housekeeping ----------
clean: ## Remove build artifacts
	rm -rf bin

.PHONY: help setup go-tools up down logs init status remove \
        dev run build test fmt vet tidy lint backup restore docker-build docker-push clean
