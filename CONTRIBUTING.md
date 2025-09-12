# Contributing Guide

We welcome contributions to the Vault Raft Backup & Restore Operator! This project is maintained by Chapsvision, but we encourage community contributions and use.

By contributing to this repository, you agree that your contributions will be licensed under the Apache 2.0 License.

---

## Development Setup

```bash
# Clone the repository
git clone https://github.com/Chapsvision-dev/vault-raft-backup-restore.git
cd vault-raft-backup-restore

# Install tooling (if using asdf)
make setup
make go-tools

# Start local Vault (Raft mode)
make up
make init

# Dev loop (auto-reloads on changes)
make dev
```

Run checks before submitting:

```bash
make fmt
make tidy
make lint
make test
```

---

## Branching

Use short, descriptive branch names:

* `feat/<short-desc>` - new features
* `fix/<short-desc>` - bug fixes
* `docs/<short-desc>` - documentation
* `chore/<short-desc>` - maintenance

---

## Commit Messages — Conventional Commits

We follow [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

**Format:**

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Common types:**

* `feat`: a new feature
* `fix`: a bug fix
* `docs`: documentation only changes
* `chore`: tooling or maintenance
* `refactor`: code changes that neither fix a bug nor add a feature
* `perf`: performance improvements
* `test`: adding or correcting tests
* `ci`: CI/CD changes

**Examples:**

```
feat(backup): add Azure Blob snapshot upload
fix(restore): handle missing local file gracefully
docs: update Quickstart for Vault
chore(ci): enable caching for go modules
```

If relevant, include footers like:

```
BREAKING CHANGE: rename flag --snapshot-uri to --source
Closes #123
```

---

## Pull Requests

* Keep PRs small and focused.
* Add/update tests and docs.
* Link issues (e.g., `Closes #123`).
* CI must pass.
* Wait for at least one approval before merging.

---

## Code Style

* Run `make tidy` to keep `go.mod` clean.
* Run `make fmt` to enforce Go formatting.
* Run `make lint` to check code quality with golangci-lint (config: `.golangci.yml`).
* Prefer small, testable packages in `internal/`.
* Use structured logging (`zerolog`) responsibly; avoid noisy logs in hot paths.
* Add tests for new features and bug fixes.

---

## Releases

* Releases are tagged from `main`.
* Changelog is generated from Conventional Commits.
* Versioning follows [Semantic Versioning](https://semver.org/).

---

## Getting Help

* Check existing [issues](https://github.com/Chapsvision-dev/vault-raft-backup-restore/issues)
* Open a new issue for bugs or feature requests
* Start a [discussion](https://github.com/Chapsvision-dev/vault-raft-backup-restore/discussions) for questions

---

## License

Licensed under the Apache License, Version 2.0. See [LICENSE.md](./LICENSE.md) for details.
