---
title: Commands
description: Built-in slash commands for Dolphin agent
slug: commands
weight: 12
---

Dolphin provides built-in slash commands available in any transport (terminal, SSH, email, chat, etc.). Use `/help` in-session to see the full list.

## Session Management

| Command | Description |
|---------|-------------|
| `/exit` or `exit` or `quit` | Exit the agent |
| `/new` | Start a fresh session (the previous session is summarized first) |
| `/status` | Show current session and agent status |
| `/cancel [id]` | Cancel all running tasks, or a specific task by ID |
| `/reload` | Reload (restart) the agent |

## Information

| Command | Description |
|---------|-------------|
| `/help` | Display help information |
| `/mcp` | List all registered MCP tools with descriptions |
| `/agents [name]` | List agents and their status; specify a name for details |
| `/skills [sub]` | List available skills. Subcommands: `new`, `delete`, `show` |
| `/commands [sub]` | List user-defined commands. Subcommands: `new`, `delete`, `show` |
| `/workflow [sub]` | List available workflows. Subcommands: `new`, `delete`, `show` |
| `/sessions [sub]` | List historical sessions. Subcommand: `dump <id>` |
| `/context [sub]` | Show context summary. Subcommands: `system`, `current`, `<section>` |
| `/transport` | Show enabled transports |

## Configuration

| Command | Description |
|---------|-------------|
| `/config [sub]` | View or modify configuration. Subcommands: `get`, `set` |
| `/model [name]` | List or switch the LLM model |
| `/provider [sub]` | List or switch LLM provider. Subcommand: `switch [name]` |

## Task Management

| Command | Description |
|---------|-------------|
| `/crontab` | View scheduled cron tasks |

## Other

| Command | Description |
|---------|-------------|
| `/forget <name>` | Reset conversation context for an agent |
| `/feedback` | Send feedback to the development team via email |

## Usage Notes

- Subcommands are used as `/command subcommand [args]`, e.g. `/skills new`
- Use `/help <command>` for detailed usage of a specific command
- All commands work in every transport — terminal, SSH, email, MQTT, DingTalk
- User-defined commands (from `.dolphin/commands/`) also use the `/` prefix

> Last modified: 2026-05-26
