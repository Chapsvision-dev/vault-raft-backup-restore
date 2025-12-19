# Vault Raft Backup & Restore Operator

[![CI](https://github.com/Chapsvision-dev/vault-raft-backup-restore/actions/workflows/ci.yml/badge.svg)](https://github.com/Chapsvision-dev/vault-raft-backup-restore/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Chapsvision-dev/vault-raft-backup-restore)](https://goreportcard.com/report/github.com/Chapsvision-dev/vault-raft-backup-restore)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Chapsvision-dev/vault-raft-backup-restore)](https://go.dev/)

A lightweight Go operator to **backup** and **restore** Vault Raft snapshots, with pluggable storage providers (initial support: **Azure Blob Storage**).

## Why?

Vault's integrated storage (Raft) is the recommended backend in modern deployments.
This operator automates snapshot management so you can:

* Safely **backup** Raft state to cloud/object storage.
* **Restore** state for disaster recovery or local testing.
* Integrate with CI/CD pipelines and automation.

## Features

* Vault Raft snapshot support (`/v1/sys/storage/raft/snapshot`)
* **Pluggable auth**:
  * Static Vault Token (dev/local)
  * Kubernetes ServiceAccount + Vault Role (production)
* Pluggable providers (Azure included)
* Local dev environment via Docker Compose
* Developer-friendly Makefile targets
* CI-ready commands (`build`, `test`, `lint`)

---

## Installation

### Binary

Download the latest release from the [releases page](https://github.com/Chapsvision-dev/vault-raft-backup-restore/releases).

### Docker

```bash
docker pull ghcr.io/chapsvision-dev/vault-raft-backup-operator:latest
```

### From Source

```bash
git clone https://github.com/Chapsvision-dev/vault-raft-backup-restore.git
cd vault-raft-backup-restore
make build
```

---

## Quickstart

### Prerequisites

* Go (≥ 1.25)
* Docker + Docker Compose
* (Optional) [`gow`](https://github.com/mitranim/gow) for live reload
* (Optional) `golangci-lint` for extended linting

### Setup

**For asdf users:**

If you use [asdf](https://asdf-vm.com/) to manage tool versions, the project includes a `.tool-versions` file that specifies the exact Go version (1.25.5):

```bash
# Install the Go plugin if not already installed
asdf plugin add golang https://github.com/asdf-community/asdf-golang.git

# Install the exact version specified in .tool-versions
asdf install

# Verify the version
asdf current golang
# golang          1.25.5          /path/to/project/.tool-versions
```

**Standard setup:**

```bash
# Align local tool versions (if using asdf)
make setup

# Install gow (live reload dev tool)
make go-tools
```

### Run Vault (Raft mode) locally

```bash
# Start Vault in Docker Compose
make up

# Check logs
make logs

# Initialize + unseal (prints root token)
make init
```

Vault UI will be available at:
👉 [http://localhost:8200](http://localhost:8200)

### Backup / Restore

Ensure your `.env` is set with Vault + provider settings.
Two auth modes are supported:

* `VAULT_AUTH_METHOD=token` (requires `VAULT_TOKEN`)
* `VAULT_AUTH_METHOD=kubernetes` (requires ServiceAccount JWT + role)

```bash
# Run backup: creates a Raft snapshot and uploads to Azure
make backup

# Run restore: downloads a snapshot from Azure and restores it into Vault
make restore
```

---

## Configuration

Use a `.env` file in the project root. Example:

```dotenv
########################################
# Vault connection
########################################
VAULT_ADDR=http://localhost:8200
VAULT_AUTH_METHOD=token   # token | kubernetes

# If method=token
VAULT_TOKEN=s.xxxxx

# If method=kubernetes
VAULT_K8S_ROLE=vault-raft-backup
VAULT_K8S_MOUNT=kubernetes
VAULT_K8S_JWT_PATH=/var/run/secrets/kubernetes.io/serviceaccount/token
# VAULT_K8S_AUDIENCE= # optional

########################################
# Backup provider
########################################
BACKUP_PROVIDER=azure
AZURE_STORAGE_ACCOUNT=myaccount
AZURE_STORAGE_CONTAINER=vault-backups
AZURE_STORAGE_SAS=?sv=2025-...

########################################
# Backup / Restore
########################################
BACKUP_TARGET=snapshots
RESTORE_SOURCE=snapshots/2025-09-12T14-53-26Z.snap
```

See [.env.dist](.env.dist) for a complete example.

---

## Common Tasks

```bash
# Show all available commands
make help

# Build / test / lint
make build
make test
make lint

# Run operator with live reload
make dev

# Clean build artifacts
make clean

# Docker image
make docker-build
make docker-push
```

---

## Project Layout

* `cmd/operator/` – CLI entrypoint
* `internal/config/` – configuration loading
* `internal/auth/` – Vault authentication (token, Kubernetes)
* `internal/vault/` – Vault Raft snapshot primitives
* `internal/provider/` – provider interfaces & registry
* `internal/provider/azure/` – Azure provider
* `internal/snapshot/`, `internal/restore/` – services
* `internal/retry/`, `internal/util/`, `internal/logx/` – helpers

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

Licensed under the Apache License, Version 2.0.
See [LICENSE.md](./LICENSE.md) for details.

---

## Community

* 🐛 [Report a bug](https://github.com/Chapsvision-dev/vault-raft-backup-restore/issues/new?template=bug_report.yml)
* 💡 [Request a feature](https://github.com/Chapsvision-dev/vault-raft-backup-restore/issues/new?template=feature_request.yml)
* 💬 [Start a discussion](https://github.com/Chapsvision-dev/vault-raft-backup-restore/discussions)
* 🔒 [Report a security vulnerability](https://github.com/Chapsvision-dev/vault-raft-backup-restore/security/advisories/new)
