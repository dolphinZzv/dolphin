# MQTT Transport

The MQTT transport connects Dolphin to an MQTT broker, enabling agent interaction through publish/subscribe messaging. Useful for IoT integrations and event-driven workflows.

## Configuration

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

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable MQTT transport |
| `broker` | string | `"tcp://localhost:1883"` | MQTT broker URL |
| `subscribe_topic` | string | `"dolphin/in"` | Topic to subscribe for incoming messages |
| `publish_topic` | string | `"dolphin/out"` | Topic to publish responses to |
| `client_id` | string | `"dolphin-agent"` | MQTT client ID |
| `username` | string | `""` | MQTT username (if broker requires auth) |
| `password` | string | `""` | MQTT password |

## Usage

### 1. Enable in config

```yaml
transport:
  mqtt:
    enabled: true
    broker: tcp://localhost:1883
    subscribe_topic: dolphin/in
    publish_topic: dolphin/out
```

### 2. Start Dolphin

```bash
dolphin
```

### 3. Subscribe to response topic

```bash
mosquitto_sub -t dolphin/out
```

### 4. Publish a message

```bash
mosquitto_pub -t dolphin/in -m "What time is it?"
```

The response will appear in the `mosquitto_sub` terminal.

## Security Notes

- Use TLS-enabled brokers (`tcps://` or `mqtts://`) for production
- Always set `username`/`password` when the broker requires authentication
- The embedded MQTT broker (see `ServersConfig`) provides a lightweight option for local development

---

> Last modified: 2026-05-22
