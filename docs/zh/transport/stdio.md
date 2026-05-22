# Stdio 传输层

Stdio transport 从 stdin 读取、向 stdout 写入。这是 Dolphin 在终端中交互运行时的默认传输层。

## 配置

```yaml
transport:
  stdio:
    enabled: true
    markdown_render: true
    markdown_style: dracula
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `true` | 启用 stdio transport |
| `markdown_render` | bool | `true` | 用 rich 格式渲染 Markdown 输出 |
| `markdown_style` | string | `"dracula"` | Glamour Markdown 主题 (https://github.com/charmbracelet/glamour) |

## 使用方式

```bash
dolphin
```

Stdio transport 默认自动启动，进入交互式 REPL。

## 斜杠命令

| 命令 | 说明 |
|------|------|
| `/new` | 开始新会话 |
| `/reset` | 重置到干净状态 |
| `/context` | 显示上下文摘要 |
| `/config` | 查看或修改配置 |
| `/help` | 帮助 |
| `/exit` | 退出 |
| `Ctrl+C` | 中断/退出 |

---

> 最后更新: 2026-05-22
