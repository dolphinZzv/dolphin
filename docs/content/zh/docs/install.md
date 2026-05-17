---
title: 安装
description: 在 Linux、macOS 或 Windows 上安装小海豚
slug: install
weight: 10
---

小海豚支持 **Linux**、**macOS** 和 **Windows** 系统。选择最适合你的安装方式。

## 前置要求

- **LLM API 密钥** — 来自 DeepSeek、MiniMax、Kimi、智谱 GLM、通义千问，或 Anthropic
- **Go 1.26+**（仅源码编译时需要）

## 方式一：Homebrew 安装（推荐 macOS/Linux）

```bash
brew install dolphin-ai
```

## 方式二：下载预编译二进制

从 [latest release](https://github.com/dolphinZzv/dolphin/releases/latest) 下载对应平台的压缩包。

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

## 方式三：源码编译

需要 Go 1.26+ 和 `git`。

```bash
git clone https://github.com/dolphinZzv/dolphin.git
cd dolphin
go build -o dolphin-ai .
sudo mv dolphin-ai /usr/local/bin/
```

### macOS

`make` 包含在 Xcode Command Line Tools 中。如未安装：

```bash
xcode-select --install
```

然后：

```bash
make build   # 生成 ./dolphin-ai（版本号 = dev）
```

或手动编译：

```bash
go build -ldflags="-X 'dolphin/cmd.Version=$(VERSION)'" -o dolphin-ai .
```

### Windows

**方式 A — Go build（PowerShell / cmd）：**

```powershell
# 开发版本（版本号 = dev）
go build -o dolphin-ai.exe .

# 发布版本
$env:VERSION = "v1.0.0"
go build -ldflags="-X 'dolphin/cmd.Version=$env:VERSION'" -o dolphin-ai.exe .
```

**方式 B — Make（Windows 原生）：**

通过以下方式安装 `make`：

```powershell
# Chocolatey
choco install make

# winget
winget install GnuWin32.Make
```

然后编译：

```powershell
make build   # 生成 ./dolphin-ai.exe（版本号 = dev）
make build VERSION=v1.0.0
```

**方式 C — Git Bash / WSL：**

```bash
make build   # 生成 ./dolphin-ai.exe（版本号 = dev）
```

## 验证安装

```bash
dolphin-ai --version
```

应看到如下输出：

```
dolphin-ai dev
```

## 配置 API 密钥

小海豚运行至少需要一个 API 密钥。通过环境变量设置：

```bash
# DeepSeek
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_MODEL="deepseek-v4-flash"
export DZ_LLM_BASE_URL="https://api.deepseek.com/v1"
export DZ_LLM_TYPE="openai"

./dolphin-ai
```

首次运行会进入设置向导 — 选择角色、选填生成配置文件和系统信息文件。所有数据均存储在本地。

### 中国地区推荐模型

#### DeepSeek
`deepseek-v4-flash` → `https://api.deepseek.com/v1`

#### MiniMax
`MiniMax-M2.7` → `https://api.minimax.chat/v1`

#### 智谱 GLM
`glm-5` → `https://open.bigmodel.cn/api/paas/v4`

#### 通义千问
`qwen3.6-max-preview` → `https://dashscope.aliyuncs.com/compatible-mode/v1`

#### Kimi
`kimi-k2.6` → `https://api.moonshot.ai/v1`

以上均设置 `DZ_LLM_TYPE=openai` 即可使用。

## 升级

使用内置的更新命令：

```bash
dolphin update          # 升级到最新版本
dolphin update v1.0.0   # 升级到指定版本
dolphin update --list   # 列出可用版本
```

或重新通过上述方式安装。

## 常见问题

### "command not found: dolphin"

二进制文件不在 `PATH` 中。将其移动到 `PATH` 包含的目录（如 `/usr/local/bin`），或将安装目录加入 `PATH`：

```bash
export PATH=$PATH:/usr/local/bin
```

### "permission denied"

确保二进制文件有执行权限：

```bash
chmod +x /path/to/dolphin
```

### 没有 Go 环境

使用方式一（下载预编译二进制）代替源码编译。

### 校验文件完整性

每个 release 附带 `checksums.txt` 文件。验证下载的压缩包：

```bash
sha256sum dolphin_*.tar.gz
# 与 release 中的 checksums.txt 对比
```

> Last modified: 2026-05-17
