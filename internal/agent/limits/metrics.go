package limits

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	llmRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_limits_requests_total",
			Help: "Total LLM requests by level",
		},
		[]string{"level"},
	)

	llmTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_limits_tokens_total",
			Help: "Total LLM tokens by direction and level",
		},
		[]string{"direction", "level"},
	)

	llmLimitsCurrent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llm_limits_current",
			Help: "Current usage by type and level",
		},
		[]string{"type", "level"},
	)

	llmLimitsBlockedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_limits_blocked_total",
			Help: "Total blocked requests by limit type and enforcement",
		},
		[]string{"limit_type", "enforcement"},
	)

	llmConcurrencyCurrent = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "llm_limits_concurrency_current",
			Help: "Current concurrent LLM calls",
		},
	)
)

func RecordRequest(level string) {
	llmRequestsTotal.WithLabelValues(level).Inc()
	llmLimitsCurrent.WithLabelValues("requests", level).Inc()
}

func RecordTokens(direction, level string, count int) {
	llmTokensTotal.WithLabelValues(direction, level).Add(float64(count))
	llmLimitsCurrent.WithLabelValues("tokens_"+direction, level).Add(float64(count))
}

func RecordBlocked(limitType, enforcement string) {
	llmLimitsBlockedTotal.WithLabelValues(limitType, enforcement).Inc()
}

func RecordConcurrency(count int) {
	llmConcurrencyCurrent.Set(float64(count))
}
