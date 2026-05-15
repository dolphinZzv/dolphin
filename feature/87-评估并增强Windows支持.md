# 评估并增强 Windows 支持

## 来源
- Chick Issue #87 (编号 61)
- 标题: "评估并增强 Windows 支持"

## 需求描述
评估当前 Dolphin 项目在 Windows 平台上的兼容性，识别不兼容的代码点，并实施增强使项目能够在 Windows 上正常运行。

## 当前状态（调研结果）
- **可以构建**: GoReleaser 已配置 Windows 构建目标，CGO_ENABLED=0 跨编译正常
- **运行时问题**:
  1. Shell MCP 工具硬编码 `sh -c`，Windows 无原生 sh
  2. Script 插件硬编码 `sh` 作为解释器
  3. Session/temp 路径硬编码 `/tmp/dolphin`，Windows 无 /tmp
  4. 系统配置路径硬编码 `/etc/dolphin`，Windows 产生无效路径
  5. `syscall.SIGTERM` 在 Windows 上行为不同（有降级 fallback）
  6. `SHELL` 环境变量在 Windows 上为空
  7. 多处测试硬编码 Unix 路径（/tmp）
- **CI**: 无 Windows runner，无 Windows 测试覆盖
- **测试**: 用户将用 GitHub Actions 或本地 Windows 测试

## 范围界定
- 优先修复运行时阻塞问题（1-6）
- 为关键修改补充单元测试
- CI 加 Windows runner 作为后续
- 用户可以在本地 Windows 机器上验证

## 设计
见 `design/modules/windows-support.md`
