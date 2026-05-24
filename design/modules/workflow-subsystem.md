# Workflow Subsystem

## Overview

Workflow 子系统提供了一套机制，将 LLM 行为约束到固定的、逐步的 Markdown 文件中。每个 workflow 是一个带 YAML frontmatter 的 Markdown 文件，LLM 在匹配到对应任务时必须使用 `run_workflow` 工具严格遵循每个步骤，不可即兴发挥或跳过步骤。

Workflow 是强制性约束，而非建议性指南。

## File Format

```
.dolphin/workflows/<name>/WORKFLOW.md
```

```markdown
---
name: deploy-check
description: Check deployment health
---

When I ask you to run the deployment check, follow these steps:
1. Run `kubectl get pods --all-namespaces`
2. Run `kubectl get nodes`
3. Summarize findings
```

- YAML frontmatter 由 `---` 分隔，支持 `name` 和 `description` 字段
- 主体为 Markdown，作为 workflow 的内容
- 目录名即为 workflow 名称，frontmatter 中的 `name` 可覆盖

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  LLM / Agent                      │
│  (list_workflows / load_workflow / run_workflow)  │
└────────────────────┬────────────────────────────┘
                     │ MCP Tool Calls
                     ▼
┌─────────────────────────────────────────────────┐
│              Workflow Manager                     │
│  ┌─────────┐  ┌──────────┐  ┌────────────────┐  │
│  │  Load()  │  │  tools   │  │  ContextMD()   │  │
│  │  Reload()│  │  (8个)   │  │  (注入系统提示) │  │
│  │  Watch   │  └──────────┘  └────────────────┘  │
│  └─────────┘                                      │
│  ┌──────────────────────────────────────────┐     │
│  │  Agent可见性过滤                           │     │
│  │  ListForAgent / GetForAgent               │     │
│  └──────────────────────────────────────────┘     │
└────────────────────┬────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
  .dolphin/workflows/     ~/.dolphin/workflows/
  (项目级，可写)            (用户级，只读)
```

### Key Components

| Component | File | Responsibility |
|-----------|------|----------------|
| `Manager` | `internal/subsystem/workflow/workflow.go` | 加载、管理、提供 workflows |
| `Workflow` | `internal/subsystem/workflow/workflow.go` | 单个 workflow 的数据结构 |
| `subsystem.Provider` | `internal/subsystem/registry.go` | 子系统注册接口 |
| CLI | `cmd/workflow.go` | `dolphin workflow` 命令 |

## Manager API

### Lifecycle

| Method | Description |
|--------|-------------|
| `NewManager(dirs ...string)` | 创建一个 Manager，支持多目录（首个为可写目录） |
| `Load()` | 扫描所有目录，加载 WORKFLOW.md 文件 |
| `Register(wf *Workflow)` | 直接注册一个 Workflow 对象 |
| `Unregister(name string)` | 注销一个 Workflow |
| `Disable(name string)` | 重命名目录为 `.disabled/<name>` |
| `Enable(name string)` | 恢复禁用的 Workflow |
| `Reload()` | 重新加载所有 Workflow |
| `WatchAndReload(ctx, interval)` | 定时轮询文件系统变更 |

### Agent Visibility

| Method | Description |
|--------|-------------|
| `ListForAgent(allowed []string)` | 返回 agent 可见的 workflow 列表。`allowed` 为空/ nil 返回全部 |
| `GetForAgent(name, allowed)` | 返回指定 workflow，若 agent 不可见则返回 nil |

## LLM Tool Definitions

通过 `subsystem.Provider.ToolDefs()` 注册 8 个 MCP 工具：

### Always Available（3个）

| Tool | Handler | Description |
|------|---------|-------------|
| `list_workflows` | `m.handleListWorkflows` | 列出所有可用 workflows |
| `load_workflow` | `m.handleLoadWorkflow` | 加载指定 workflow 的完整内容 |
| `run_workflow` | `m.handleRunWorkflow` | 执行 workflow（返回步骤内容） |

### Self-Evolution Only（5个，需 `flags.self_evolution: true`）

| Tool | Handler | Description |
|------|---------|-------------|
| `create_workflow` | `m.handleCreateWorkflow` | 创建新的 workflow 文件 |
| `update_workflow` | `m.handleUpdateWorkflow` | 更新已有 workflow |
| `delete_workflow` | `m.handleDeleteWorkflow` | 永久删除 workflow |
| `enable_workflow` | `m.handleEnableWorkflow` | 启用禁用的 workflow |
| `disable_workflow` | `m.handleDisableWorkflow` | 禁用（重命名为 `.disabled/`） |

## Context Injection

Workflow 子系统通过 `subsystem.Provider.ContextMD()` 向系统提示注入当前可用的 workflow 列表：

```
## Available Workflows

- **deploy-check**: Check deployment health
- **code-review**: Code review checklist
```

每个 workflow 的内容可通过 MCP 工具 `load_workflow` 按需加载，避免提示词膨胀。

## Configuration

```yaml
workflows:
  dir: .dolphin/workflows    # 默认值
```

在 `cmd/root.go` 中初始化时支持多目录：项目级 `.dolphin/workflows` + 用户级 `~/.dolphin/workflows`。

## Subsystem Registration

在 `cmd/root.go` 启动时注册：

```go
wfmr := workflow.NewManager(cfg.Workflows.Dir, userWorkflowDir)
subsystem.Register(wfmr)
```

实现了 `subsystem.Provider` 接口的三个方法：

| Method | Description |
|--------|-------------|
| `Name() string` | 返回 `"workflow"` |
| `ContextMD() string` | 返回可用的 workflow 列表 Markdown |
| `ToolDefs() []ToolDef` | 返回 MCP 工具定义列表 |

## CLI Commands

`dolphin workflow` 命令提供 7 个子命令：

| Command | Description |
|---------|-------------|
| `dolphin workflow list` | 列出所有 workflows |
| `dolphin workflow show <name>` | 显示指定 workflow |
| `dolphin workflow new <name>` | 创建新 workflow 文件 |
| `dolphin workflow delete <name>` | 删除 workflow |
| `dolphin workflow disable <name>` | 禁用 workflow |
| `dolphin workflow enable <name>` | 重新启用 workflow |

## 文件变更

| File | Description |
|------|-------------|
| `internal/subsystem/workflow/workflow.go` | Manager 核心实现 + 8 个工具 handler |
| `internal/subsystem/workflow/workflow_test.go` | 1156 行测试 |
| `internal/subsystem/registry.go` | Provider 接口 + 全局注册表 |
| `cmd/root.go` | 初始化 + 注册 workflow 管理器 |
| `cmd/workflow.go` | CLI 命令注册 |
| `internal/config/config_types.go` | `WorkflowsConfig` 类型 |
| `internal/config/config_gen.go` | 默认值 `.dolphin/workflows` |
| `internal/agent/agent_types.go` | `AgentDef` 的 `Workflows` 可见性字段 |
| `internal/agent/agent_pool.go` | 子代理创建时传递可见性列表 |
| `internal/agent/coordinator_tools.go` | `create_agent` 处传参 |
| `internal/skill/skill.go` | `ListForAgent` / `GetForAgent`（同模式） |

<!-- last-modified: 2026-05-24 -->
