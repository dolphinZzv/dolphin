package brain

import "dolphin/internal/i18n"

func init() {
	i18n.Register("brain",
		"en", i18n.Dict{
			"mkdir_failed":     "brain: mkdir: %w",
			"open_repo_failed": "brain: open repo: %w",
			"init_failed":      "brain: git init: %w",
			"path_traversal":   "brain: path traversal denied: %s",
			"read_failed":      "brain: read %s: %w",
			"mkdir_fail":       "brain: mkdir %s: %w",
			"write_failed":     "brain: write %s: %w",
			"list_failed":      "brain: list: %w",
			"not_initialized":  "brain: not initialized",
			"status_format":    "%s %s (%s)\n",
		},
		"zh", i18n.Dict{
			"mkdir_failed":     "大脑: 创建目录失败: %w",
			"open_repo_failed": "大脑: 打开仓库失败: %w",
			"init_failed":      "大脑: git 初始化失败: %w",
			"path_traversal":   "大脑: 路径遍历被拒绝: %s",
			"read_failed":      "大脑: 读取 %s 失败: %w",
			"mkdir_fail":       "大脑: 创建目录 %s 失败: %w",
			"write_failed":     "大脑: 写入 %s 失败: %w",
			"list_failed":      "大脑: 列出失败: %w",
			"not_initialized":  "大脑: 未初始化",
			"status_format":    "%s %s (%s)\n",
		},
	)
}
