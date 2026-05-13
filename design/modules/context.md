# Context Building (`internal/context/`)

## Builder

`Builder.Build()` / `Builder.BuildForAgent(agentName)` 按顺序拼接系统提示：

1. **PREFACE.md** — `//go:embed` 硬编码，系统身份定义
2. **BUILTIN_SKILLS.md** — `//go:embed`，内置技能声明
3. **AGENTS.md** — agentDir → projectDir → userDir → systemDir fallback
4. **RULES.md** — 同上
5. **USER.md** — 同上
6. **SYSTEM.md** — user 目录，首次运行生成，每次会话注入

## Caching

文件内容按 mtime 缓存，仅在文件变更时重新读取。

## Agent-Specific Context

`BuildForAgent("reviewer")` 会在 `.dolphin/agents/reviewer/` 下优先查找 AGENTS.md/RULES.md/USER.md，fallback 到项目/用户/系统目录。
