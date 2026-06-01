package permission

import "dolphin/internal/i18n"

func init() {
	i18n.Register("permission",
		"en", i18n.Dict{
			"result_allow":    "allow",
			"result_deny":     "deny",
			"result_no_match": "no_match",
		},
		"zh", i18n.Dict{
			"result_allow":    "允许",
			"result_deny":     "拒绝",
			"result_no_match": "无匹配",
		},
	)
}
