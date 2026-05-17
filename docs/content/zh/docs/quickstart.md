---
title: 快速开始
description: 5 分钟上手小海豚
slug: quickstart
weight: 5
---

5 分钟让小海豚跑起来。你需要一个 LLM API 密钥和一个终端。

## 1. 安装小海豚

**Homebrew 安装（macOS/Linux，推荐）：**

```bash
brew install dolphin-ai
dolphin-ai --version
```

**Go 安装**（需 Go 1.26+）：

```bash
go build -o dolphin-ai .
```

**Windows（PowerShell）：**

```powershell
$VERSION = "v0.2.1"
Invoke-WebRequest -Uri "https://github.com/dolphinZzv/dolphin/releases/download/$VERSION/dolphin-ai_${VERSION}_windows_x86_64.zip" -OutFile "dolphin-ai.zip"
Expand-Archive -Path "dolphin-ai.zip" -DestinationPath .
Move-Item .\dolphin-ai.exe "$env:LOCALAPPDATA\Microsoft\WindowsApps\dolphin-ai.exe"
```

详见[完整安装指南]({{< relref "docs/install" >}})。

## 2. 设置 API 密钥

按你的服务商设置全部环境变量：

{{< tabs >}}
{{< tab title="DeepSeek" id="deepseek" active="true" >}}
```bash
export DZ_LLM_TYPE="openai"
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_BASE_URL="https://api.deepseek.com/v1"
export DZ_LLM_MODEL="deepseek-v4-flash"
```
{{< /tab >}}
{{< tab title="通义千问" id="tongyi" >}}
```bash
export DZ_LLM_TYPE="openai"
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_BASE_URL="https://dashscope.aliyuncs.com/compatible-mode/v1"
export DZ_LLM_MODEL="qwen3.6-max-preview"
```
{{< /tab >}}
{{< tab title="智谱 GLM" id="zhipu" >}}
```bash
export DZ_LLM_TYPE="openai"
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_BASE_URL="https://open.bigmodel.cn/api/paas/v4"
export DZ_LLM_MODEL="glm-5.1"
```
{{< /tab >}}
{{< tab title="MiniMax" id="minimax" >}}
```bash
export DZ_LLM_TYPE="openai"
export DZ_LLM_API_KEY="sk-..."
export DZ_LLM_BASE_URL="https://api.minimaxi.com/v1"
export DZ_LLM_MODEL="MiniMax-M2.7"
```
{{< /tab >}}
{{< /tabs >}}

## 3. 启动小海豚

```bash
dolphin-ai
```

首次运行会启动设置向导：

1. **选择称呼** — 小海豚怎么称呼你
2. **生成配置** — 可选保存默认 `~/.dolphin/config.yaml`
3. **生成技能** — 可选保存入门技能文件

设置完成后会出现提示符：

```
Dolphin >
```

## 4. 试试看

```
Dolphin > 这个目录下有哪些文件？

Dolphin > 创建 hello.txt，内容写上"你好，小海豚！"

Dolphin > 查看系统环境信息
```

小海豚可以执行命令、读写文件、浏览网页等等。

## 5. 常用命令

```bash
dolphin-ai --version      # 查看版本
dolphin-ai init           # 初始化项目
dolphin-ai setup          # 交互式设置向导
dolphin-ai skills list    # 列出已安装技能
dolphin-ai sessions list  # 列出会话历史
```

## 6. 接下来

- **[配置参考]({{< relref "docs/config" >}})** — 自定义提供商、传输层和工具
- **[安装指南]({{< relref "docs/install" >}})** — 所有安装方式和故障排查

> Last modified: 2026-05-17
