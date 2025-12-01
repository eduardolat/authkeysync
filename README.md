<p align="center">
  <img src="docs/assets/logo.png" alt="AuthKeySync Logo" width="200" />
</p>

<h1 align="center">AuthKeySync</h1>

<p align="center">
  <strong>Automatically synchronize SSH public keys from remote URLs to your servers</strong>
</p>

<p align="center">
  <a href="https://github.com/eduardolat/authkeysync/actions/workflows/ci.yaml?query=branch%3Amain">
    <img src="https://github.com/eduardolat/authkeysync/actions/workflows/ci.yaml/badge.svg" alt="CI Status"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/eduardolat/authkeysync">
    <img src="https://goreportcard.com/badge/github.com/eduardolat/authkeysync" alt="Go Report Card"/>
  </a>
  <a href="https://github.com/eduardolat/authkeysync/releases/latest">
    <img src="https://img.shields.io/github/release/eduardolat/authkeysync.svg" alt="Release Version"/>
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/github/license/eduardolat/authkeysync.svg" alt="License"/>
  </a>
  <a href="https://github.com/eduardolat/authkeysync">
    <img src="https://img.shields.io/github/stars/eduardolat/authkeysync?style=flat&label=github+stars"/>
  </a>
</p>

<p align="center">
  📖 <a href="https://eduardolat.github.io/authkeysync"><strong>Full Documentation</strong></a>
</p>

## The Problem

Managing SSH access across multiple servers is painful:

- Team members join or leave, and you need to update `authorized_keys` on every server
- Developers rotate their SSH keys, and now you have 20 servers to update
- You're using Infrastructure as Code, but SSH key management is still manual
- Homemade bash scripts for key management are often poorly written, insecure, or have subtle bugs that can lock you out of your servers
- You want to use GitHub/GitLab keys, but copying them everywhere is tedious

## The Solution

**AuthKeySync** is a lightweight CLI that fetches SSH public keys from URLs (GitHub, GitLab, your own API) and syncs them to your servers. It's safe, reliable, and designed for automation.

```yaml
# /etc/authkeysync/config.yaml
users:
  - username: "deploy"
    sources:
      - url: "https://github.com/your-username.keys"
      - url: "https://github.com/another-username.keys"
```

Run `authkeysync` (manually or via cron), and your `authorized_keys` is updated. That's it.

## Key Features

- **Single binary**: No dependencies, just download and run
- **Safe by default**: Preserves existing local keys, creates backups, uses atomic writes
- **Fail-safe**: If any source fails, the update is aborted to prevent lockouts
- **Flexible sources**: GitHub, GitLab, or any URL returning plain text SSH keys
- **API support**: POST requests with custom headers for authenticated APIs
- **IaC-ready**: Stateless, idempotent, perfect for Ansible/Terraform/cloud-init
- **Cross-platform**: Works on Linux and macOS (AMD64 and ARM64)

## Quick Start

### 1. Download

Get the latest binary from the [releases page](https://github.com/eduardolat/authkeysync/releases):

```bash
# Linux AMD64
curl -Lo authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-amd64
chmod +x authkeysync
sudo mv authkeysync /usr/local/bin/
```

### 2. Configure

Create a config file at `/etc/authkeysync/config.yaml`:

```yaml
policy:
  backup_enabled: true # Create backups before changes (default: true)
  backup_retention_count: 10 # Number of backups to keep (default: 10)
  preserve_local_keys: true # Keep keys not in remote sources (default: true)

users:
  - username: "root"
    sources:
      - url: "https://github.com/your-username.keys"
```

### 3. Run

```bash
# Test first with dry-run
sudo authkeysync --dry-run

# Apply changes
sudo authkeysync
```

### 4. Automate

Set up a cron job or systemd timer to run periodically:

```bash
# Every 5 minutes
echo "*/5 * * * * root /usr/local/bin/authkeysync" | sudo tee /etc/cron.d/authkeysync
```

## Configuration Options

### Policy (all optional)

| Option                   | Type | Default | Description                                       |
| ------------------------ | ---- | ------- | ------------------------------------------------- |
| `backup_enabled`         | bool | `true`  | Create backups before modifying `authorized_keys` |
| `backup_retention_count` | int  | `10`    | Number of backup files to keep per user           |
| `preserve_local_keys`    | bool | `true`  | Keep existing keys that are not in remote sources |

### Users (required)

| Option     | Type   | Required | Description                              |
| ---------- | ------ | -------- | ---------------------------------------- |
| `username` | string | Yes      | System username (e.g., `root`, `deploy`) |
| `sources`  | list   | Yes      | List of key sources                      |

### Sources

| Option            | Type   | Default    | Description                                    |
| ----------------- | ------ | ---------- | ---------------------------------------------- |
| `url`             | string | (required) | URL that returns plain text SSH keys           |
| `method`          | string | `GET`      | HTTP method: `GET` or `POST`                   |
| `headers`         | map    | `{}`       | Custom HTTP headers (e.g., for authentication) |
| `body`            | string | `""`       | Request body for POST requests                 |
| `timeout_seconds` | int    | `10`       | Request timeout in seconds                     |

### Example with all options

```yaml
policy:
  backup_enabled: true
  backup_retention_count: 10
  preserve_local_keys: true

users:
  - username: "deploy"
    sources:
      # Simple GitHub keys
      - url: "https://github.com/your-username.keys"

      # Private API with authentication
      - url: "https://keys.yourcompany.com/api/keys"
        method: "POST"
        headers:
          Authorization: "Bearer your-secret-token"
          Content-Type: "application/json"
        body: '{"environment": "production"}'
        timeout_seconds: 5
```

## CLI Options

| Option            | Description                                                   |
| ----------------- | ------------------------------------------------------------- |
| `--config <path>` | Path to config file (default: `/etc/authkeysync/config.yaml`) |
| `--dry-run`       | Simulate sync without modifying any files                     |
| `--version`       | Show version information and exit                             |
| `--help`          | Show help message                                             |

### Exit Codes

| Code | Meaning                                   |
| ---- | ----------------------------------------- |
| `0`  | Success: all users processed or skipped   |
| `1`  | Failure: at least one user failed to sync |

## How It Works

1. **Fetch**: Downloads SSH keys from configured URLs
2. **Parse**: Validates and deduplicates keys across all sources
3. **Merge**: Optionally preserves keys that only exist locally
4. **Write**: Atomically updates `authorized_keys` with proper permissions
5. **Backup**: Creates timestamped backups before any changes

If any source fails to fetch, AuthKeySync **aborts the update for that user** to prevent lockouts. Your existing access remains intact.

## Support the Project

If you find AuthKeySync useful, please consider giving it a star on GitHub and following me on X (Twitter) for updates:

- [Star on GitHub](https://github.com/eduardolat/authkeysync)
- [Follow @eduardoolat on X](https://eduardo.lat/x)

## License

MIT License. See [LICENSE](LICENSE) for details.
