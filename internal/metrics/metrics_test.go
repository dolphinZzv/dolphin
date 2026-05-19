package metrics

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestCounter(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_counter", "test help", map[string]string{"env": "test"})
	if c.Value() != 0 {
		t.Errorf("expected 0, got %d", c.Value())
	}
	c.Inc()
	if c.Value() != 1 {
		t.Errorf("expected 1, got %d", c.Value())
	}
	c.Add(5)
	if c.Value() != 6 {
		t.Errorf("expected 6, got %d", c.Value())
	}
}

func TestCounterViaDefaultRegistry(t *testing.T) {
	_ = NewCounter("test_default_counter", "default help", nil)
}

func TestGauge(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("test_gauge", "test help", nil)
	g.Set(42)
	if g.Value() != 42 {
		t.Errorf("expected 42, got %d", g.Value())
	}
	g.Add(-10)
	if g.Value() != 32 {
		t.Errorf("expected 32, got %d", g.Value())
	}
}

func TestHistogram(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("test_histogram", "test help", map[string]string{"service": "test"}, nil)
	h.Observe(0.1)
	h.Observe(0.5)
	h.Observe(1.0)
}

func TestRenderCounter(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("http_requests_total", "Total HTTP requests", map[string]string{"method": "GET"}).Inc()

	output := r.Render()
	if !strings.Contains(output, "http_requests_total") {
		t.Errorf("expected metric name in output, got: %s", output)
	}
	if !strings.Contains(output, "counter") {
		t.Errorf("expected TYPE counter in output")
	}
	if !strings.Contains(output, `method="GET"`) {
		t.Errorf("expected labels in output")
	}
}

func TestRenderGauge(t *testing.T) {
	r := NewRegistry()
	r.NewGauge("pool_size", "Pool size", nil).Set(3)

	output := r.Render()
	if !strings.Contains(output, "pool_size") {
		t.Errorf("expected metric name in output")
	}
	if !strings.Contains(output, "gauge") {
		t.Errorf("expected TYPE gauge in output")
	}
}

func TestRenderHistogram(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("request_duration", "Request duration", nil, []float64{0.1, 0.5, 1.0})
	h.Observe(0.2)
	h.Observe(0.8)

	output := r.Render()
	if !strings.Contains(output, "request_duration_bucket") {
		t.Errorf("expected histogram buckets in output, got: %s", output)
	}
	if !strings.Contains(output, `le="+Inf"`) {
		t.Errorf("expected +Inf bucket in output")
	}
	if !strings.Contains(output, "request_duration_sum") {
		t.Errorf("expected histogram sum in output")
	}
	if !strings.Contains(output, "request_duration_count") {
		t.Errorf("expected histogram count in output")
	}
}

