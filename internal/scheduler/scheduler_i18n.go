package scheduler

import "dolphin/internal/i18n"

func init() {
	i18n.Register("scheduler",
		"en", i18n.Dict{
			"task_enabled":   "enabled",
			"task_disabled":  "disabled",
			"task_pending":   "pending",
			"task_not_found": "task not found",
		},
		"zh", i18n.Dict{
			"task_enabled":   "已启用",
			"task_disabled":  "已禁用",
			"task_pending":   "等待中",
			"task_not_found": "任务未找到",
		},
	)
}
