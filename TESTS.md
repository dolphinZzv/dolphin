# Tests

## Unit Tests

Run: `make test` (`go test -race -count=1 ./...`)

| Package | Tests | Time | Coverage | File |
|---------|-------|------|----------|------|
| agent | AgentDef, AgentPool, ChannelIO, Compressor, Config, Coordinator, Loop, OpenAI, E2E, Summary, Turn | 5.244s | 52.2% | [agent/](internal/agent/) |
| command | Parse, Manager, Load, MultiDir, Usage, Reload | 1.101s | 96.9% | [command/](internal/command/) |
| config | ConfigGen, Career, EnvOverrides, ProjectDetect, RepoFetcher, SSH, Merge | 1.513s | 46.4% | [config/](internal/config/) |
| context | Builder, LoadFiles, Cache | 1.060s | 81.5% | [context/](internal/context/) |
| diary | Sync, Prune, Lock, AtomicWrite, Summary | 2.291s | 78.2% | [diary/](internal/diary/) |
| event | Emit, Subscribe, Webhook, LogWriter | 2.463s | 84.9% | [event/](internal/event/) |
| hook | Registry, Priority, Abort, Rewrite | 1.088s | 100.0% | [hook/](internal/hook/) |
| mcp | Shell, CDP, Email, Webhook, Registry, SSE, Client | 6.613s | 51.1% | [mcp/](internal/mcp/) |
| metrics | Counter, Gauge, Histogram, Timer, Render | 1.051s | 81.6% | [metrics/](internal/metrics/) |
| plugin | Manager, LoadScripts, HookScript | 1.070s | 75.3% | [plugin/](internal/plugin/) |
| scheduler | Parse, AddTask, RemoveTask, Cron, Due, Persist | 1.474s | 88.2% | [scheduler/](internal/scheduler/) |
| session | Manager, LogMessage, Summary, ReadEvents | 1.166s | 68.1% | [session/](internal/session/) |
| skill | Parse, Manager, Search, TopSkills | 1.050s | 67.2% | [skill/](internal/skill/) |
| transport | Stdio, SSH, MQTT, Email, EmbeddedBroker | 1.681s | 45.6% | [transport/](internal/transport/) |
| **Total** | **~270 tests** | **~29s** | **52.2%** | |

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
