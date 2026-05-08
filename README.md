# DolphinzZ

AI coding agent with MCP tool support. Runs via stdio, SSH, or MQTT.

## Quick Start

```bash
# Build
make build

# Set your API key and run
export DZ_LLM_API_KEY="sk-..."
./dolphinzZ
```

## Configuration

Priority (higher overrides lower):
1. Environment variables (`DZ_*`)
2. Project: `.dolphinzZ/config.yaml`
3. User: `~/.dolphinzZ/config.yaml`
4. System: `/etc/dolphinzZ/config.yaml`

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DZ_LLM_API_KEY` | — | API key |
| `DZ_LLM_TYPE` | `openai` | `openai` or `anthropic` |
| `DZ_LLM_MODEL` | `gpt-4o` | Model name |
| `DZ_LLM_BASE_URL` | `https://api.openai.com/v1` | API base URL |
| `DZ_LLM_MAX_TOKENS` | `4096` | Max output tokens |
| `DZ_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

### Example config (`.dolphinzZ/config.yaml`)

```yaml
llm:
  type: "anthropic"
  base_url: "https://api.anthropic.com"
  api_key: ""
  model: "claude-sonnet-4-20250514"
  max_tokens: 4096

session:
  dir: "./sessions"
  max_loop: 50
  max_age: "24h"

transport:
  stdio:
    enabled: true
  ssh:
    enabled: false
  mqtt:
    enabled: false

mcp:
  shell:
    enabled: true
    allowed_commands: []
    timeout_seconds: 30
  cdp:
    enabled: true
    headless: true
```

## Usage

### stdio (default)

```bash
./dolphinzZ
```

Type your message at the `>` prompt. Built-in commands:

- `/exit`, `/quit` — end session
- `/help` — show help

### SSH

Enable in config, then connect:

```bash
ssh dolphinzZ@<host> -p 2222
```

### MQTT

Enable in config. Subscribe to `dolphinzZ/agent/response` and publish to `dolphinzZ/agent/command`:

```bash
mosquitto_sub -t "dolphinzZ/agent/response" &
mosquitto_pub -t "dolphinzZ/agent/command" -m "your prompt"
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `shell` | Execute shell commands with timeout control |
| `cdp` | Browser automation via CDP (navigate, click, screenshot, evaluate JS) |
| External | Any stdio-based MCP server (configured via `mcp.servers`) |

## Development

```bash
make test    # run all tests
make build   # build binary
make fmt     # format code
make clean   # clean build artifacts
```

## Safety

- Shell commands are unrestricted by default (`allowed_commands: []`). Set explicit allowlist for production use.
- SSH password is stored in plaintext at `~/.dolphinzZ/ssh_password`. Use SSH key authentication for better security.
- Session files are retained for 24 hours by default (`session.max_age`). Old files are cleaned up automatically.