func TestRenderEmpty(t *testing.T) {
	r := NewRegistry()
	output := r.Render()
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestTimer(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("timer_test", "test", nil, nil)
	timer := StartTimer(h)
	timer.Stop()
	// Should not panic, value should be > 0
	if h.count.Load() != 1 {
		t.Errorf("expected 1 observation, got %d", h.count.Load())
	}
}

func TestRenderHTTP(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("test", "test", nil).Inc()
	body, ctype := r.RenderHTTP()
	if ctype != "text/plain; version=0.0.4; charset=utf-8" {
		t.Errorf("unexpected content type: %s", ctype)
	}
	if !strings.Contains(body, "test") {
		t.Errorf("expected test metric in body")
	}
}

func TestHistogramDefaultBuckets(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("default_buckets", "test", nil, nil)
	if len(h.bounds) != 11 {
		t.Errorf("expected 11 default buckets, got %d", len(h.bounds))
	}
}

func TestMultipleLabels(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("multi_label", "test", map[string]string{"app": "test", "env": "prod"})
	c.Inc()

	output := r.Render()
	if !strings.Contains(output, `app="test"`) {
		t.Errorf("expected app label in output")
	}
	if !strings.Contains(output, `env="prod"`) {
		t.Errorf("expected env label in output")
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("concurrent", "test", nil)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			c.Inc()
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 100; i++ {
			c.Inc()
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	if c.Value() != 200 {
		t.Errorf("expected 200, got %d", c.Value())
	}
}

// ---- Labeled metric tests ----

func TestLabeledCounter(t *testing.T) {
	r := NewRegistry()
	lc := r.NewLabeledCounter("requests_total", "Requests by status", "status", nil)

	c200 := lc.With("200")
	c500 := lc.With("500")
	c200.Inc()
	c200.Inc()
	c500.Inc()

	if c200.Value() != 2 {
		t.Errorf("expected 2, got %d", c200.Value())
	}
	if c500.Value() != 1 {
		t.Errorf("expected 1, got %d", c500.Value())
	}
}

func TestLabeledCounterWithBaseLabels(t *testing.T) {
	r := NewRegistry()
	lc := r.NewLabeledCounter("api_calls", "API calls by endpoint", "endpoint",
		map[string]string{"service": "dolphin"})

	c := lc.With("/health")
	c.Inc()

	output := r.Render()
	if !strings.Contains(output, `service="dolphin"`) {
		t.Errorf("expected base label in output, got: %s", output)
	}
	if !strings.Contains(output, `endpoint="/health"`) {
		t.Errorf("expected dynamic label in output")
	}
}

func TestLabeledCounterWithTwice(t *testing.T) {
	r := NewRegistry()
	lc := r.NewLabeledCounter("dup", "test", "env", nil)

	c1 := lc.With("prod")
	c2 := lc.With("prod") // same label value
	if c1 != c2 {
		t.Errorf("expected same counter for same label value")
	}
}

func TestRenderLabeledCounter(t *testing.T) {
	r := NewRegistry()
	lc := r.NewLabeledCounter("labeled_reqs", "Labeled requests", "method", nil)
	lc.With("GET").Inc()
	lc.With("POST").Inc()
	lc.With("POST").Inc()

	output := r.Render()

	if !strings.Contains(output, "# HELP labeled_reqs Labeled requests") {
		t.Errorf("expected HELP line, got: %s", output)
	}
	if !strings.Contains(output, "# TYPE labeled_reqs counter") {
		t.Errorf("expected TYPE line")
	}

	if !strings.Contains(output, `labeled_reqs{method="GET"} 1`) {
		t.Errorf("expected GET value line, got: %s", output)
	}
	if !strings.Contains(output, `labeled_reqs{method="POST"} 2`) {
		t.Errorf("expected POST value line, got: %s", output)
	}
}

func TestLabeledHistogram(t *testing.T) {
	r := NewRegistry()
	lh := r.NewLabeledHistogram("latency", "Latency by route", "route", nil,
		[]float64{0.1, 0.5, 1.0})

	h1 := lh.With("/api")
	h2 := lh.With("/static")
	h1.Observe(0.2)
	h1.Observe(0.8)
	h2.Observe(0.05)

	output := r.Render()
	if !strings.Contains(output, "# HELP latency Latency by route") {
		t.Errorf("expected HELP line, got: %s", output)
	}
	if !strings.Contains(output, "# TYPE latency histogram") {
		t.Errorf("expected TYPE line")
	}
	if !strings.Contains(output, `route="/api"`) {
		t.Errorf("expected /api label in output")
	}
	if !strings.Contains(output, `route="/static"`) {
		t.Errorf("expected /static label in output")
	}
	if !strings.Contains(output, "_bucket") {
		t.Errorf("expected histogram buckets in output")
	}
	if !strings.Contains(output, "_sum") {
		t.Errorf("expected histogram sum in output")
	}
	if !strings.Contains(output, "_count") {
		t.Errorf("expected histogram count in output")
	}
}

func TestLabeledHistogramWithBaseLabels(t *testing.T) {
	r := NewRegistry()
	lh := r.NewLabeledHistogram("db_query", "DB query duration", "query",
		map[string]string{"db": "primary"}, []float64{0.5, 1.0})
	lh.With("SELECT").Observe(0.3)

	output := r.Render()
	if !strings.Contains(output, `db="primary"`) {
		t.Errorf("expected base label in output, got: %s", output)
	}
	if !strings.Contains(output, `query="SELECT"`) {
		t.Errorf("expected dynamic label in output")
	}
}

func TestLabeledHistogramEmptyBounds(t *testing.T) {
	r := NewRegistry()
	lh := r.NewLabeledHistogram("def_buckets", "test", "tag", nil, nil)
	h := lh.With("a")
	h.Observe(0.5)

	if len(h.bounds) != 11 {
		t.Errorf("expected 11 default buckets, got %d", len(h.bounds))
	}
}

func TestConcurrentLabeledCounterWith(t *testing.T) {
	r := NewRegistry()
	lc := r.NewLabeledCounter("conc_labeled", "test", "worker", nil)
	done := make(chan struct{}, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				c := lc.With(fmt.Sprintf("worker-%d", id%3))
				c.Inc()
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Render should not panic under concurrent access
	_ = r.Render()

	total := lc.With("worker-0").Value() +
		lc.With("worker-1").Value() +
		lc.With("worker-2").Value()
	// 10 goroutines * 100 incs = 1000 total (but division among 3 workers is uneven)
	if total != 1000 {
		t.Errorf("expected total 1000, got %d", total)
	}
}

func TestConcurrentLabeledHistogramWith(t *testing.T) {
	r := NewRegistry()
	lh := r.NewLabeledHistogram("conc_lh", "test", "tag", nil, nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h := lh.With(fmt.Sprintf("tag-%d", id%3))
				h.Observe(float64(j) * 0.01)
			}
		}(i)
	}
	wg.Wait()

	// Render should not panic under concurrent access
	_ = r.Render()
}

func TestLabeledCounterDefaultRegistry(t *testing.T) {
	lc := NewLabeledCounter("def_labeled_counter", "default registry test", "env", nil)
	lc.With("test").Inc()
	output := Render()
	if !strings.Contains(output, "def_labeled_counter") {
		t.Errorf("expected metric in default registry render")
	}
}

func TestLabeledHistogramDefaultRegistry(t *testing.T) {
	lh := NewLabeledHistogram("def_labeled_hist", "default registry test", "tag", nil, nil)
	lh.With("val").Observe(0.5)
	output := Render()
	if !strings.Contains(output, "def_labeled_hist") {
		t.Errorf("expected metric in default registry render")
	}
}

func TestRenderLabeledMetricsEmpty(t *testing.T) {
	r := NewRegistry()
	r.NewLabeledCounter("empty_counter", "no values", "x", nil)
	r.NewLabeledHistogram("empty_histogram", "no values", "y", nil, nil)
	// No With() calls made — should produce no output
	output := r.Render()
	if output != "" {
		t.Errorf("expected empty output for unpopulated labeled metrics, got: %s", output)
	}
}

func TestRenderMixedLabeledAndUnlabeled(t *testing.T) {
	r := NewRegistry()
	// Unlabeled counter
	r.NewCounter("plain_requests", "Plain requests", nil).Inc()
	// Labeled counter with two values
	lc := r.NewLabeledCounter("labeled_requests", "Labeled requests", "status", nil)
	lc.With("200").Inc()
	lc.With("500").Inc()
	lc.With("500").Inc()
	// Unlabeled histogram
	h := r.NewHistogram("latency", "Latency", nil, []float64{0.5, 1.0})
	h.Observe(0.3)

	output := r.Render()

	if !strings.Contains(output, "plain_requests") {
		t.Errorf("expected unlabeled counter in output")
	}
	if !strings.Contains(output, "labeled_requests") {
		t.Errorf("expected labeled counter in output")
	}
	if !strings.Contains(output, `status="200"`) {
		t.Errorf("expected label value 200 in output")
	}
	if !strings.Contains(output, `status="500"`) {
		t.Errorf("expected label value 500 in output")
	}
	if !strings.Contains(output, "latency_bucket") {
		t.Errorf("expected histogram buckets in output")
	}

	lines := strings.Split(output, "\n")
	helpLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			helpLines++
		}
	}
	// 3 metric families = 3 HELP lines (plain_requests, labeled_requests, latency)
	if helpLines != 3 {
		t.Errorf("expected 3 HELP lines, got %d. Output:\n%s", helpLines, output)
	}
}

