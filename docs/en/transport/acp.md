# ACP Transport — IBM BeeAI Agent Communication Protocol

The ACP transport implements the [IBM BeeAI Agent Communication Protocol (ACP)](https://github.com/i-am-bee/beeai-platform-specifications) using REST over HTTP. It exposes a standard task management API for inter-agent communication.

> **Status**: v1 — basic task lifecycle with sync/async modes.

## Configuration

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

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable ACP transport |
| `listen_addr` | string | `":8333"` | HTTP listen address |
| `agent_id` | string | `"dolphin"` | Unique agent identifier |
| `agent_name` | string | `"Dolphin AI Agent"` | Human-readable agent name |
| `agent_version` | string | `"0.1.0"` | Agent version |
| `agent_description` | string | — | Agent description |
| `capabilities` | []string | — | Capability list |
| `sync_timeout` | string | `"60s"` | Max wait time for synchronous task execution |
| `api_key` | string | `""` | Bearer token auth. Empty = auth disabled |
| `tls_enabled` | bool | `false` | Enable TLS |
| `tls_cert_file` | string | `""` | TLS cert file path |
| `tls_key_file` | string | `""` | TLS key file path |
| `peers` | []Peer | `[]` | Known peer agents (for future outbound use) |

## Usage

### 1. Enable in config

```yaml
transport:
  acp:
    enabled: true
    listen_addr: ":8333"
```

### 2. Start Dolphin

```bash
dolphin
```

### 3. View capabilities

```bash
curl http://localhost:8333/capabilities
```

### 4. Send a task (sync)

```bash
curl -X POST http://localhost:8333/tasks \
  -H "Content-Type: application/json" \
  -d '{"id":"task-001","task":"What time is it?"}'
```

### 5. Query or cancel (optional)

```bash
# List all tasks
curl http://localhost:8333/tasks

# Query a specific task
curl http://localhost:8333/tasks/task-001

# Cancel a running task
curl -X DELETE http://localhost:8333/tasks/task-001
```

## API Endpoints

### `GET /capabilities`

Returns agent metadata and capabilities.

```bash
curl http://localhost:8333/capabilities
```

```json
{
  "agentId": "dolphin",
  "name": "Dolphin AI Agent",
  "version": "0.1.0",
  "description": "Cross-terminal/email/chat/SSH AI agent",
  "capabilities": ["task-execution", "shell-command", "web-search"],
  "protocol": "acp",
  "protocolVersion": "0.1"
}
```

### `POST /tasks`

Create and execute a new task.

**Sync mode** (default):
```bash
curl -X POST http://localhost:8333/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "task-001",
    "task": "What time is it?"
  }'
```

**Async mode** (with `Prefer: respond-async` header):
```bash
curl -X POST http://localhost:8333/tasks \
  -H "Content-Type: application/json" \
  -H "Prefer: respond-async" \
  -d '{
    "id": "task-001",
    "task": "What time is it?"
  }'
```
Returns `202 Accepted` immediately, with the task status. Use `GET /tasks/{id}` to poll for completion.

### `GET /tasks`

List all tasks:

```bash
curl http://localhost:8333/tasks
```

### `GET /tasks/{id}`

Get task status and result:

```bash
curl http://localhost:8333/tasks/task-001
```

Response (completed):
```json
{
  "id": "task-001",
  "status": "completed",
  "output": {
    "result": "2026-05-22 10:24",
    "contentType": "text/plain"
  },
  "metadata": {
    "created": "2026-05-22T10:24:00+08:00",
    "completed": "2026-05-22T10:24:02+08:00"
  }
}
```

### `DELETE /tasks/{id}`

Cancel a running task:

```bash
curl -X DELETE http://localhost:8333/tasks/task-001
```

### Request Format

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `id` | string | no | Task ID (auto-generated if empty) |
| `agentId` | string | no | Target agent identifier |
| `sessionId` | string | no | Session identifier for multi-turn |
| `task` | string | yes | Task description / instruction |
| `context` | string | no | Additional context |
| `metadata` | object | no | Custom key-value pairs |

## Authentication

Configure `api_key` in the ACP config. Clients must then include:

```
Authorization: Bearer <api_key>
```

When `api_key` is empty, all requests are accepted without authentication.

## Task States

```
pending → running → completed
                → failed
                → cancelled
```

## Limitations (v1)

- Agent card (`GET /agents/{id}`) returns the same data as `/capabilities`
- In-memory task storage
- No streaming responses
- No outbound peer communication (planned v2)

## See Also

- [A2A Transport](a2a.md) — Google Agent-to-Agent protocol
- [IBM BeeAI ACP Specification](https://github.com/i-am-bee/beeai-platform-specifications)
- [Design Doc](../../design/modules/acp-transport.md)

---

> Last modified: 2026-05-22
