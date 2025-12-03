# Configuration

AuthKeySync is configured via a YAML file. By default, it looks for `/etc/authkeysync/config.yaml`.

## Basic Example

```
policy:
  backup_enabled: true
  preserve_local_keys: true

users:
  - username: "root"
    sources:
      - url: "https://github.com/your-username.keys"
```

## Complete Reference

### Policy Section

The `policy` section defines global behavior for all users. All fields are optional and have sensible defaults.

| Option                   | Type | Default | Description                                       |
| ------------------------ | ---- | ------- | ------------------------------------------------- |
| `backup_enabled`         | bool | `true`  | Create backups before modifying `authorized_keys` |
| `backup_retention_count` | int  | `10`    | Number of backup files to keep per user           |
| `preserve_local_keys`    | bool | `true`  | Keep existing keys that are not in remote sources |

#### About `preserve_local_keys`

This is a critical safety setting:

- **`true` (default)**: Keys that exist locally but not in any remote source are preserved. This prevents accidental lockouts if you have manually added keys.
- **`false`**: The `authorized_keys` file will contain **only** the keys from remote sources. Any manually added keys will be removed.

Be careful with `preserve_local_keys: false`

Setting this to `false` means remote sources become the single source of truth. If a source is misconfigured or returns empty, you could lose access.

### Users Section

The `users` section is a list of system users to manage.

| Option     | Type   | Required | Description                              |
| ---------- | ------ | -------- | ---------------------------------------- |
| `username` | string | Yes      | System username (e.g., `root`, `deploy`) |
| `sources`  | list   | Yes      | List of key sources (see below)          |

### Sources

Each source defines where to fetch SSH keys from.

| Option            | Type   | Default    | Description                          |
| ----------------- | ------ | ---------- | ------------------------------------ |
| `url`             | string | (required) | URL that returns plain text SSH keys |
| `method`          | string | `GET`      | HTTP method: `GET` or `POST`         |
| `headers`         | map    | `{}`       | Custom HTTP headers                  |
| `body`            | string | `""`       | Request body for POST requests       |
| `timeout_seconds` | int    | `10`       | Request timeout in seconds           |

## Common Configurations

### GitHub Keys

GitHub exposes public SSH keys at `https://github.com/{username}.keys`:

```
users:
  - username: "deploy"
    sources:
      - url: "https://github.com/your-username.keys"
      - url: "https://github.com/another-username.keys"
      - url: "https://github.com/third-username.keys"
```

### GitLab Keys

GitLab exposes public SSH keys at `https://gitlab.com/{username}.keys`:

```
users:
  - username: "deploy"
    sources:
      - url: "https://gitlab.com/your-username.keys"
```

### Self-Hosted GitLab

```
users:
  - username: "deploy"
    sources:
      - url: "https://gitlab.yourcompany.com/your-username.keys"
```

### Multiple Sources per User

You can combine keys from multiple sources:

```
users:
  - username: "admin"
    sources:
      # GitHub keys
      - url: "https://github.com/your-username.keys"
      # GitLab keys
      - url: "https://gitlab.com/your-username.keys"
      # Internal key server
      - url: "https://keys.yourcompany.com/your-username"
```

### Private API with Authentication

For internal key servers that require authentication:

```
users:
  - username: "deploy"
    sources:
      - url: "https://vault.yourcompany.com/v1/ssh/keys"
        method: "POST"
        headers:
          Authorization: "Bearer your-secret-token"
          Content-Type: "application/json"
        body: '{"role": "deployment", "environment": "prod"}'
        timeout_seconds: 5
```

### Multiple Users

```
policy:
  backup_enabled: true
  preserve_local_keys: true
  backup_retention_count: 5

users:
  - username: "root"
    sources:
      - url: "https://github.com/admin-username.keys"

  - username: "deploy"
    sources:
      - url: "https://github.com/ci-bot-username.keys"
      - url: "https://github.com/developer-username.keys"

  - username: "backup"
    sources:
      - url: "https://keys.yourcompany.com/backup-service"
        headers:
          X-API-Key: "secret-key"
```

### Disabling Backups

If you don't need backup files:

```
policy:
  backup_enabled: false
  preserve_local_keys: true

users:
  - username: "root"
    sources:
      - url: "https://github.com/your-username.keys"
```

### Strict Mode (Remote-Only Keys)

To make remote sources the single source of truth:

```
policy:
  backup_enabled: true
  backup_retention_count: 20
  preserve_local_keys: false

users:
  - username: "deploy"
    sources:
      - url: "https://github.com/team-lead-username.keys"
```

Warning

With `preserve_local_keys: false`, any key not present in your sources will be **removed**. Make sure your sources are reliable before enabling this.

## Output Format

AuthKeySync generates a well-structured `authorized_keys` file:

```
# ──────────────────────────────────────────────────────────────────
# Generated by AuthKeySync
# Version:   vX.X.X
# Commit:    <commit hash>
# Built:     <ISO 8601 timestamp (UTC)>
# Last sync: <ISO 8601 timestamp (UTC)>
# More info: https://github.com/eduardolat/authkeysync
# ──────────────────────────────────────────────────────────────────

# Source: https://github.com/your-username.keys
ssh-ed25519 AAAA... user@laptop

# Source: https://github.com/another-username.keys
ssh-rsa AAAA... user@workstation

# Local (preserved)
ssh-ed25519 AAAA... manually-added-key
```

## Backups

When `backup_enabled: true`, AuthKeySync creates backups in:

```
~/.ssh/authorized_keys_backups/
├── authorized_keys_20240115_103045_abcdef
├── authorized_keys_20240115_093012_ghijkl
└── authorized_keys_20240114_180022_mnopqr
```

Backups are only created when the content actually changes. The oldest files are automatically deleted based on `backup_retention_count`.

## Validation

AuthKeySync validates the configuration file on startup. Common errors:

- **No users defined**: At least one user is required
- **Empty username**: Username cannot be blank
- **No sources**: Each user must have at least one source
- **Empty URL**: Each source must have a URL
- **Invalid method**: Only `GET` and `POST` are supported
- **Invalid timeout**: Timeout must be positive

## Environment Considerations

### Permissions

The config file may contain sensitive data (API tokens). Protect it:

```
sudo chmod 600 /etc/authkeysync/config.yaml
```

### Network Access

Ensure your server can reach the configured URLs. For internal APIs, check:

- Firewall rules
- DNS resolution
- Proxy settings (if applicable)

### User Requirements

For each configured user:

1. The user must exist in the system
1. The user must have a home directory
1. The `~/.ssh/` directory must exist

If any of these conditions are not met, AuthKeySync logs a warning and skips that user.

## Next Steps

- [Usage](https://eduardolat.github.io/authkeysync/usage/index.md): CLI options and automation
- [Technical Specification](https://eduardolat.github.io/authkeysync/spec/index.md): Deep dive into behavior
