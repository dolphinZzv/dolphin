# Email 传输层

Email transport 允许通过电子邮件与 Dolphin 交互。它轮询收件箱（IMAP 或 POP3）获取新消息，处理后通过 SMTP 回复。

## 配置

```yaml
transport:
  email:
    enabled: false
    protocol: imap              # "imap" 或 "pop3"
    smtp_host: smtp.gmail.com
    smtp_port: 587
    imap_host: imap.gmail.com
    imap_port: 993
    pop3_host: ""               # 默认使用 imap_host
    pop3_port: 995
    username: ""
    password: ""
    from: ""
    use_tls: true
    skip_tls_verify: false
    poll_interval: 10s
    allowed_senders: []
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用 Email transport |
| `protocol` | string | `"imap"` | 收件箱协议：`"imap"` 或 `"pop3"` |
| `smtp_host` | string | — | SMTP 服务器地址 |
| `smtp_port` | int | `587` | SMTP 服务器端口 |
| `imap_host` | string | — | IMAP 服务器地址 |
| `imap_port` | int | `993` | IMAP 服务器端口 |
| `pop3_host` | string | — | POP3 服务器地址（默认用 IMAP 地址） |
| `pop3_port` | int | `995` | POP3 服务器端口 |
| `username` | string | — | 邮箱用户名 |
| `password` | string | — | 邮箱密码或应用密码 |
| `from` | string | — | 回复邮件时使用的发件地址 |
| `use_tls` | bool | `true` | 使用 TLS 连接 SMTP |
| `skip_tls_verify` | bool | `false` | 跳过 TLS 证书验证 |
| `poll_interval` | string | `"10s"` | 轮询新邮件的间隔 |
| `allowed_senders` | []string | `[]` | 仅处理这些地址的邮件（空 = 全部放行） |

## 使用方式

### 1. 启用配置

**Gmail（国外）：**
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

> Gmail 使用[应用专用密码](https://support.google.com/accounts/answer/185833)，不要使用主密码。

**QQ 邮箱（国内）：**
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

> QQ 邮箱使用[授权码](https://service.mail.qq.com/detail/0/75)（开启 IMAP/SMTP 后获取），不要使用登录密码。

### 2. 启动 Dolphin

```bash
dolphin
```

### 3. 发送邮件

向已配置的收件箱发送邮件，Dolphin 会：
  1. 按设定的 `poll_interval` 轮询收件箱
  2. 将邮件内容作为任务处理
  3. 通过 SMTP 回复结果

## 安全说明

- Gmail 使用[应用专用密码](https://support.google.com/accounts/answer/185833)；切勿使用主密码
- 设置 `allowed_senders` 限制可交互的发件人
- 生产环境务必开启 `use_tls`
- `skip_tls_verify` 仅用于自签名证书测试

---

> 最后更新: 2026-05-22
