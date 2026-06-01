package config

import "dolphin/internal/i18n"

func init() {
	i18n.Register("config",
		"en", i18n.Dict{
			"missing_provider": "config: missing llm.provider",
			"missing_fields":   "config: missing required fields: %s",
			"missing_model":    "config: missing llm.model",
			"read_failed":      "config: read %s: %w",
			"parse_failed":     "config: parse %s: %w",
		},
		"zh", i18n.Dict{
			"missing_provider": "配置: 缺少 llm.provider",
			"missing_fields":   "配置: 缺少必要字段: %s",
			"missing_model":    "配置: 缺少 llm.model",
			"read_failed":      "配置: 读取 %s 失败: %w",
			"parse_failed":     "配置: 解析 %s 失败: %w",
		},
	)
}
