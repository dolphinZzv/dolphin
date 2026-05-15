package agent

import "dolphin/internal/metrics"

// Agent-level metrics collected via the global metrics registry.
var (
	// LLM provider metrics (labeled by provider name)
	llmRequests     = metrics.NewLabeledCounter("llm_requests_total", "Total LLM API requests", "provider", nil)
	llmErrors       = metrics.NewLabeledCounter("llm_errors_total", "Total LLM API errors", "provider", nil)
	llmDuration     = metrics.NewLabeledHistogram("llm_request_duration_seconds", "LLM request duration", "provider", nil, nil)
	llmInputTokens  = metrics.NewLabeledCounter("llm_input_tokens_total", "Total LLM input tokens", "provider", nil)
	llmOutputTokens = metrics.NewLabeledCounter("llm_output_tokens_total", "Total LLM output tokens", "provider", nil)

	// Task metrics
	taskDispatched = metrics.NewCounter("agent_tasks_dispatched_total", "Total tasks dispatched to sub-agents", map[string]string{})
	taskCompleted  = metrics.NewCounter("agent_tasks_completed_total", "Total tasks completed by sub-agents", map[string]string{})
	taskFailed     = metrics.NewCounter("agent_tasks_failed_total", "Total sub-agent task failures", map[string]string{})

	// Pool gauge (updated by pool lifecycle)
	agentPoolSize = metrics.NewGauge("agent_pool_size", "Current number of registered agents in pool", map[string]string{})
	activeAgents  = metrics.NewGauge("agent_active_agents", "Current number of active (busy) agents", map[string]string{})
)
