# Metrics (`internal/metrics/` — v0.3)

## Types

- `Counter` — 累计值 (llm_requests_total)
- `Gauge` — 即时值 (agent_pool_size)
- `Histogram` — 分布 (llm_request_duration_seconds)

## Hierarchy

| Level | Metrics | Location |
|-------|---------|----------|
| Agent | llm_requests/errors/duration, input/output tokens, tasks*, pool size | `agent/metrics.go` |
| MCP | tool_calls/errors/duration | `mcp/registry.go` |
| Transport | connections, messages | `transport/metrics.go` |

## Export

`metrics/http.go` — `/metrics` HTTP handler (Prometheus text format)
