package context

import "dolphin/internal/i18n"

func init() {
	i18n.Register("context",
		"en", i18n.Dict{
			"section_workspace": "## Workspace\nYour workspace directory is %s.",
			"section_brain":     "## Brain Index\nThe following is an index of my long-term memory (brain) stored as a git repository.",
			"section_design":    "## Design Notes\n",
			"section_soul":      "## Soul\n",
			"section_transport": "## Transport Context\n",
			"section_skill":     "## Skill: ",
			"default_prompt":    "You are Dolphin, an AI assistant.",
		},
		"zh", i18n.Dict{
			"section_workspace": "## 工作区\n你的工作目录是 %s。",
			"section_brain":     "## 大脑索引\n以下是我长期记忆（以 git 仓库形式存储的大脑）的索引。",
			"section_design":    "## 设计说明\n",
			"section_soul":      "## 灵魂\n",
			"section_transport": "## 传输上下文\n",
			"section_skill":     "## 技能: ",
			"default_prompt":    "你是 Dolphin，一个 AI 助手。",
		},
	)
}
