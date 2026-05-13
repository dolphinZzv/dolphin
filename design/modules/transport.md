# Transport Layer (`internal/transport/`)

## Interfaces

```go
type Transport interface {
    Name() string
    Start(ctx) error   // 阻塞直到会话结束
    Close() error
}

type UserIO interface {
    ReadLine() (string, error)
    WriteLine(string) error
    WriteString(string) error
    Capabilities() Capabilities
    Context() context.Context
}
```

每个 Transport 绑定一个独立的 Coordinator goroutine。

## Implementations

| Transport | Library | Mechanism | Capabilities |
|-----------|---------|-----------|-------------|
| **stdio** | `chzyer/readline` | stdin/stdout 行编辑 | 全部支持 |
| **SSH** | `golang.org/x/crypto/ssh` | TCP :2222, 密码认证 | 全部支持 |
| **MQTT** | `eclipse/paho.mqtt.golang` | Subscribe command topic, Publish response topic | 非流式 |
| **Email** | `net/smtp` + `emersion/go-imap` | SMTP 发送, IMAP 轮询, subject → 命令 | 非流式, 需确认 |
