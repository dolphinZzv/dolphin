# Tests

## Unit Tests

Run: `make test` (`go test -race -count=1 ./...`)

| Package | Tests | Time | File |
|---------|-------|------|------|
| agent | AgentDef, AgentPool, ChannelIO, Compressor, Config, Coordinator, Loop, OpenAI, E2E, Summary, Turn | 4.047s | [agent/](internal/agent/) |
| command | Parse, Manager, Load, MultiDir, Usage, Reload | 0.029s | [command/](internal/command/) |
| config | ConfigGen, Career, EnvOverrides, ProjectDetect, RepoFetcher, SSH, Merge | 0.091s | [config/](internal/config/) |
| context | Builder, LoadFiles, Cache | 0.025s | [context/](internal/context/) |
| diary | Sync, Prune, Lock, AtomicWrite, Summary | 1.061s | [diary/](internal/diary/) |
| event | Emit, Subscribe, Webhook, LogWriter | 1.438s | [event/](internal/event/) |
| hook | Registry, Priority, Abort, Rewrite | 0.008s | [hook/](internal/hook/) |
| mcp | Shell, CDP, Email, Webhook, Registry, SSE, Client | 5.314s | [mcp/](internal/mcp/) |
| metrics | Counter, Gauge, Histogram, Timer, Render | 0.010s | [metrics/](internal/metrics/) |
| plugin | Manager, LoadScripts, HookScript | 0.083s | [plugin/](internal/plugin/) |
| scheduler | Parse, AddTask, RemoveTask, Cron, Due, Persist | 0.428s | [scheduler/](internal/scheduler/) |
| session | Manager, LogMessage, Summary, ReadEvents | 0.038s | [session/](internal/session/) |
| skill | Parse, Manager, Search, TopSkills | 0.023s | [skill/](internal/skill/) |
| transport | Stdio, SSH, MQTT, Email, EmbeddedBroker | 0.556s | [transport/](internal/transport/) |
| **Total** | **~270 tests** | **~13s** | |

## Smoke Tests

| Test | Time | Scenario | File |
|------|------|----------|------|
| LLM valid key | ~3.6s | Send "abc 第一个字是什么？只回答一个字", verify LLM returns "a" | [scripts/llm-smoke.sh](scripts/llm-smoke.sh) |
| LLM invalid key | ~5s | Send request with bad API key, verify auth error message | [scripts/llm-smoke.sh](scripts/llm-smoke.sh) |

## CI Workflows

| Workflow | Trigger | Steps | File |
|----------|---------|-------|------|
| CI | push (main/tags), PR | fmt → vet → build → test → coverage → docker release | [ci.yml](.github/workflows/ci.yml) |
| LLM Smoke Test | push (main/tags), PR, manual | build → valid key test → invalid key test | [llm-smoke.yml](.github/workflows/llm-smoke.yml) |
