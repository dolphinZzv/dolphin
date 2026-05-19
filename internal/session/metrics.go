package session

import "dolphin/internal/metrics"

// Session-level metrics for per-session token usage tracking.
// Exposed as Prometheus gauges labeled by session_id; cleaned up when a session ends.
var (
	sessionInputTokens  = metrics.NewLabeledGauge("session_input_tokens", "Total LLM input tokens for the session", "session_id", nil)
	sessionOutputTokens = metrics.NewLabeledGauge("session_output_tokens", "Total LLM output tokens for the session", "session_id", nil)
)

// SetSessionTokens records the cumulative token usage for a session.
func SetSessionTokens(sessionID string, inputTokens, outputTokens int) {
	sessionInputTokens.With(sessionID).Set(int64(inputTokens))
	sessionOutputTokens.With(sessionID).Set(int64(outputTokens))
}

// ClearSessionTokens removes the token metrics for a finished session.
func ClearSessionTokens(sessionID string) {
	sessionInputTokens.Delete(sessionID)
	sessionOutputTokens.Delete(sessionID)
}
