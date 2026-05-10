package metrics

import (
	"net/http"

	"go.uber.org/zap"
)

// Handler returns an http.Handler that serves Prometheus-formatted metrics.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, contentType := defaultRegistry.RenderHTTP()
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(body))
		if err != nil {
			zap.S().Warnw("metrics: write response failed", "error", err)
		}
	})
}
