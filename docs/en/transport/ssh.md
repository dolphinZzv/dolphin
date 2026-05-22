# SSH Transport

The SSH transport exposes Dolphin as an SSH server. Clients connect via any SSH client (`ssh`, PuTTY, etc.) and interact with the agent through a shell interface.

## Configuration

```yaml
transport:
  ssh:
    enabled: false
    addr: ":2222"
    host_key: ~/.dolphin/ssh_host_key
    username: dolphin
    password: ""
    markdown_render: true
    markdown_style: dracula
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable SSH transport |
| `addr` | string | `":2222"` | SSH listen address |
| `host_key` | string | `"~/.dolphin/ssh_host_key"` | Host private key path |
| `username` | string | `"dolphin"` | Username for password auth |
| `password` | string | `""` | Password for auth. Empty = password auth disabled |
| `markdown_render` | bool | `true` | Render markdown in responses |
| `markdown_style` | string | `"dracula"` | Glamour markdown style |

## Usage

### 1. Enable in config

```yaml
transport:
  ssh:
    enabled: true
    addr: ":2222"
```

### 2. Start Dolphin

```bash
dolphin
```

### 3. Connect via SSH

```bash
ssh dolphin@localhost -p 2222
```

If a password is configured, you will be prompted to enter it.

## Security Notes

- The host key is auto-generated if the configured path does not exist
- Password authentication is disabled when `password` is empty
- Consider using SSH key authentication as a more secure alternative
- Change the default port (`:2222`) to avoid automated scanning

---

> Last modified: 2026-05-22
