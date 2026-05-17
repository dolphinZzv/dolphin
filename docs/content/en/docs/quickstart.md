---
title: Quick Start
description: Get Dolphin running in 5 minutes
slug: quickstart
weight: 5
---

Get Dolphin running in five minutes. You'll need an LLM API key and a terminal.

## 1. Install Dolphin

**Homebrew install (macOS/Linux, recommended):**

```bash
brew install dolphin-ai
dolphin-ai --version
```

**Build from source** (requires Go 1.26+):

```bash
go build -o dolphin-ai .
```

**Windows (PowerShell):**

```powershell
$VERSION = "v0.2.1"
Invoke-WebRequest -Uri "https://github.com/dolphinZzv/dolphin/releases/download/$VERSION/dolphin-ai_${VERSION}_windows_x86_64.zip" -OutFile "dolphin-ai.zip"
Expand-Archive -Path "dolphin-ai.zip" -DestinationPath .
Move-Item .\dolphin-ai.exe "$env:LOCALAPPDATA\Microsoft\WindowsApps\dolphin-ai.exe"
```

See the [full install guide]({{< relref "docs/install" >}}) for more options.

## 2. Set your API key

Choose your provider and set all required variables:

{{< tabs >}}
{{< tab title="Anthropic" id="anthropic" active="true" >}}
```bash
export DZ_LLM_TYPE="anthropic"
export DZ_LLM_API_KEY="sk-ant-..."
export DZ_LLM_BASE_URL="https://api.anthropic.com/v1"
export DZ_LLM_MODEL="claude-opus-4-7"
```
{{< /tab >}}
{{< tab title="OpenAI" id="openai" >}}
```bash
export DZ_LLM_TYPE="openai"
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_BASE_URL="https://api.openai.com/v1"
export DZ_LLM_MODEL="gpt-5.5"
```
{{< /tab >}}
{{< tab title="DeepSeek" id="deepseek" >}}
```bash
export DZ_LLM_TYPE="openai"
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_BASE_URL="https://api.deepseek.com/v1"
export DZ_LLM_MODEL="deepseek-v4-flash"
```
{{< /tab >}}
{{< /tabs >}}

## 3. Start Dolphin

```bash
dolphin-ai
```

On first run Dolphin starts a setup wizard:

1. **Choose your role** — how Dolphin addresses you
2. **Generate config** — optionally save a default `~/.dolphin/config.yaml`
3. **Generate skills** — optionally save starter skill files

Once setup completes, you'll see the prompt:

```
Dolphin >
```

## 4. Try it out

```
Dolphin > what files are in this directory?

Dolphin > create hello.txt with "Hello, Dolphin!" in it

Dolphin > show me the system environment
```

Dolphin can run commands, read and write files, browse the web, and more — all from the prompt.

## 5. Common Commands

```bash
dolphin-ai --version     # Check version
dolphin-ai init          # Initialize project
dolphin-ai setup         # Interactive setup wizard
dolphin-ai skills list   # List installed skills
dolphin-ai sessions list # List session history
```

## 6. Next Steps

Paste the output into https://mermaid.live to render.

## 6. What's next?

- **[Configuration Reference]({{< relref "docs/config" >}})** — customize providers, transports, and tools
- **[Install Guide]({{< relref "docs/install" >}})** — all install options and troubleshooting

> Last modified: 2026-05-17
