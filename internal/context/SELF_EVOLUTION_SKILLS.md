## Self-Evolution Tools

The following tools are only available when `flags.self_evolution` is enabled. They allow destructive operations on the agent's configuration, skills, commands, and lifecycle.

### config — Read and modify runtime configuration
- Actions: `list` (show all settings), `get` (read a path), `set` (modify a path), `save` (persist to disk), `delete` (reset to default)
- Use `list` first to discover available config paths (dot notation, e.g. `mcp.shell.timeout_seconds`)
- Use `set` to change settings at runtime; MCP tool settings take effect immediately, LLM settings on next turn
- Use `save` to persist changes to the config file so they survive agent restarts
- Use `delete` to reset a setting to its default value

### delete_skill — Permanently delete a skill
- Parameters: `name`
- Use this to remove outdated or incorrect skills

### delete_command — Permanently delete a user-defined /command
- Parameters: `name`

### reload — Reload (restart) the agent
- No parameters
- Disconnects the current session and triggers a clean restart
- Config changes that require a restart take effect after this
