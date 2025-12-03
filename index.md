# Quick Start

**AuthKeySync** is a lightweight CLI utility that fetches SSH public keys from remote URLs (GitHub, GitLab, or any API) and syncs them to the `authorized_keys` files on your servers.

This is a quick start guide to get you up and running. For more details, see:

- [Installation](https://eduardolat.github.io/authkeysync/installation/index.md): All download options and platform support
- [Configuration](https://eduardolat.github.io/authkeysync/configuration/index.md): Complete reference for all options
- [Usage](https://eduardolat.github.io/authkeysync/usage/index.md): CLI options, automation, and troubleshooting
- [Technical Specification](https://eduardolat.github.io/authkeysync/spec/index.md): Full technical details and behavior

## Why AuthKeySync?

Managing SSH keys manually is tedious and error-prone:

- When someone joins your team, you add their key to 20 servers
- When someone leaves, you hope you remembered all the places their key exists
- When someone rotates their key, the cycle repeats
- Homemade bash scripts for this are often poorly written, insecure, or have subtle bugs that can lock you out

With AuthKeySync, you define your key sources once in a YAML file and let it handle the rest. When someone rotates their GitHub SSH key, all servers pick up the change on the next sync.

## Install

Download the binary for your platform from the [releases page](https://github.com/eduardolat/authkeysync/releases):

```
# Linux AMD64
curl -Lo authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-amd64
chmod +x authkeysync
sudo mv authkeysync /usr/local/bin/
```

Other platforms: Linux ARM64, macOS Intel, macOS Apple Silicon. See [Installation](https://eduardolat.github.io/authkeysync/installation/index.md) for details.

## Configure

Create `/etc/authkeysync/config.yaml`:

```
policy:
  backup_enabled: true # Create backups before changes (default: true)
  backup_retention_count: 10 # Number of backups to keep (default: 10)
  preserve_local_keys: true # Keep keys not in remote sources (default: true)

users:
  - username: "root"
    sources:
      - url: "https://github.com/your-username.keys"
      - url: "https://github.com/another-username.keys"
```

### All Configuration Options

**Policy** (all fields optional, shown with defaults):

| Option                   | Type | Default | Description                                       |
| ------------------------ | ---- | ------- | ------------------------------------------------- |
| `backup_enabled`         | bool | `true`  | Create backups before modifying `authorized_keys` |
| `backup_retention_count` | int  | `10`    | Number of backup files to keep per user           |
| `preserve_local_keys`    | bool | `true`  | Keep existing keys that are not in remote sources |

**Users** (required):

| Option     | Type   | Required | Description                              |
| ---------- | ------ | -------- | ---------------------------------------- |
| `username` | string | Yes      | System username (e.g., `root`, `deploy`) |
| `sources`  | list   | Yes      | List of key sources                      |

**Sources**:

| Option            | Type   | Default    | Description                                    |
| ----------------- | ------ | ---------- | ---------------------------------------------- |
| `url`             | string | (required) | URL that returns plain text SSH keys           |
| `method`          | string | `GET`      | HTTP method: `GET` or `POST`                   |
| `headers`         | map    | `{}`       | Custom HTTP headers (e.g., for authentication) |
| `body`            | string | `""`       | Request body for POST requests                 |
| `timeout_seconds` | int    | `10`       | Request timeout in seconds                     |

See [Configuration](https://eduardolat.github.io/authkeysync/configuration/index.md) for more examples.

## Run

```
# Test first (no changes made)
sudo authkeysync --dry-run

# Apply changes
sudo authkeysync
```

### CLI Options

| Option            | Description                                                   |
| ----------------- | ------------------------------------------------------------- |
| `--config <path>` | Path to config file (default: `/etc/authkeysync/config.yaml`) |
| `--dry-run`       | Simulate sync without modifying any files                     |
| `--debug`         | Enable debug logging (most verbose)                           |
| `--quiet`         | Show only warnings and errors (recommended for cron)          |
| `--silent`        | Show only errors (most quiet)                                 |
| `--version`       | Show version information                                      |
| `--help`          | Show help message                                             |

See [Usage](https://eduardolat.github.io/authkeysync/usage/index.md) for automation, monitoring, and troubleshooting.

## Automate

Set up a cron job to run periodically (use `--quiet` to reduce log noise):

```
# Every 5 minutes (--quiet shows only warnings and errors)
echo "*/5 * * * * root /usr/local/bin/authkeysync --quiet" | sudo tee /etc/cron.d/authkeysync
```

See [Usage](https://eduardolat.github.io/authkeysync/usage/index.md) for systemd timers, cloud-init, Ansible, and Terraform examples.

## How It Works

1. **Fetch**: Downloads keys from all configured URLs
1. **Parse**: Validates SSH key format and removes duplicates
1. **Merge**: Combines remote keys with local keys (if enabled)
1. **Write**: Atomically updates `authorized_keys` with proper permissions

If any fetch fails, AuthKeySync **aborts the update for that user** to prevent accidental lockouts.

For complete technical details about parsing rules, deduplication logic, atomic writes, and backup management, see the [Technical Specification](https://eduardolat.github.io/authkeysync/spec/index.md).

## Next Steps

- **[Installation](https://eduardolat.github.io/authkeysync/installation/index.md)**: All download options and platform support
- **[Configuration](https://eduardolat.github.io/authkeysync/configuration/index.md)**: Complete reference for all options
- **[Usage](https://eduardolat.github.io/authkeysync/usage/index.md)**: CLI options, automation, and troubleshooting
- **[Technical Specification](https://eduardolat.github.io/authkeysync/spec/index.md)**: Full technical details and behavior
