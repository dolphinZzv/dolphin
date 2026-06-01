package tool

import "dolphin/internal/i18n"

func init() {
	i18n.Register("tool",
		"en", i18n.Dict{
			// general
			"not_found":      "tool %q not found",
			"invalid_args":   "invalid args: %s",
			"connect_failed": "failed to connect: %s",
			"loaded_from":    "loaded %d tools from %s",
			"timed_out":      "tool execution timed out",

			// shell
			"shell_desc":         "Execute a shell command and get the output",
			"shell_cmd_required": "command is required",

			// brain tools
			"brain_read_desc":     "Read a file from the brain (long-term memory) directory",
			"brain_write_desc":    "Write content to a file in the brain directory",
			"brain_list_desc":     "List all .md files in the brain directory",
			"brain_log_desc":      "View recent git commit history of the brain",
			"brain_path_required": "path is required",
			"brain_read_failed":   "brain read failed: %s",
			"brain_write_failed":  "brain write failed: %s",
			"brain_written":       "written to brain: %s",
			"brain_list_failed":   "brain list failed: %s",
			"brain_empty":         "(empty)",
			"brain_log_failed":    "brain log failed: %s",
			"brain_no_commits":    "(no commits)",

			// skill tools
			"skill_create_desc":   "Create a new skill. Args: {name, description?, prompt?, tools?}",
			"skill_search_desc":   "Search for skills by query",
			"skill_load_desc":     "Load/enable a skill by name",
			"skill_update_desc":   "Update an existing skill",
			"skill_delete_desc":   "Delete a skill by name",
			"skill_invalid_def":   "invalid skill definition: %s",
			"skill_name_required": "skill name is required",
			"skill_save_failed":   "failed to save skill: %s",
			"skill_created":       "skill '%s' created",
			"skill_not_found":     "skill not found: %s",
			"skill_loaded":        "skill '%s' loaded",
			"skill_updated":       "skill '%s' updated",
			"skill_update_failed": "failed to update: %s",
			"skill_deleted":       "skill '%s' deleted",
			"skill_delete_failed": "failed to delete: %s",

			// session tools
			"session_list_desc":   "List all sessions with their IDs and timestamps",
			"session_switch_desc": "Switch to a different session by ID. Args: {id}",
			"session_list_failed": "failed to list sessions: %s",
			"session_none":        "no sessions found",
			"session_switch_fail": "failed to switch: %s",
			"session_switched":    "switched to session %s",
			"session_id_required": "session ID is required",

			// scheduler tools
			"scheduler_create_desc": "Create a scheduled task. Args: {name, schedule, command}",
			"scheduler_list_desc":   "List all scheduled tasks with their status",
			"scheduler_delete_desc": "Delete a scheduled task by ID. Args: {id}",
			"scheduler_delay_desc":  "Schedule a one-shot delayed task. Args: {name, delay, command}",
			"scheduler_create_fail": "failed to create task: %s",
			"scheduler_created":     "task %q created (id: %s)",
			"scheduler_none":        "no scheduled tasks",
			"scheduler_delete_fail": "failed to delete: %s",
			"scheduler_deleted":     "task deleted",
		},
		"zh", i18n.Dict{
			// general
			"not_found":      "工具 %q 未找到",
			"invalid_args":   "无效参数: %s",
			"connect_failed": "连接失败: %s",
			"loaded_from":    "已从 %2$s 加载 %1$d 个工具",
			"timed_out":      "工具执行超时",

			// shell
			"shell_desc":         "执行一条 shell 命令并获取输出",
			"shell_cmd_required": "必须提供命令",

			// brain tools
			"brain_read_desc":     "从大脑（长期记忆）目录读取文件",
			"brain_write_desc":    "向大脑目录中的文件写入内容",
			"brain_list_desc":     "列出大脑目录中的所有 .md 文件",
			"brain_log_desc":      "查看大脑最近的 git 提交历史",
			"brain_path_required": "必须提供路径",
			"brain_read_failed":   "大脑读取失败: %s",
			"brain_write_failed":  "大脑写入失败: %s",
			"brain_written":       "已写入大脑: %s",
			"brain_list_failed":   "大脑列表失败: %s",
			"brain_empty":         "（空）",
			"brain_log_failed":    "大脑日志失败: %s",
			"brain_no_commits":    "（无提交）",

			// skill tools
			"skill_create_desc":   "创建新技能。参数: {name, description?, prompt?, tools?}",
			"skill_search_desc":   "搜索技能",
			"skill_load_desc":     "加载/启用技能",
			"skill_update_desc":   "更新已有技能",
			"skill_delete_desc":   "按名称删除技能",
			"skill_invalid_def":   "无效的技能定义: %s",
			"skill_name_required": "必须提供技能名称",
			"skill_save_failed":   "保存技能失败: %s",
			"skill_created":       "技能 '%s' 已创建",
			"skill_not_found":     "技能未找到: %s",
			"skill_loaded":        "技能 '%s' 已加载",
			"skill_updated":       "技能 '%s' 已更新",
			"skill_update_failed": "更新失败: %s",
			"skill_deleted":       "技能 '%s' 已删除",
			"skill_delete_failed": "删除失败: %s",

			// session tools
			"session_list_desc":   "列出所有会话的 ID 和时间戳",
			"session_switch_desc": "切换到指定 ID 的会话。参数: {id}",
			"session_list_failed": "列出会话失败: %s",
			"session_none":        "没有找到会话",
			"session_switch_fail": "切换失败: %s",
			"session_switched":    "已切换到会话 %s",
			"session_id_required": "必须提供会话 ID",

			// scheduler tools
			"scheduler_create_desc": "创建定时任务。参数: {name, schedule, command}",
			"scheduler_list_desc":   "列出所有定时任务及其状态",
			"scheduler_delete_desc": "按 ID 删除定时任务。参数: {id}",
			"scheduler_delay_desc":  "创建一次性延迟任务。参数: {name, delay, command}",
			"scheduler_create_fail": "创建任务失败: %s",
			"scheduler_created":     "任务 %q 已创建（ID: %s）",
			"scheduler_none":        "没有定时任务",
			"scheduler_delete_fail": "删除失败: %s",
			"scheduler_deleted":     "任务已删除",
		},
	)
}
