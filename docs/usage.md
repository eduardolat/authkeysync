# Usage

This guide covers how to run AuthKeySync and automate it for continuous synchronization.

## CLI Options

```bash
authkeysync [options]
```

| Option            | Description                                                   |
| ----------------- | ------------------------------------------------------------- |
| `--config <path>` | Path to config file (default: `/etc/authkeysync/config.yaml`) |
| `--dry-run`       | Simulate sync without modifying any files                     |
| `--version`       | Show version information and exit                             |
| `--help`          | Show help message                                             |

## Basic Usage

### Run with Default Config

```bash
sudo authkeysync
```

This reads `/etc/authkeysync/config.yaml` and syncs all configured users.

### Use Custom Config Path

```bash
sudo authkeysync --config /path/to/config.yaml
```

### Dry Run (Preview Changes)

The `--dry-run` flag simulates the sync without making any changes:

```bash
sudo authkeysync --dry-run
```

This is useful for:

- Testing a new configuration
- Verifying source URLs are accessible
- Previewing what would be written

## Exit Codes

AuthKeySync uses exit codes to indicate success or failure:

| Exit Code | Meaning                                                                      |
| --------- | ---------------------------------------------------------------------------- |
| `0`       | Success: all users processed (or skipped due to missing user/ssh dir)        |
| `1`       | Failure: at least one user failed to sync (network error, write error, etc.) |

Use these codes for monitoring and alerting.

## Automation

### Cron Job

Create a cron job to run AuthKeySync periodically:

```bash
# Run every 5 minutes
echo "*/5 * * * * root /usr/local/bin/authkeysync" | sudo tee /etc/cron.d/authkeysync
```

Or edit root's crontab directly:

```bash
sudo crontab -e
```

Add:

```cron
# Sync SSH keys every 5 minutes
*/5 * * * * /usr/local/bin/authkeysync >> /var/log/authkeysync.log 2>&1
```

### Systemd Timer

For systems using systemd, create a timer unit:

**`/etc/systemd/system/authkeysync.service`**

```ini
[Unit]
Description=AuthKeySync SSH Key Synchronization
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/authkeysync
StandardOutput=journal
StandardError=journal
```

**`/etc/systemd/system/authkeysync.timer`**

```ini
[Unit]
Description=Run AuthKeySync every 5 minutes

[Timer]
OnBootSec=1min
OnUnitActiveSec=5min

[Install]
WantedBy=timers.target
```

Enable the timer:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now authkeysync.timer
```

Check status:

```bash
sudo systemctl status authkeysync.timer
sudo systemctl list-timers | grep authkeysync
```

View logs:

```bash
sudo journalctl -u authkeysync.service
```

### Cloud-Init

For cloud instances, include AuthKeySync in your cloud-init configuration:

```yaml
#cloud-config
write_files:
  - path: /etc/authkeysync/config.yaml
    permissions: "0600"
    content: |
      policy:
        backup_enabled: true
        preserve_local_keys: true
      users:
        - username: "root"
          sources:
            - url: "https://github.com/your-username.keys"

runcmd:
  # Download AuthKeySync
  - curl -Lo /usr/local/bin/authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-amd64
  - chmod +x /usr/local/bin/authkeysync

  # Run initial sync
  - /usr/local/bin/authkeysync

  # Setup cron job
  - echo "*/5 * * * * root /usr/local/bin/authkeysync" > /etc/cron.d/authkeysync
```

### Ansible

```yaml
- name: Install AuthKeySync
  hosts: all
  become: yes
  tasks:
    - name: Download AuthKeySync binary
      get_url:
        url: https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-amd64
        dest: /usr/local/bin/authkeysync
        mode: "0755"

    - name: Create config directory
      file:
        path: /etc/authkeysync
        state: directory
        mode: "0755"

    - name: Deploy configuration
      copy:
        dest: /etc/authkeysync/config.yaml
        mode: "0600"
        content: |
          policy:
            backup_enabled: true
            preserve_local_keys: true
          users:
            - username: "root"
              sources:
                - url: "https://github.com/your-username.keys"

    - name: Setup cron job
      cron:
        name: "AuthKeySync"
        minute: "*/5"
        job: "/usr/local/bin/authkeysync"
        user: root

    - name: Run initial sync
      command: /usr/local/bin/authkeysync
```

### Terraform (with user_data)

```hcl
resource "aws_instance" "web" {
  ami           = "ami-0123456789"
  instance_type = "t3.micro"

  user_data = <<-EOF
    #!/bin/bash
    curl -Lo /usr/local/bin/authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-amd64
    chmod +x /usr/local/bin/authkeysync

    mkdir -p /etc/authkeysync
    cat > /etc/authkeysync/config.yaml <<EOC
    policy:
      backup_enabled: true
      preserve_local_keys: true
    users:
      - username: "root"
        sources:
          - url: "https://github.com/your-username.keys"
    EOC

    /usr/local/bin/authkeysync
    echo "*/5 * * * * root /usr/local/bin/authkeysync" > /etc/cron.d/authkeysync
  EOF
}
```

## Monitoring

### Log Output

AuthKeySync outputs structured logs to stdout, compatible with journald and log aggregators:

```
time=2024-01-15T10:30:45Z level=INFO msg="AuthKeySync starting" version=v1.0.0 config=/etc/authkeysync/config.yaml dry_run=false
time=2024-01-15T10:30:45Z level=INFO msg="configuration loaded" users=2 backup_enabled=true backup_retention=10 preserve_local_keys=true
time=2024-01-15T10:30:45Z level=INFO msg="processing user" username=root
time=2024-01-15T10:30:46Z level=INFO msg="fetched keys from source" username=root url=https://github.com/your-username.keys keys=2 discarded_lines=0
time=2024-01-15T10:30:46Z level=INFO msg="updated authorized_keys" username=root path=/root/.ssh/authorized_keys keys=2
time=2024-01-15T10:30:46Z level=INFO msg="synchronization complete" success=2 skipped=0 failed=0
time=2024-01-15T10:30:46Z level=INFO msg="all users processed successfully"
```

### Health Checks

For monitoring systems, check:

1. **Exit code**: `0` means success, `1` means failure
2. **Log messages**: Look for `level=ERROR` entries
3. **File modification time**: Check if `authorized_keys` is being updated

Example monitoring script:

```bash
#!/bin/bash
if ! /usr/local/bin/authkeysync 2>&1; then
    echo "CRITICAL: AuthKeySync failed"
    exit 2
fi
echo "OK: AuthKeySync completed successfully"
exit 0
```

## Troubleshooting

### User Not Found

```
level=WARN msg="user not found in system, skipping" username=deploy
```

**Solution**: Create the user or fix the username in config.

### SSH Directory Missing

```
level=WARN msg=".ssh directory not found, skipping" username=deploy
```

**Solution**: Create the `.ssh` directory:

```bash
sudo mkdir -p /home/deploy/.ssh
sudo chown deploy:deploy /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh
```

### Network Errors

```
level=ERROR msg="failed to fetch keys, aborting user sync" username=root error="source https://github.com/your-username.keys failed: request failed: ..."
```

**Solutions**:

- Check network connectivity
- Verify the URL is correct
- Check firewall rules
- For private APIs, verify authentication headers

### Permission Denied

```
level=ERROR msg="failed to write authorized_keys" username=deploy error="permission denied"
```

**Solution**: Run AuthKeySync as root or with appropriate permissions.

## Next Steps

- [Configuration](configuration.md): Detailed configuration options
- [Technical Specification](spec.md): Deep dive into internals
