# Email Transport

The Email transport allows interaction with Dolphin via email. It polls an inbox (IMAP or POP3) for incoming messages, processes them, and replies via SMTP.

## Configuration

```yaml
transport:
  email:
    enabled: false
    protocol: imap              # "imap" or "pop3"
    smtp_host: smtp.gmail.com
    smtp_port: 587
    imap_host: imap.gmail.com
    imap_port: 993
    pop3_host: ""               # defaults to imap_host
    pop3_port: 995
    username: ""
    password: ""
    from: ""
    use_tls: true
    skip_tls_verify: false
    poll_interval: 10s
    allowed_senders: []
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Email transport |
| `protocol` | string | `"imap"` | Inbox protocol: `"imap"` or `"pop3"` |
| `smtp_host` | string | — | SMTP server hostname |
| `smtp_port` | int | `587` | SMTP server port |
| `imap_host` | string | — | IMAP server hostname |
| `imap_port` | int | `993` | IMAP server port |
| `pop3_host` | string | — | POP3 server hostname (defaults to IMAP host) |
| `pop3_port` | int | `995` | POP3 server port |
| `username` | string | — | Email account username |
| `password` | string | — | Email account password or app password |
| `from` | string | — | From address for replies |
| `use_tls` | bool | `true` | Use TLS for SMTP |
| `skip_tls_verify` | bool | `false` | Skip TLS certificate verification |
| `poll_interval` | string | `"10s"` | How often to poll for new messages |
| `allowed_senders` | []string | `[]` | Only process emails from these addresses (empty = allow all) |

## Usage

### 1. Enable in config

**Gmail (international):**
```yaml
transport:
  email:
    enabled: true
    protocol: imap
    smtp_host: smtp.gmail.com
    smtp_port: 587
    imap_host: imap.gmail.com
    imap_port: 993
    username: your-email@gmail.com
    password: your-app-password
    from: your-email@gmail.com
    poll_interval: 10s
```

> For Gmail, use an [App Password](https://support.google.com/accounts/answer/185833) instead of your main password.

**QQ Mail (China):**
```yaml
transport:
  email:
    enabled: true
    protocol: imap
    smtp_host: smtp.qq.com
    smtp_port: 587
    imap_host: imap.qq.com
    imap_port: 993
    username: your-email@qq.com
    password: your-auth-code
    from: your-email@qq.com
    poll_interval: 10s
```

> For QQ Mail, use an [Authorization Code](https://service.mail.qq.com/detail/0/75) (开启 IMAP/SMTP 后获取) instead of your login password.

### 2. Start Dolphin

```bash
dolphin
```

### 3. Send an email

Send an email to the configured inbox address. Dolphin will:
  1. Poll the inbox at the configured `poll_interval`
  2. Process the message content as a task
  3. Reply via SMTP with the response

## Security Notes

- Use [App Passwords](https://support.google.com/accounts/answer/185833) for Gmail; never use your main account password
- Set `allowed_senders` to restrict which addresses can interact with the agent
- Enable `use_tls` for all production email servers
- `skip_tls_verify` should only be used for testing with self-signed certs

---

> Last modified: 2026-05-22
