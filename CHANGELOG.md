# Changelog

All notable changes to this project will be documented in this file. See [Conventional Commits](https://conventionalcommits.org) for commit guidelines.

## 1.0.0 (2025-12-23)

### ⚠ BREAKING CHANGES

* establishes first versioned release with semver workflow

### Features

* add automated semver release system with Docker Hub multi-arch builds ([992fb2e](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/992fb2e02a82a1ab58acf98d3915d17cc0fddea6))
* initial release of vault-raft-backup-restore ([b558f6c](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/b558f6c4a1264cd934018812ae808e10e9ae223d))

### Bug Fixes

* prevent releases from being created as drafts ([6c3613f](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/6c3613fa8c5d7b0f45f062357202c6a94bc2aa86))

### Code Refactoring

* **azure:** reduce Backup() cyclomatic complexity from 17 to 8 ([3c4f9b5](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/3c4f9b5355e2a2f5981086dbcc99287076dbf97e))
* **config:** reduce Load() cyclomatic complexity from 30 to 11 ([e467a3a](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/e467a3ade3c0a34bf0b19b2a7f3f4f1147aebfc8))
* **vault:** reduce snapshot functions cyclomatic complexity ([ca9a95e](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/ca9a95ec88130188f7368e7f21aa8c5b44bcb651))

### Documentation

* add asdf setup instructions for Go version management ([2042541](https://github.com/Chapsvision-dev/vault-raft-backup-restore/commit/204254119b59e63cf64b38372770c038bae25df2))
