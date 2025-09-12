# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of vault-raft-backup-restore seriously. If you believe you have found a security vulnerability, please report it to us responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via GitHub's Security Advisory feature:

1. Go to https://github.com/Chapsvision-dev/vault-raft-backup-restore/security/advisories/new
2. Fill out the advisory form with details about the vulnerability
3. Click "Submit report"

Alternatively, you can email security reports to: **cloudopsbur@chapsvision.com**

### What to Include

Please include the following information in your report:

* Type of vulnerability (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
* Full paths of source file(s) related to the manifestation of the issue
* The location of the affected source code (tag/branch/commit or direct URL)
* Any special configuration required to reproduce the issue
* Step-by-step instructions to reproduce the issue
* Proof-of-concept or exploit code (if possible)
* Impact of the issue, including how an attacker might exploit it

### Response Timeline

* **Within 24 hours**: We will acknowledge receipt of your vulnerability report
* **Within 7 days**: We will provide a more detailed response indicating the next steps
* **Within 30 days**: We aim to release a fix for confirmed vulnerabilities

### Disclosure Policy

* We request that you do not publicly disclose the vulnerability until we have released a fix
* We will credit you in the security advisory (unless you prefer to remain anonymous)
* We will coordinate the disclosure timeline with you

## Security Best Practices

When using vault-raft-backup-restore, we recommend:

### Authentication

* **Never commit** Vault tokens or credentials to version control
* Use Kubernetes ServiceAccount authentication in production
* Rotate tokens regularly
* Use least-privilege access for Vault policies

### Storage

* Use SAS tokens with minimal permissions for Azure Blob Storage
* Set appropriate expiration times on SAS tokens
* Use HTTPS for all Vault communication
* Enable TLS verification (`VAULT_SKIP_VERIFY=false`)

### Snapshots

* Encrypt snapshots at rest (use Azure's encryption features)
* Restrict access to snapshot storage containers
* Regularly test your restore procedures
* Keep snapshots in a separate account/subscription from production

### Deployment

* Run the operator with minimal privileges
* Use read-only filesystem where possible
* Enable security contexts in Kubernetes (non-root user, drop capabilities)
* Scan container images for vulnerabilities regularly

## Security Updates

Security updates will be released as:

* Patch releases for the current major version
* Documented in [CHANGELOG.md](CHANGELOG.md)
* Announced in GitHub Releases
* Tagged with the `security` label

## Known Security Considerations

### Snapshot Contents

Vault snapshots contain sensitive data including:

* All secrets stored in Vault
* Encryption keys
* Audit logs
* Vault configuration

**Always treat snapshots as highly sensitive data.**

### In-Transit Data

During backup/restore operations:

* Snapshots are transferred in memory between Vault and storage
* Use HTTPS for Vault API calls
* Use encrypted connections to cloud storage

### Audit

This operator does not provide its own audit logging. For audit trails:

* Enable Vault audit logging
* Monitor cloud storage access logs
* Use cloud provider audit features (Azure Monitor, etc.)

## Security Contacts

For security-related questions that are not vulnerabilities, please open a GitHub Discussion in the Security category.

---

Thank you for helping keep vault-raft-backup-restore and its users safe!
