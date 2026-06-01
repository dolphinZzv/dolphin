package mcp

import "dolphin/internal/i18n"

func init() {
	i18n.Register("mcp",
		"en", i18n.Dict{
			"error_code":      "mcp: %s (code %d)",
			"unmarshal_tools": "mcp: unmarshal tools: %w",
			"error_msg":       "mcp error: %s",
			"marshal_error":   "mcp: marshal: %w",
			"request_error":   "mcp: request: %w",
			"http_error":      "mcp: http: %w",
			"read_error":      "mcp: read: %w",
			"http_status":     "mcp: http status %d: %s",
			"unmarshal_error": "mcp: unmarshal: %w",
			"no_sse_result":   "mcp: no result in SSE stream",
		},
		"zh", i18n.Dict{
			"error_code":      "MCP: %s（错误码 %d）",
			"unmarshal_tools": "MCP: 解析工具列表失败: %w",
			"error_msg":       "MCP 错误: %s",
			"marshal_error":   "MCP: 序列化失败: %w",
			"request_error":   "MCP: 请求失败: %w",
			"http_error":      "MCP: HTTP 错误: %w",
			"read_error":      "MCP: 读取失败: %w",
			"http_status":     "MCP: HTTP 状态 %d: %s",
			"unmarshal_error": "MCP: 解析失败: %w",
			"no_sse_result":   "MCP: SSE 流中没有结果",
		},
	)
}
