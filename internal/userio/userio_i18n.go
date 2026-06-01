package userio

import "dolphin/internal/i18n"

func init() {
	i18n.Register("userio",
		"en", i18n.Dict{
			"no_password_support": "transport %s does not support password input",
			"no_confirm_support":  "transport %s does not support interactive confirm",
			"confirm_prompt":      " (y/n): ",
			"no_select_support":   "transport %s does not support interactive select",
			"select_item":         "%d. %s",
		},
		"zh", i18n.Dict{
			"no_password_support": "传输 %s 不支持密码输入",
			"no_confirm_support":  "传输 %s 不支持交互式确认",
			"confirm_prompt":      " (y/n): ",
			"no_select_support":   "传输 %s 不支持交互式选择",
			"select_item":         "%d. %s",
		},
	)
}
