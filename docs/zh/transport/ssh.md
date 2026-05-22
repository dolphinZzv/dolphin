# SSH 传输层

SSH transport 将 Dolphin 暴露为一个 SSH 服务器。客户端可通过任意 SSH 客户端（`ssh`、PuTTY 等）连接并通过 Shell 与 Agent 交互。

## 配置

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

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用 SSH transport |
| `addr` | string | `":2222"` | SSH 监听地址 |
| `host_key` | string | `"~/.dolphin/ssh_host_key"` | 主机私钥路径 |
| `username` | string | `"dolphin"` | 用户名 |
| `password` | string | `""` | 密码（空 = 禁用密码认证） |
| `markdown_render` | bool | `true` | 渲染 Markdown 输出 |
| `markdown_style` | string | `"dracula"` | Glamour Markdown 主题 |

## 使用方式

### 1. 启用配置

```yaml
transport:
  ssh:
    enabled: true
    addr: ":2222"
```

### 2. 启动 Dolphin

```bash
dolphin
```

### 3. 通过 SSH 连接

```bash
ssh dolphin@localhost -p 2222
```

如果已配置密码，会提示输入。

## 安全说明

- 主机密钥在指定路径不存在时会自动生成
- `password` 为空时密码认证自动禁用
- 建议修改默认端口（`:2222`）以避免自动扫描攻击

---

> 最后更新: 2026-05-22
