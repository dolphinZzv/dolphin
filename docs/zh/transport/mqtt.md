# MQTT 传输层

MQTT transport 将 Dolphin 连接到 MQTT Broker，通过发布/订阅消息实现 Agent 交互。适用于 IoT 集成和事件驱动的工作流。

## 配置

```yaml
transport:
  mqtt:
    enabled: false
    broker: tcp://localhost:1883
    subscribe_topic: dolphin/in
    publish_topic: dolphin/out
    client_id: dolphin-agent
    username: ""
    password: ""
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用 MQTT transport |
| `broker` | string | `"tcp://localhost:1883"` | MQTT Broker 地址 |
| `subscribe_topic` | string | `"dolphin/in"` | 订阅主题（接收消息） |
| `publish_topic` | string | `"dolphin/out"` | 发布主题（发送响应） |
| `client_id` | string | `"dolphin-agent"` | MQTT 客户端 ID |
| `username` | string | `""` | MQTT 用户名 |
| `password` | string | `""` | MQTT 密码 |

## 使用方式

### 1. 启用配置

```yaml
transport:
  mqtt:
    enabled: true
    broker: tcp://localhost:1883
    subscribe_topic: dolphin/in
    publish_topic: dolphin/out
```

### 2. 启动 Dolphin

```bash
dolphin
```

### 3. 订阅响应主题

```bash
mosquitto_sub -t dolphin/out
```

### 4. 发布消息

```bash
mosquitto_pub -t dolphin/in -m "几点了？"
```

响应会出现在 `mosquitto_sub` 终端中。

## 安全说明

- 生产环境建议使用 TLS 加密的 Broker（`tcps://` 或 `mqtts://`）
- 配置 `username`/`password` 以使用 Broker 鉴权
- 嵌入式的 MQTT Broker（参见 `ServersConfig`）可用于本地开发

---

> 最后更新: 2026-05-22
