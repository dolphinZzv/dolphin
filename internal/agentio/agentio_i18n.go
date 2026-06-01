package agentio

import "dolphin/internal/i18n"

func init() {
	i18n.Register("agentio",
		"en", i18n.Dict{
			"reply_prefix":      "\n%s> ",
			"session_switched":  "--- Session switched to: %s ---",
			"session_broadcast": "\n--- Session switched to: %s ---\n",
		},
		"zh", i18n.Dict{
			"reply_prefix":      "\n%s> ",
			"session_switched":  "--- 会话已切换到: %s ---",
			"session_broadcast": "\n--- 会话已切换到: %s ---\n",
		},
	)
}
