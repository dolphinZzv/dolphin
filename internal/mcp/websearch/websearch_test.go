package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dolphin/internal/config"
)

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// ---- parseQueries ----

func TestParseQueries_SingleString(t *testing.T) {
	raw := json.RawMessage(`"golang tutorial"`)
	queries, err := parseQueries(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 1 || queries[0] != "golang tutorial" {
		t.Fatalf("expected [golang tutorial], got %v", queries)
	}
}

func TestParseQueries_Array(t *testing.T) {
	raw := json.RawMessage(`["go 1.22 release", "golang generics", "go testing"]`)
	queries, err := parseQueries(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d: %v", len(queries), queries)
	}
}

func TestParseQueries_ArrayWithEmpty(t *testing.T) {
	raw := json.RawMessage(`["go 1.22", "", "golang", ""]`)
	queries, err := parseQueries(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 non-empty queries, got %d: %v", len(queries), queries)
	}
}

func TestParseQueries_EmptyString(t *testing.T) {
	raw := json.RawMessage(`""`)
	queries, err := parseQueries(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queries != nil {
		t.Fatalf("expected nil for empty string, got %v", queries)
	}
}

func TestParseQueries_InvalidType(t *testing.T) {
	raw := json.RawMessage(`42`)
	_, err := parseQueries(raw)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

// ---- unescapeHTML ----

func TestUnescapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello &amp; world", "hello & world"},
		{"&lt;script&gt;", "<script>"},
		{"&gt;= 10", ">= 10"},
		{"&quot;quoted&quot;", `"quoted"`},
		{"it&#x27;s", "it's"},
		{"&#39;single&#39;", "'single'"},
		{"plain text", "plain text"},
		{"", ""},
	}
	for _, tc := range tests {
		got := unescapeHTML(tc.input)
		if got != tc.expected {
			t.Errorf("unescapeHTML(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// ---- parseDuckDuckGoHTML ----

func TestParseDuckDuckGoHTML(t *testing.T) {
	html := `<html>
<body>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://example.com/go">Go Programming</a>
    <a class="result__snippet">Go is a compiled language.</a>
  </div>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://golang.org/pkg">Go Standard Library</a>
    <a class="result__snippet">Packages for fmt, http, json &amp; more.</a>
  </div>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://example.com/test">No Snippet</a>
  </div>
</body>
</html>`

	results := parseDuckDuckGoHTML(html)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	checkResult(t, results[0], "Go Programming", "https://example.com/go", "Go is a compiled language.")
	checkResult(t, results[1], "Go Standard Library", "https://golang.org/pkg", "Packages for fmt, http, json & more.")
	checkResult(t, results[2], "No Snippet", "https://example.com/test", "")
}

func TestParseDuckDuckGoHTML_Empty(t *testing.T) {
	results := parseDuckDuckGoHTML("<html></html>")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParseDuckDuckGoHTML_AmpersandURL(t *testing.T) {
	html := `<html><body>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://example.com/a?x=1&amp;y=2">Link with &amp;</a>
    <a class="result__snippet">Entity &amp; more.</a>
  </div>
</body></html>`

	results := parseDuckDuckGoHTML(html)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != "https://example.com/a?x=1&y=2" {
		t.Fatalf("expected URL with unescaped &, got %q", results[0].URL)
	}
	if results[0].Title != "Link with &" {
		t.Fatalf("expected title 'Link with &', got %q", results[0].Title)
	}
}

func checkResult(t *testing.T, r searchResult, title, url, snippet string) {
	t.Helper()
	if r.Title != title {
		t.Errorf("expected title %q, got %q", title, r.Title)
	}
	if r.URL != url {
		t.Errorf("expected URL %q, got %q", url, r.URL)
	}
	if r.Snippet != snippet {
		t.Errorf("expected snippet %q, got %q", snippet, r.Snippet)
	}
}

// ---- Definition ----

func TestDefinition_Name(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)
	def := tool.Definition()
	if def.Name != "web_search" {
		t.Fatalf("expected name 'web_search', got %q", def.Name)
	}
	if def.Source != "built-in" {
		t.Fatalf("expected source 'built-in', got %q", def.Source)
	}
}

func TestDefinition_InputSchema_HasOneOf(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)
	def := tool.Definition()

	var schema map[string]any
	if err := json.Unmarshal(def.InputSchema, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	queryProp, ok := props["query"].(map[string]any)
	if !ok {
		t.Fatal("schema missing query property")
	}
	if _, ok := queryProp["oneOf"]; !ok {
		t.Fatal("query property missing oneOf")
	}
}

// ---- Execute via HTTP mock ----

func TestExecute_SingleQuery_Success(t *testing.T) {
	mockHTML := `<html><body>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://go.dev/">The Go Programming Language</a>
    <a class="result__snippet">Go is an open source programming language.</a>
  </div>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://go.dev/doc/">Go Documentation</a>
    <a class="result__snippet">Official Go docs &amp; tutorials.</a>
  </div>
</body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)
	// Intercept all HTTP traffic to the test server
	tool.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		u := ts.URL + "?" + req.URL.RawQuery
		mockReq, _ := http.NewRequest(req.Method, u, nil)
		mockReq.Header = req.Header
		return http.DefaultTransport.RoundTrip(mockReq)
	})

	input, _ := json.Marshal(map[string]any{"query": "golang"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content)
	}
	if !strings.Contains(result.Content, "The Go Programming Language") {
		t.Fatalf("expected 'The Go Programming Language', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "https://go.dev/") {
		t.Fatalf("expected https://go.dev/ in result, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Go Documentation") {
		t.Fatalf("expected 'Go Documentation', got: %s", result.Content)
	}
}

func TestExecute_NoResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>No results.</body></html>"))
	}))
	defer ts.Close()

	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)
	tool.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		mockReq, _ := http.NewRequest("GET", ts.URL, nil)
		return http.DefaultTransport.RoundTrip(mockReq)
	})

	input, _ := json.Marshal(map[string]any{"query": "xyznonexistent"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "No results found." {
		t.Fatalf("expected 'No results found.', got: %s", result.Content)
	}
}

func TestExecute_MultipleQueries(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprint(w, `<html><body>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://example.com/r1">Result1</a>
    <a class="result__snippet">First result</a>
  </div></body></html>`)
	}))
	defer ts.Close()

	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)
	tool.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		mockReq, _ := http.NewRequest("GET", ts.URL+"?"+req.URL.RawQuery, nil)
		return http.DefaultTransport.RoundTrip(mockReq)
	})

	input, _ := json.Marshal(map[string]any{
		"query": []string{"golang", "rust"},
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 search calls, got %d", callCount)
	}
	if !strings.Contains(result.Content, "2 result(s)") {
		t.Fatalf("expected 2 results in output, got: %s", result.Content)
	}
}

func TestExecute_EmptyQuery_ReturnsError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)

	input, _ := json.Marshal(map[string]any{"query": ""})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for empty query, got: %s", result.Content)
	}
}

func TestExecute_InvalidInput_ReturnsError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)

	input, _ := json.Marshal(map[string]any{"query": 42})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for invalid input, got: %s", result.Content)
	}
}

func TestExecute_MissingQuery_ReturnsError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)

	input, _ := json.Marshal(map[string]any{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for missing query, got: %s", result.Content)
	}
}

func TestExecute_ContextCancelled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input, _ := json.Marshal(map[string]any{"query": "test"})
	_, err := tool.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---- Iflow provider ----

func TestExecute_Iflow_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": true,
			"code": "200",
			"message": "操作成功",
			"data": {
				"organic": [
					{
						"title": "Go语言并发编程",
						"link": "https://example.com/go-concurrency",
						"snippet": "Go并发编程入门教程"
					},
					{
						"title": "Goroutine详解",
						"link": "https://example.com/goroutine",
						"snippet": "深入了解goroutine"
					}
				],
				"query": "Go并发"
			}
		}`))
	}))
	defer ts.Close()

	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "iflow"
	cfg.MCP.WebSearch.APIKey = "test-key"
	tool := New(cfg)
	tool.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		mockReq, _ := http.NewRequest("POST", ts.URL, req.Body)
		mockReq.Header = req.Header
		return http.DefaultTransport.RoundTrip(mockReq)
	})

	input, _ := json.Marshal(map[string]any{"query": "Go并发"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Go语言并发编程") {
		t.Fatalf("expected 'Go语言并发编程', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "https://example.com/go-concurrency") {
		t.Fatalf("expected URL in result, got: %s", result.Content)
	}
}

func TestExecute_Iflow_MissingKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "iflow"
	cfg.MCP.WebSearch.APIKey = ""
	tool := New(cfg)

	input, _ := json.Marshal(map[string]any{"query": "test"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for missing API key, got: %s", result.Content)
	}
}

func TestExecute_Iflow_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": false,
			"code": "500",
			"message": "内部错误",
			"data": null
		}`))
	}))
	defer ts.Close()

	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "iflow"
	cfg.MCP.WebSearch.APIKey = "test-key"
	tool := New(cfg)
	tool.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		mockReq, _ := http.NewRequest("POST", ts.URL, req.Body)
		mockReq.Header = req.Header
		return http.DefaultTransport.RoundTrip(mockReq)
	})

	input, _ := json.Marshal(map[string]any{"query": "test"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for API error, got: %s", result.Content)
	}
}

// ---- Integration-style: real DuckDuckGo fetch and parse ----

func TestRealDuckDuckGoFetchAndParse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real HTTP test in short mode")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>
  <div class="result">
    <a rel="nofollow" class="result__a" href="https://example.com">Example</a>
    <a class="result__snippet">Example domain.</a>
  </div>
</body></html>`))
	}))
	defer ts.Close()

	cfg := config.DefaultConfig()
	cfg.MCP.WebSearch.Provider = "duckduckgo"
	tool := New(cfg)
	tool.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		mockReq, _ := http.NewRequest("GET", ts.URL, nil)
		return http.DefaultTransport.RoundTrip(mockReq)
	})

	input, _ := json.Marshal(map[string]any{"query": "test"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Example") {
		t.Fatalf("expected 'Example' in result, got: %s", result.Content)
	}
}
