package provider

import "dolphin/internal/metrics"

// Provider-level metrics collected via the global metrics registry.
var (
	llmRequests     = metrics.NewLabeledCounter("llm_requests_total", "Total LLM API requests", "provider", nil)
	llmErrors       = metrics.NewLabeledCounter("llm_errors_total", "Total LLM API errors", "provider", nil)
	llmDuration     = metrics.NewLabeledHistogram("llm_request_duration_seconds", "LLM request duration", "provider", nil, nil)
	llmInputTokens  = metrics.NewLabeledCounter("llm_input_tokens_total", "Total LLM input tokens", "provider", nil)
	llmOutputTokens = metrics.NewLabeledCounter("llm_output_tokens_total", "Total LLM output tokens", "provider", nil)
)
