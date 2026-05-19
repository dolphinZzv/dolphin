package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dolphin/internal/config"
	"dolphin/internal/mcp"

	"go.uber.org/zap"
)

// Tool provides web search capabilities.
type Tool struct {
	cfg    *config.MCPWebSearchConfig
	schema json.RawMessage
	client *http.Client
}

// searchInput is the JSON-unmarshal shape for the Execute input.
type searchInput struct {
	Query json.RawMessage `json:"query"`
}

func New(cfg *config.Config) *Tool {
	schema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"oneOf": []map[string]any{
					{"type": "string", "description": "Search query"},
					{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Multiple search queries — each is searched independently and results are merged",
					},
				},
				"description": "Search query string or array of query strings. When multiple queries are provided, each is searched independently.",
			},
		},
		"required": []string{"query"},
	})
	return &Tool{
		cfg:    &cfg.MCP.WebSearch,
		schema: schema,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (w *Tool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name:        "web_search",
		Description: "Search the web for current information. Accepts a single query string or an array of queries for multi-angle research.",
		InputSchema: w.schema,
		Priority:    w.cfg.Priority,
		Source:      "built-in",
	}
}

func (w *Tool) Execute(ctx context.Context, input json.RawMessage) (*mcp.ToolResult, error) {
	var params searchInput
	if err := json.Unmarshal(input, &params); err != nil {
		return &mcp.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	queries, err := parseQueries(params.Query)
	if err != nil {
		return &mcp.ToolResult{Content: fmt.Sprintf("invalid query: %v", err), IsError: true}, nil
	}
	if len(queries) == 0 {
		return &mcp.ToolResult{Content: "no query provided", IsError: true}, nil
	}

	zap.S().Debugw("web_search: executing", "provider", w.cfg.Provider, "queries", len(queries))

	var allResults []searchResult
	for _, q := range queries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		results, err := w.search(ctx, q)
		if err != nil {
			return &mcp.ToolResult{Content: fmt.Sprintf("search failed for %q: %v", q, err), IsError: true}, nil
		}
		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		return &mcp.ToolResult{Content: "No results found."}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d result(s):\n\n", len(allResults))
	for i, r := range allResults {
		fmt.Fprintf(&sb, "%d. [%s](%s)\n", i+1, r.Title, r.URL)
		if r.Snippet != "" {
			fmt.Fprintf(&sb, "   %s\n", r.Snippet)
		}
		sb.WriteString("\n")
	}
	return &mcp.ToolResult{Content: sb.String()}, nil
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

func (w *Tool) search(ctx context.Context, query string) ([]searchResult, error) {
	switch w.cfg.Provider {
	case "serper":
		return w.searchSerper(ctx, query)
	case "iflow":
		return w.searchIflow(ctx, query)
	default:
		return w.searchDuckDuckGo(ctx, query)
	}
}

// ---- DuckDuckGo (zero-config HTML scraping) ----

func (w *Tool) searchDuckDuckGo(ctx context.Context, query string) ([]searchResult, error) {
	u := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DolphinAgent/1.0)")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return parseDuckDuckGoHTML(string(body)), nil
}

func parseDuckDuckGoHTML(html string) []searchResult {
	var results []searchResult
	// Parse the simplified DDG HTML result page.
	// Each result is in a <div class="result"> with:
	//   <a class="result__a" href="...">title</a>
	//   <a class="result__snippet">snippet</a>
	// We use simple string matching since the HTML structure is stable.

	// Extract result blocks by finding result__a links
	linkStart := `<a rel="nofollow" class="result__a" href="`
	linkEnd := `</a>`
	snippetClass := `class="result__snippet">`
	snippetEnd := `</a>`

	remaining := html
	for {
		idx := strings.Index(remaining, linkStart)
		if idx < 0 {
			break
		}
		remaining = remaining[idx+len(linkStart):]

		// Extract URL
		quoteIdx := strings.IndexByte(remaining, '"')
		if quoteIdx < 0 {
			continue
		}
		resultURL := remaining[:quoteIdx]
		// Unescape HTML entities
		resultURL = strings.ReplaceAll(resultURL, "&amp;", "&")

		// Extract title (after </a> there's the title text before linkEnd)
		titleTag := `">`
		titleIdx := strings.Index(remaining, titleTag)
		if titleIdx < 0 {
			continue
		}
		titlePart := remaining[titleIdx+len(titleTag):]
		endIdx := strings.Index(titlePart, linkEnd)
		if endIdx < 0 {
			continue
		}
		title := titlePart[:endIdx]

		// Advance past this result
		remaining = titlePart[endIdx+len(linkEnd):]

		// Extract snippet
		snipIdx := strings.Index(remaining, snippetClass)
		snippet := ""
		if snipIdx >= 0 {
			snipPart := remaining[snipIdx+len(snippetClass):]
			snipEnd := strings.Index(snipPart, snippetEnd)
			if snipEnd >= 0 {
				snippet = snipPart[:snipEnd]
			}
		}

		results = append(results, searchResult{
			Title:   unescapeHTML(title),
			URL:     resultURL,
			Snippet: unescapeHTML(snippet),
		})
		if len(results) >= 10 {
			break
		}
	}
	return results
}

func unescapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}

// ---- Serper.dev API ----

func (w *Tool) searchSerper(ctx context.Context, query string) ([]searchResult, error) {
	if w.cfg.APIKey == "" {
		return nil, fmt.Errorf("serper provider requires api_key (set mcp.web_search.api_key)")
	}

	payload, _ := json.Marshal(map[string]string{"q": query})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://google.serper.dev/search", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", w.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("serper request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read serper response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("serper API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var serperResp struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(body, &serperResp); err != nil {
		return nil, fmt.Errorf("parse serper response: %w", err)
	}

	var results []searchResult
	for _, r := range serperResp.Organic {
		results = append(results, searchResult{
			Title:   r.Title,
			URL:     r.Link,
			Snippet: r.Snippet,
		})
	}
	return results, nil
}

// ---- iflow.cn API ----

func (w *Tool) searchIflow(ctx context.Context, query string) ([]searchResult, error) {
	if w.cfg.APIKey == "" {
		return nil, fmt.Errorf("iflow provider requires api_key (set mcp.web_search.api_key)")
	}

	payload, _ := json.Marshal(map[string]any{
		"keywords": query,
		"num":      5,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://platform.iflow.cn/api/search/webSearch", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+w.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iflow request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read iflow response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("iflow API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var bizResp struct {
		Success bool            `json:"success"`
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &bizResp); err != nil {
		return nil, fmt.Errorf("parse iflow response: %w", err)
	}
	if !bizResp.Success {
		return nil, fmt.Errorf("iflow API error: %s (code: %s)", bizResp.Message, bizResp.Code)
	}

	var data struct {
		Organic []struct {
			Title string `json:"title"`
			Link  string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(bizResp.Data, &data); err != nil {
		return nil, fmt.Errorf("parse iflow data: %w", err)
	}

	var results []searchResult
	for _, r := range data.Organic {
		results = append(results, searchResult{
			Title:   r.Title,
			URL:     r.Link,
			Snippet: r.Snippet,
		})
	}
	return results, nil
}

// ---- Helpers ----

// parseQueries extracts one or more query strings from a JSON value
// that can be either a string or an array of strings.
func parseQueries(raw json.RawMessage) ([]string, error) {
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil, nil
		}
		return []string{s}, nil
	}

	// Try array of strings
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("query must be a string or array of strings")
	}
	// Filter empty strings
	var out []string
	for _, q := range arr {
		if q != "" {
			out = append(out, q)
		}
	}
	return out, nil
}
