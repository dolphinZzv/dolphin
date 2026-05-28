# DingTalk 传输层

DingTalk transport 将 Dolphin 与[阿里钉钉](https://www.dingtalk.com/)集成，通过钉钉开放平台的 Bot API 进行通信。

## 配置

```yaml
transport:
  dingtalk:
    enabled: false
    client_id: ""              # 钉钉开放平台的 AppKey
    client_secret: ""          # 钉钉开放平台的 AppSecret
    startup_webhook: ""        # 启动通知的钉钉群机器人 Webhook URL（可选）
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用 DingTalk transport |
| `client_id` | string | `""` | 钉钉开放平台 AppKey |
| `client_secret` | string | `""` | 钉钉开放平台 AppSecret |
| `startup_webhook` | string | `""` | 可选。钉钉群机器人 Webhook URL，transport 启动时自动发送通知到指定群 |

## 启动通知

配置 `startup_webhook` 后，每次 DingTalk Stream 连接建立时（包括初始启动和凭据变更后重连），会自动向指定钉钉群发送一条启动通知消息。

示例（需在群机器人中添加自定义机器人获取 Webhook URL）：

```yaml
transport:
  dingtalk:
    enabled: true
    client_id: "your_app_key"
    client_secret: "your_app_secret"
    startup_webhook: "https://oapi.dingtalk.com/robot/send?access_token=your_token"
```

## 设置步骤

1. 在[钉钉开放平台](https://open-dev.dingtalk.com/)创建 Bot 应用
2. 开启 **消息推送** 功能
3. 获取 **AppKey**（即 `client_id`）和 **AppSecret**（即 `client_secret`）
4. 配置到 `transport.dingtalk` 中

## 使用方式

配置并启用后，Dolphin 会自动响应发送给钉钉 Bot 的消息。

## 相关链接

- [钉钉开放平台控制台](https://open-dev.dingtalk.com/)
- [钉钉机器人开发文档](https://open.dingtalk.com/document/orgapp/robot-overview)
- [AppKey 和 AppSecret 获取指南](https://open.dingtalk.com/document/orgapp/create-an-application)

## 参考

- [A2A 传输层](a2a.md) — Google Agent-to-Agent 协议

---

> 最后更新: 2026-05-22
