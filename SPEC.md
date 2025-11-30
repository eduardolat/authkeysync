# AuthKeySync Technical Specification

**Target Architecture:** Linux & macOS (POSIX Compliant)

## 1. Executive Summary

**AuthKeySync** is a lightweight, high-integrity CLI utility designed to synchronize SSH public keys from remote URLs into local `authorized_keys` files.

Unlike complex configuration management agents, AuthKeySync follows the **Unix Philosophy**: it does one thing well (synchronization) and exits. It is architected for **Infrastructure as Code (IAC)** environments (but can be used everywhere and in any way), providing idempotency, atomic file operations, and zero-dependency portability.

### Core Design Principles

1. **Universal Portability:** The binary is statically compiled, ensuring it runs on any Linux distribution (Alpine, Debian, RHEL, NixOS, etc) or macOS version without requiring complex system libraries or other dependencies.
2. **One-Shot Execution:** The tool is stateless. It executes a single synchronization cycle and terminates. Scheduling is delegated to the native OS facility (Systemd Timers, Cron, Launchd, etc).
3. **Fail-Safe Isolation:** A failure in one user's synchronization process **must never** impact other users or corrupt existing access.

### User Responsibilities

AuthKeySync is designed to be as safe as possible: it uses atomic writes, preserves local keys by default, creates backups, and aborts on network errors. However, the tool **cannot validate the trustworthiness or correctness of remote sources**. The user is responsible for:

1. **Source Trustworthiness:** Only configure sources that you control or fully trust. Sources must not be publicly editable (e.g., a wiki page, a public gist, or an unauthenticated API). A malicious actor with write access to a source can inject their own SSH keys and gain system access.

2. **Source Content Validity:** Ensure that configured sources return valid SSH public keys in plain text format. AuthKeySync performs minimal structural validation but does not verify cryptographic correctness. Garbage in, garbage out.

3. **Access Control:** Protect the AuthKeySync configuration file (`/etc/authkeysync/config.yaml`) with appropriate permissions. This file may contain sensitive information (API tokens, internal URLs) and defines who can access your systems.

4. **Monitoring:** Monitor synchronization logs and exit codes. A persistent exit code `1` indicates a problem that requires attention.

AuthKeySync's responsibility ends at correctly fetching, parsing, deduplicating, and atomically writing the keys. **The security of your sources is your responsibility.**

## 2. Configuration Specification

The application is configured exclusively via **YAML**.

- **Default Path:** `/etc/authkeysync/config.yaml`
- **CLI Override:** `authkeysync --config <path>`

### 2.1 Configuration Schema

The configuration is divided into two sections: `policy` (global behavior) and `users` (target definitions).

#### Section: `policy`

Defines the safety rules for the synchronization process.

| Field                    | Type | Required | Default | Description                                                                                                                                                                                         |
| :----------------------- | :--- | :------- | :------ | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `backup_enabled`         | bool | No       | `true`  | If `true`, a backup of the existing `authorized_keys` is created before overwriting.                                                                                                                |
| `backup_retention_count` | int  | No       | `10`    | Number of unique backup files to keep per user. Oldest files are deleted first.                                                                                                                     |
| `preserve_local_keys`    | bool | No       | `true`  | **Critical.** If `true`, keys found in the local file that are absent from remote sources are **kept** (merged). If `false`, the local file is **overwritten** to exactly match the remote sources. |

#### Section: `users`

A list of system users to manage.

| Field      | Type   | Required | Default | Description                                                |
| :--------- | :----- | :------- | :------ | :--------------------------------------------------------- |
| `username` | string | **Yes**  | N/A     | The exact system login name (e.g., `root`, `bob`, `john`). |
| `sources`  | list   | **Yes**  | N/A     | A list of source objects (see below) to fetch keys from.   |

#### Section: `users[].sources`

Defines the HTTP endpoint for fetching keys.

| Field             | Type   | Required | Default | Description                                                                     |
| :---------------- | :----- | :------- | :------ | :------------------------------------------------------------------------------ |
| `url`             | string | **Yes**  | N/A     | The remote URL. **Must return plain text** (standard `authorized_keys` format). |
| `method`          | string | No       | `"GET"` | HTTP Method. Supported: `GET`, `POST`.                                          |
| `headers`         | map    | No       | `{}`    | Key-Value map for custom headers (e.g., `Authorization`).                       |
| `body`            | string | No       | `""`    | Raw string body payload for `POST` requests (used for auth/query parameters).   |
| `timeout_seconds` | int    | No       | `10`    | Max duration to wait for this specific request.                                 |

### 2.2 Example Configuration