// ---- LabeledGauge tests ----

func TestLabeledGauge(t *testing.T) {
	r := NewRegistry()
	lg := r.NewLabeledGauge("pool_tokens", "Tokens in pool", "pool", nil)

	g1 := lg.With("pool-a")
	g2 := lg.With("pool-b")
	g1.Set(100)
	g2.Set(200)

	if g1.Value() != 100 {
		t.Errorf("expected 100, got %d", g1.Value())
	}
	if g2.Value() != 200 {
		t.Errorf("expected 200, got %d", g2.Value())
	}
}

func TestLabeledGaugeWithBaseLabels(t *testing.T) {
	r := NewRegistry()
	lg := r.NewLabeledGauge("session_tokens", "Session tokens", "session",
		map[string]string{"service": "dolphin"})

	lg.With("sess-1").Set(50)
	output := r.Render()
	if !strings.Contains(output, `service="dolphin"`) {
		t.Errorf("expected base label in output, got: %s", output)
	}
	if !strings.Contains(output, `session="sess-1"`) {
		t.Errorf("expected dynamic label in output")
	}
}

func TestLabeledGaugeWithTwice(t *testing.T) {
	r := NewRegistry()
	lg := r.NewLabeledGauge("dup", "test", "env", nil)

	g1 := lg.With("prod")
	g2 := lg.With("prod") // same label value
	if g1 != g2 {
		t.Errorf("expected same gauge for same label value")
	}
}

