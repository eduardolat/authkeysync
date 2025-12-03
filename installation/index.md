# Installation

AuthKeySync is distributed as a single static binary with no external dependencies. Just download and run.

## Download

Download the appropriate binary for your system from the [GitHub Releases](https://github.com/eduardolat/authkeysync/releases) page.

### Linux (AMD64)

```
curl -Lo authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-amd64
chmod +x authkeysync
sudo mv authkeysync /usr/local/bin/
```

### Linux (ARM64)

```
curl -Lo authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-linux-arm64
chmod +x authkeysync
sudo mv authkeysync /usr/local/bin/
```

### macOS (Intel)

```
curl -Lo authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-darwin-amd64
chmod +x authkeysync
sudo mv authkeysync /usr/local/bin/
```

### macOS (Apple Silicon)

```
curl -Lo authkeysync https://github.com/eduardolat/authkeysync/releases/latest/download/authkeysync-darwin-arm64
chmod +x authkeysync
sudo mv authkeysync /usr/local/bin/
```

## Verify Installation

```
authkeysync --version
```

You should see output like:

```
    _         _   _     _  __          ____                   
   / \  _   _| |_| |__ | |/ /___ _   _/ ___| _   _ _ __   ___ 
  / _ \| | | | __| '_ \| ' // _ \ | | \___ \| | | | '_ \ / __|
 / ___ \ |_| | |_| | | | . \  __/ |_| |___) | |_| | | | | (__ 
/_/   \_\__,_|\__|_| |_|_|\_\___|\__, |____/ \__, |_| |_|\___|
                                 |___/       |___/            

Version: v0.0.0
Commit:  abc1234
Built:   ISO 8601 timestamp (UTC)
```

## Platform Support

- **Operating System**: Linux or macOS
- **Architecture**: AMD64 (x86_64) or ARM64 (aarch64)
- **Permissions**: Root access required to modify other users' `authorized_keys`

For each configured user, AuthKeySync requires the user to exist and have a `~/.ssh` directory. If a user or their `.ssh` directory doesn't exist, AuthKeySync will skip that user and continue with the next one.

## Create Configuration Directory

AuthKeySync expects its configuration at `/etc/authkeysync/config.yaml` by default:

```
sudo mkdir -p /etc/authkeysync
```

You can use a different config path with the `--config` flag:

```
authkeysync --config /path/to/your/config.yaml
```

## Next Steps

- [Configuration](https://eduardolat.github.io/authkeysync/configuration/index.md): Set up your key sources
- [Usage](https://eduardolat.github.io/authkeysync/usage/index.md): Run and automate AuthKeySync
