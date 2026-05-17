---
title: Install
description: Install Dolphin on Linux, macOS, or Windows
slug: install
weight: 10
---

Dolphin runs on **Linux**, **macOS**, and **Windows**. Choose the method that works best for you.

## Prerequisites

- **LLM API key** — from OpenAI (or any OpenAI-compatible provider), Anthropic, or a regional LLM service
- **Go 1.26+** (only required for building from source)

## Option 1: Homebrew install (recommended macOS/Linux)

```bash
brew install dolphin-ai
```

## Option 2: Download a pre-built binary

Download the archive for your platform from the [latest release](https://github.com/dolphinZzv/dolphin/releases/latest).

```bash
# macOS Apple Silicon (M1/M2/M3/M4)
curl -L https://github.com/dolphinZzv/dolphin/releases/latest/download/dolphin-ai_0.2.9_macOS_arm64.tar.gz | tar -xz
sudo mv dolphin-ai /usr/local/bin/

# macOS Intel
curl -L https://github.com/dolphinZzv/dolphin/releases/latest/download/dolphin-ai_0.2.9_macOS_x86_64.tar.gz | tar -xz
sudo mv dolphin-ai /usr/local/bin/

# Linux x86_64
curl -L https://github.com/dolphinZzv/dolphin/releases/latest/download/dolphin-ai_0.2.9_linux_x86_64.tar.gz | tar -xz
sudo mv dolphin-ai /usr/local/bin/

# Linux arm64
curl -L https://github.com/dolphinZzv/dolphin/releases/latest/download/dolphin-ai_0.2.9_linux_arm64.tar.gz | tar -xz
sudo mv dolphin-ai /usr/local/bin/
```

## Option 3: Build from source

Requires Go 1.26+ and `git`.

```bash
git clone https://github.com/dolphinZzv/dolphin.git
cd dolphin
go build -o dolphin-ai .
sudo mv dolphin-ai /usr/local/bin/
```

## Verify installation

```bash
dolphin-ai --version
```

### Recommended models

#### OpenAI
`gpt-4o` → `https://api.openai.com/v1`

#### Anthropic
`claude-sonnet-4-6` → `https://api.anthropic.com/v1`

#### DeepSeek
`deepseek-v4-flash` → `https://api.deepseek.com/v1`

#### MiniMax
`MiniMax-M2.7` → `https://api.minimax.chat/v1`

#### Zhipu GLM
`glm-5` → `https://open.bigmodel.cn/api/paas/v4`

#### Qwen
`qwen3.6-max-preview` → `https://dashscope.aliyuncs.com/compatible-mode/v1`

#### Kimi
`kimi-k2.6` → `https://api.moonshot.ai/v1`

Set `DZ_LLM_TYPE=openai` for OpenAI-compatible APIs, or `DZ_LLM_TYPE=anthropic` for Anthropic.

## Updating

Use the built-in update command:

```bash
dolphin update          # update to the latest release
dolphin update v1.0.0   # update to a specific version
dolphin update --list   # list available versions
```

Or re-install using one of the methods above.

## Troubleshooting

### "command not found: dolphin"

The binary isn't in your `PATH`. Either move it to a directory in your `PATH` (e.g. `/usr/local/bin`) or add the install directory:

```bash
export PATH=$PATH:/usr/local/bin
```

### "permission denied"

Make sure the binary is executable:

```bash
chmod +x /path/to/dolphin
```

### "Go not found"

Download a pre-built binary (Option 1) instead of building from source.

### Checksum verification

Each release includes a `checksums.txt` file. Verify your download:

```bash
sha256sum dolphin_*.tar.gz
# compare against checksums.txt from the release
```

> Last modified: 2026-05-17