func TestRenderLabeledGauge(t *testing.T) {
	r := NewRegistry()
	lg := r.NewLabeledGauge("session_input_tokens", "Session input tokens", "session_id", nil)
	lg.With("abc123").Set(150)
	lg.With("def456").Set(300)

	output := r.Render()

	if !strings.Contains(output, "# HELP session_input_tokens Session input tokens") {
		t.Errorf("expected HELP line, got: %s", output)
	}
	if !strings.Contains(output, "# TYPE session_input_tokens gauge") {
		t.Errorf("expected TYPE gauge line")
	}
	if !strings.Contains(output, `session_input_tokens{session_id="abc123"} 150`) {
		t.Errorf("expected abc123 value line, got: %s", output)
	}
	if !strings.Contains(output, `session_input_tokens{session_id="def456"} 300`) {
		t.Errorf("expected def456 value line, got: %s", output)
	}
}

func TestLabeledGaugeDelete(t *testing.T) {
	r := NewRegistry()
	lg := r.NewLabeledGauge("temp", "Temporary metric", "id", nil)
	lg.With("sess-1").Set(100)
	lg.With("sess-2").Set(200)

	outputBefore := r.Render()
	if !strings.Contains(outputBefore, `id="sess-1"`) {
		t.Errorf("expected sess-1 before delete")
	}

	lg.Delete("sess-1")
	outputAfter := r.Render()
	if strings.Contains(outputAfter, `id="sess-1"`) {
		t.Errorf("expected sess-1 to be removed after delete, got: %s", outputAfter)
	}
	if !strings.Contains(outputAfter, `id="sess-2"`) {
		t.Errorf("expected sess-2 to remain after delete")
	}
}

func TestLabeledCounterDelete(t *testing.T) {
	r := NewRegistry()
	lc := r.NewLabeledCounter("reqs", "Requests by endpoint", "endpoint", nil)
	lc.With("/api").Inc()
	lc.With("/health").Inc()

	outputBefore := r.Render()
	if !strings.Contains(outputBefore, `endpoint="/api"`) {
		t.Errorf("expected /api before delete")
	}

	lc.Delete("/api")
	outputAfter := r.Render()
	if strings.Contains(outputAfter, `endpoint="/api"`) {
		t.Errorf("expected /api to be removed after delete")
	}
	if !strings.Contains(outputAfter, `endpoint="/health"`) {
		t.Errorf("expected /health to remain after delete")
	}
}

func TestLabeledGaugeDefaultRegistry(t *testing.T) {
	lg := NewLabeledGauge("def_labeled_gauge", "default registry test", "env", nil)
	lg.With("test").Set(42)
	output := Render()
	if !strings.Contains(output, "def_labeled_gauge") {
		t.Errorf("expected metric in default registry render")
	}
}

func TestRenderLabeledGaugeEmpty(t *testing.T) {
	r := NewRegistry()
	r.NewLabeledGauge("empty_gauge", "no values", "x", nil)
	// No With() calls — should produce no output
	output := r.Render()
	if output != "" {
		t.Errorf("expected empty output for unpopulated labeled gauge, got: %s", output)
	}
}

func TestConcurrentLabeledGaugeWith(t *testing.T) {
	r := NewRegistry()
	lg := r.NewLabeledGauge("conc_gauge", "test", "worker", nil)
	done := make(chan struct{}, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				g := lg.With(fmt.Sprintf("worker-%d", id%3))
				g.Add(1)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Render should not panic under concurrent access
	_ = r.Render()

	total := lg.With("worker-0").Value() +
		lg.With("worker-1").Value() +
		lg.With("worker-2").Value()
	if total != 1000 {
		t.Errorf("expected total 1000, got %d", total)
	}
}
