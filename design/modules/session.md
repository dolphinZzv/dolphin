# Session Management (`internal/session/`)

## Format

每会话一个 `{dir}/{uuid}.jsonl`，JSONL 格式每行一个 Event:

| Event Type | Content |
|------------|---------|
| `message` | 用户/助手消息 |
| `tool_call` | LLM 请求的工具调用 |
| `tool_result` | 工具执行结果 |
| `system` | turn start/end, 错误 |
| `summary` | 会话摘要 |

## Manager

- `NewManager(dir)` → `EnsureDir()` → 多会话生命周期管理
- `Reaper` — 后台定时清理过期会话 (`max_age`)
- `Summary` — 到达 MaxLoop 时生成摘要，支持 Resume (从摘要续对话)
