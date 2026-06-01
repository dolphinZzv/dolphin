package llm

import "dolphin/internal/i18n"

func init() {
	i18n.Register("llm",
		"en", i18n.Dict{
			"unknown_model":      "llm: unknown model %q",
			"provider_not_found": "llm: provider %q not found",
			"marshal_error":      "llm: marshal request: %w",
			"create_request":     "llm: create request: %w",
			"request_failed":     "llm: request failed: %w",
			"api_error":          "llm: %s (status %d)",
			"status_error":       "llm: status %d",
			"decode_error":       "llm: decode: %w",
		},
		"zh", i18n.Dict{
			"unknown_model":      "LLM: 未知模型 %q",
			"provider_not_found": "LLM: 未找到供应商 %q",
			"marshal_error":      "LLM: 序列化请求失败: %w",
			"create_request":     "LLM: 创建请求失败: %w",
			"request_failed":     "LLM: 请求失败: %w",
			"api_error":          "LLM: %s（状态 %d）",
			"status_error":       "LLM: 状态 %d",
			"decode_error":       "LLM: 解码失败: %w",
		},
	)
}
