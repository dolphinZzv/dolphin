package skill

import "dolphin/internal/i18n"

func init() {
	i18n.Register("skill",
		"en", i18n.Dict{
			"seed_save_failed":    "seed: failed to save skill %q: %v\n",
			"seed_created":        "seed: created skill %q\n",
			"seed_already_exists": "seed: skill %q already exists (enabled=%v)\n",
			"index_title":         "# Skills\n\n",
			"index_header":        "| Name | Description | Status |\n|---|---|---|\n",
			"index_empty":         "No skills registered.\n",
			"enabled":             "enabled",
			"disabled":            "disabled",
		},
		"zh", i18n.Dict{
			"seed_save_failed":    "种子: 保存技能 %q 失败: %v\n",
			"seed_created":        "种子: 已创建技能 %q\n",
			"seed_already_exists": "种子: 技能 %q 已存在（启用=%v）\n",
			"index_title":         "# 技能\n\n",
			"index_header":        "| 名称 | 描述 | 状态 |\n|---|---|---|\n",
			"index_empty":         "没有已注册的技能。\n",
			"enabled":             "已启用",
			"disabled":            "已禁用",
		},
	)
}
