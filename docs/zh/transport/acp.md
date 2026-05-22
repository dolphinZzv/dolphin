# ACP 传输层 — IBM BeeAI Agent Communication Protocol

ACP transport 实现了 [IBM BeeAI Agent Communication Protocol (ACP)](https://github.com/i-am-bee/beeai-platform-specifications)，使用 REST over HTTP。它提供标准的任务管理 API，用于 Agent 间的通信。

> **当前状态**: v1 — 支持基本的任务生命周期管理（同步/异步）。

## 配置

```yaml
transport:
  acp:
    enabled: true
    listen_addr: ":8333"
    agent_id: dolphin
    agent_name: Dolphin AI Agent
    agent_version: "0.1.0"
    agent_description: "Cross-terminal/email/chat/SSH AI agent"
    capabilities:
      - task-execution
      - shell-command
      - web-search
    sync_timeout: 60s
    api_key: ""
    tls_enabled: false
    tls_cert_file: ""
    tls_key_file: ""
    peers: []
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用 ACP transport |
| `listen_addr` | string | `":8333"` | HTTP 监听地址 |
| `agent_id` | string | `"dolphin"` | 唯一 Agent 标识 |
| `agent_name` | string | `"Dolphin AI Agent"` | 人类可读的 Agent 名称 |
| `agent_version` | string | `"0.1.0"` | Agent 版本 |
| `agent_description` | string | — | Agent 描述 |
| `capabilities` | []string | — | 能力列表 |
| `sync_timeout` | string | `"60s"` | 同步任务最大等待时间 |
| `api_key` | string | `""` | Bearer token 鉴权。空 = 不鉴权 |
| `tls_enabled` | bool | `false` | 启用 TLS |
| `tls_cert_file` | string | `""` | TLS 证书文件路径 |
| `tls_key_file` | string | `""` | TLS 密钥文件路径 |
| `peers` | []Peer | `[]` | 已知对等体 Agent（预留） |

## 使用方式

### 1. 启用配置

```yaml
transport:
  acp:
    enabled: true
    listen_addr: ":8333"
```

### 2. 启动 Dolphin

```bash
dolphin
```

### 3. 查看能力

```bash
curl http://localhost:8333/capabilities
```

### 4. 发送任务（同步）

```bash
curl -X POST http://localhost:8333/tasks \
  -H "Content-Type: application/json" \
  -d '{"id":"task-001","task":"几点了?"}'
```

### 5. 查询或取消任务（可选）

```bash
# 列出所有任务
curl http://localhost:8333/tasks

# 查询特定任务
curl http://localhost:8333/tasks/task-001

# 取消运行中的任务
curl -X DELETE http://localhost:8333/tasks/task-001
```

## API 端点

### `GET /capabilities`

返回 Agent 元数据和能力。

### `POST /tasks`

创建并执行新任务。

**同步模式**（默认）：
```bash
curl -X POST http://localhost:8333/tasks \
  -H "Content-Type: application/json" \
  -d '{"id":"task-001","task":"几点了?"}'
```

**异步模式**（添加 `Prefer: respond-async` 请求头）：
```bash
curl -X POST http://localhost:8333/tasks \
  -H "Content-Type: application/json" \
  -H "Prefer: respond-async" \
  -d '{"id":"task-001","task":"几点了?"}'
```
返回 `202 Accepted`，然后通过 `GET /tasks/{id}` 轮询结果。

### `GET /tasks`

列出所有任务。

### `GET /tasks/{id}`

查询任务状态和结果。

### `DELETE /tasks/{id}`

取消运行中的任务。

### 请求格式

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `id` | string | 否 | 任务 ID（空则自动生成） |
| `agentId` | string | 否 | 目标 Agent 标识 |
| `sessionId` | string | 否 | 多轮会话标识 |
| `task` | string | 是 | 任务描述/指令 |
| `context` | string | 否 | 额外上下文 |
| `metadata` | object | 否 | 自定义键值对 |

## 鉴权

配置 `api_key` 后，客户端需在请求头中携带：

```
Authorization: Bearer <api_key>
```

`api_key` 为空时所有请求直接放行，不进行鉴权。

## 任务状态

```
pending → running → completed
                → failed
                → cancelled
```

## 参考

- [A2A 传输层](a2a.md) — Google Agent-to-Agent 协议
- [IBM BeeAI ACP 规范](https://github.com/i-am-bee/beeai-platform-specifications)

---

> 最后更新: 2026-05-22
