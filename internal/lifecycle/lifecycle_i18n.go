package lifecycle

import "dolphin/internal/i18n"

func init() {
	i18n.Register("lifecycle",
		"en", i18n.Dict{
			"startup_notification": "Dolphin AI assistant online ✓",
		},
		"zh", i18n.Dict{
			"startup_notification": "Dolphin AI 助手已上线 ✓",
		},
	)
}