```yaml
policy:
  backup_enabled: true
  backup_retention_count: 10
  preserve_local_keys: true # Safe default: do not delete manual keys

users:
  - username: "admin"
    sources:
      # Standard GitHub Keys (GET returning text/plain)
      - url: "https://github.com/my-admin.keys"

  - username: "deploy_bot"
    sources:
      # Corporate Vault/API (POST with Auth)
      # RESPONSE MUST BE PLAIN TEXT KEYS (Not JSON)
      - url: "https://vault.internal/v1/ssh/keys/raw"
        method: "POST"
        timeout_seconds: 5
        headers:
          Authorization: "Bearer SECRET_TOKEN_XYZ"
          Content-Type: "application/json"
        body: '{"role": "deployment", "env": "prod"}'
```

## 3. Operational Logic & Error Handling

The application implements a **Blast Radius Containment** strategy.

### 3.1 Validation Hierarchy

1. **System Check:**
   - If `username` does not exist in the OS → **Log Warning & SKIP User.**
   - If user exists but the `.ssh` directory (inside the user's home directory) is missing or invalid → **Log Warning & SKIP User.**
2. **Network Fetch:**
   - The tool iterates through all `sources` for a user.
   - **Logic:** If **ANY** source for a specific user fails (non-200 status, timeout, DNS error), the entire update for that user is marked as **FAILED**.
   - **Action:** Log Error & **ABORT** update for this user. The existing `authorized_keys` file remains untouched.

### 3.2 Key Parsing Rules

Content is processed as a plain text stream, parsed line-by-line. SSH public keys **never span multiple lines**.

> **Important:** The same parsing algorithm is applied uniformly to **both** remote source content **and** the existing local `authorized_keys` file. There is no special treatment for local keys.

#### Processing Steps

For each line (if there are no lines, the file is discarded):

1. **Trim:** Leading and trailing whitespace is removed from the line.
2. **Classify:** The trimmed line is classified according to the table below.
3. **Validate:** If classified as a potential key, structural validation is applied.

#### Line Classification

| Line Type       | Detection (after trim)                   | Action      |
| :-------------- | :--------------------------------------- | :---------- |
| Empty line      | Zero length                              | **Discard** |
| Comment line    | Starts with `#`                          | **Discard** |
| HTML/JSON error | Starts with `<`, `{`, or `[`             | **Discard** |
| Valid SSH key   | Passes structural validation (see below) | **Keep**    |
| Malformed line  | Does not match any above                 | **Discard** |

#### Structural Validation

A trimmed line is considered a valid SSH public key if **all** of the following conditions are met:

1. The line is not empty.
2. The line does not start with `#`, `<`, `{`, or `[`.
3. The line contains **at least 2 whitespace-separated fields**. Lines with 3, 4, or more fields are valid (additional fields are typically the optional comment or SSH options).

This minimal validation ensures forward compatibility with any current or future SSH key type. The tool does not maintain a whitelist of key algorithms, nor does it validate key content or encoding.

#### SSH Tolerance

OpenSSH is tolerant of malformed lines in `authorized_keys`. If a line does not represent a valid SSH key, SSH silently ignores it—authentication for valid keys continues to work normally. This means that even if a non-key line passes the structural validation above, it will not break SSH access. The worst-case scenario is a harmless, ignored line in the file.

#### Key Anatomy Reference

A standard `authorized_keys` line has the following format:

```
[options] <key-type> <base64-blob> [comment]
```

- **options** (optional): Comma-separated restrictions (e.g., `restrict,port-forwarding`).
- **key-type**: Algorithm identifier (e.g., `ssh-ed25519`, `ssh-rsa`, `ecdsa-sha2-nistp256`, `sk-ssh-ed25519@openssh.com`).
- **base64-blob**: The actual public key data, base64-encoded. **Never contains spaces or newlines.**
- **comment** (optional): Free-form text, typically `user@host`. No `#` prefix.

### 3.3 Key Deduplication

Keys are deduplicated globally across all sources and the local file.

#### Comparison Method

Two lines are considered **identical** if their **trimmed content matches exactly** (byte-for-byte comparison after whitespace trimming). This means:

- `ssh-ed25519 AAA... user@host` and `ssh-ed25519 AAA... user@laptop` are **different** keys (different comment).
- `ssh-ed25519 AAA... user@host` and `ssh-ed25519 AAA... user@host` are **identical** (whitespace is trimmed).

This simple approach avoids parsing complexity. Duplicate SSH keys with different comments do not cause SSH failures—they simply grant the same access twice, which is harmless but redundant.

#### Deduplication Rules

1. **First occurrence wins:** If a line appears in multiple sources, it is attributed to the **first source** (in configuration order) where it was found.
2. **Cross-source deduplication:** A line appearing in Source A and Source B is only listed once, under Source A.
3. **Local deduplication:** If a local line also exists in a remote source, the remote source takes precedence (the line is listed under the remote source, not under "Local").
4. **Intra-file deduplication:** Duplicate lines within the same source or local file are reduced to a single entry.

#### Logging

Deduplication events are logged to stdout for auditability. The generated `authorized_keys` file does **not** contain deduplication metadata—it remains clean and human-readable.

### 3.4 Output Format

The generated `authorized_keys` file follows a structured, auditable format.

#### File Structure

```
# ──────────────────────────────────────────────────────────────────
# Generated by AuthKeySync
# Last sync: <UTC timestamp>
# ──────────────────────────────────────────────────────────────────

# Source: <url-1>
<key-1>
<key-2>

# Source: <url-2>
<key-3>

# Local (preserved)
<key-4>
```

#### Section Order

1. **Header:** Metadata block with generation timestamp.
2. **Remote Sources:** One section per source URL, in the order defined in the configuration file. Only keys attributed to that source (after deduplication) are listed.
3. **Local Section:** Preserved local keys (only present if `preserve_local_keys=true`). Contains keys that existed in the previous `authorized_keys` file but were not found in any remote source.

#### Empty Sections

If a source yields zero keys (after deduplication), its section header is **omitted** entirely. If no local keys are preserved, the "Local (preserved)" section is omitted.

### 3.5 The Atomic Write Procedure

To prevent data corruption during power loss or system crashes, file writes are strictly atomic.

1. **Resolve Paths:** Target is `~/.ssh/authorized_keys`, resolved from the user's home directory (e.g., `/root/.ssh/authorized_keys` for root, `/home/bob/.ssh/authorized_keys` for bob).
2. **Temp File:** Create a temporary file **inside** the user's `.ssh/` directory (e.g., `~/.ssh/.authkeysync_<YYYYMMDD_HHMMSS>_<randomID>`).
   - _Constraint:_ Must be on the same filesystem partition to allow atomic `rename`.
3. **Permissions (Security Critical):**
   - Immediately execute `chmod 0600` on the temp file.
   - **Result:** Owner: RW, Group: None, Others: None.
4. **Ownership Hygiene:**
   - Resolve Target User UID and **Primary GID** from the system.
   - Execute `chown UID:GID` on the temp file.
5. **Content Flush:** Write key data and execute `fsync()` to force physical disk write.
6. **Atomic Swap:** Execute `os.Rename(temp, target)`.

### 3.6 Exit Codes

The binary communicates its status to the OS scheduler.

| Exit Code | Meaning                                                                                                                                                                                                            |
| :-------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `0`       | Success. All users were processed successfully (or skipped due to non-critical warnings such as missing user or `.ssh` directory).                                                                                 |
| `1`       | Total or partial failure. At least one user failed to synchronize due to network or I/O errors. This signals the scheduler (Systemd/Cron) to log a failure event. Some users may have been processed successfully. |

## 4. Backups

Backups are performed locally within the user's `.ssh` directory to ensure permissions are inherited correctly.

| Property      | Value                                                               |
| :------------ | :------------------------------------------------------------------ |
| **Directory** | `~/.ssh/authorized_keys_backups/` (created if missing, mode `0700`) |
| **Filename**  | `authorized_keys_<YYYYMMDD_HHMMSS>_<randomID>` (UTC timestamp)      |
| **Trigger**   | Only if content has changed **and** `backup_enabled=true`           |
| **Retention** | Controlled by `backup_retention_count`. Oldest files deleted first. |

## 5. Development Requirements

| Requirement     | Specification                                                                      |
| :-------------- | :--------------------------------------------------------------------------------- |
| **Language**    | Go (Golang)                                                                        |
| **Compilation** | Statically linked, `CGO_ENABLED=0`, no runtime dependencies                        |
| **Binary Size** | As small as reasonably achievable                                                  |
| **Logging**     | Structured logging to `stdout`, compatible with Journald and other log aggregators |
| **Testing**     | Rigorous unit and integration tests covering all edge cases (see below)            |

### 5.1 Testing Requirements

Given the security-critical nature of this tool (incorrect behavior can lock users out of systems), comprehensive testing is **mandatory**.

| Test Type             | Scope                                                                                                                                                  |
| :-------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Unit Tests**        | All parsing logic, deduplication, file path resolution, permission handling, backup rotation                                                           |
| **Integration Tests** | End-to-end sync cycles, atomic write verification, multi-user scenarios, error recovery                                                                |
| **Edge Cases**        | Empty sources, malformed input, HTML error pages, network timeouts, permission errors, missing users, missing `.ssh` directories, concurrent execution |

All tests must pass before any release. Test coverage should prioritize correctness over percentage metrics.

### 5.2 Random ID Generation

Random identifiers are used for temporary files and backup filenames to prevent collisions.

| Property      | Value                        |
| :------------ | :--------------------------- |
| **Algorithm** | NanoID                       |
| **Alphabet**  | `abcdefghijklmnopqrstuvwxyz` |
| **Length**    | 6 characters                 |

This configuration yields 26⁶ = 308,915,776 possible combinations, which is sufficient to prevent collisions in the context of file naming (typically a handful of files per user). The lowercase-only alphabet ensures compatibility with case-insensitive filesystems.
