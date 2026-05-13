# Multi-Agent Coordination (`internal/agent/` — v0.2)

## Architecture

```
Coordinator (per-connection)
  ├── Agent (LLM + Session + cloned MCP Registry)
  ├── AgentPool
  │     ├── AgentInstance "reviewer"    (持久化)
  │     ├── AgentInstance "sysadmin"    (持久化)
  │     └── AgentInstance temp-xxx      (临时, 可回收)
  └── background: cron dueCh listener
```

## AgentPool

- `taskCh / resultCh` — 异步信道通信
- `chan struct{}` 信号量控制最大并发
- Goroutine worker loop + 闲置超时回收
- `Collect()` 批量收集已完成结果

## AgentInstance

状态机: `idle → busy → completed/error/cancelled/timeout → idle`

## Coordinator MCP Tools

`dispatch_task` / `create_agent` / `get_agent_status` / `cancel_task` / `delete_agent`

## ChannelIO

`agent/channel_io.go` — 内存 UserIO 实现，通过 channel 在 Coordinator 与子 Agent 间通信。
