package transport

import (
	"io"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// MultiIO wraps a UserIO and manages per-agent IO channels.
// It intercepts @agentname-prefixed lines from ReadLine and routes them
// to the corresponding agent's inbox via OnUserMessage callback.
//
// Non-@ lines are returned as-is. @-lines for unknown agents are also
// returned as-is (not consumed).
type MultiIO struct {
	root   UserIO
	agents map[string]*agentChannel
	mu     sync.RWMutex

	// OnUserMessage is called when an @-message is intercepted.
	// Set by Coordinator to dispatch tasks via pool.Dispatch.
	OnUserMessage func(agentName, message string)
}

type agentChannel struct {
	name    string
	inputCh chan string
}

// NewMultiIO creates a MultiIO wrapping the given root UserIO.
func NewMultiIO(root UserIO) *MultiIO {
	return &MultiIO{
		root:   root,
		agents: make(map[string]*agentChannel),
	}
}

// ReadLine reads from root. If the line starts with "@agentname ", the
// remainder is routed to the agent's inbox and OnUserMessage is called.
// Non-@ lines and @-lines for unknown agents are returned as-is.
func (m *MultiIO) ReadLine() (string, error) {
	for {
		line, err := m.root.ReadLine()
		if err != nil {
			return line, err
		}

		agentName, message, ok := parseAgentMessage(line)
		if !ok {
			return line, nil
		}

		m.mu.RLock()
		ch, exists := m.agents[agentName]
		m.mu.RUnlock()

		if !exists {
			// Unknown agent — return as-is
			return line, nil
		}

		// Route to agent's inbox (non-blocking, drop if full)
		select {
		case ch.inputCh <- message:
		default:
			zap.S().Warnw("agent inbox full, dropping @-message",
				"agent", agentName,
				"cap", cap(ch.inputCh),
			)
		}

		// Notify coordinator to dispatch as a task
		if m.OnUserMessage != nil {
			m.OnUserMessage(agentName, message)
		}

		// Return empty to trigger result checking in coordinator
		// (the @-message was consumed; coordinator checks pending results)
		return "", nil
	}
}

// parseAgentMessage checks if line is "@agentname message" and extracts parts.
// Returns agent name, message, and whether it matched.
func parseAgentMessage(line string) (agentName, message string, ok bool) {
	if !strings.HasPrefix(line, "@") {
		return "", "", false
	}
	// Must be "@name ..." with a space
	spaceIdx := strings.IndexByte(line, ' ')
	if spaceIdx <= 1 {
		return "", "", false
	}
	agentName = line[1:spaceIdx]
	message = strings.TrimSpace(line[spaceIdx+1:])
	if agentName == "" || message == "" {
		return "", "", false
	}
	return agentName, message, true
}

func (m *MultiIO) WriteLine(s string) error   { return m.root.WriteLine(s) }
func (m *MultiIO) WriteString(s string) error { return m.root.WriteString(s) }
func (m *MultiIO) Flush() error               { return m.root.Flush() }
func (m *MultiIO) Capabilities() Capabilities { return m.root.Capabilities() }
func (m *MultiIO) Context() string            { return m.root.Context() }
func (m *MultiIO) Name() string               { return m.root.Name() }

// RegisterAgent creates a new SubAgentIO for the given agent name and
// registers it in the routing table. The input channel has capacity 64.
func (m *MultiIO) RegisterAgent(name string) *SubAgentIO {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := &agentChannel{
		name:    name,
		inputCh: make(chan string, 64),
	}
	m.agents[name] = ch

	zap.S().Infow("agent registered in MultiIO", "name", name)
	return &SubAgentIO{ch: ch, root: m.root, caps: m.root.Capabilities()}
}

// UnregisterAgent removes an agent from the routing table.
func (m *MultiIO) UnregisterAgent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, name)
}

// AgentNames returns the list of registered agent names.
func (m *MultiIO) AgentNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}
	return names
}

// WriteToUser writes a message to the user with the agent name prefix.
// Format: "[name] message"
func (m *MultiIO) WriteToUser(agentName, message string) error {
	return m.root.WriteLine("[" + agentName + "] " + message)
}

// SubAgentIO implements UserIO for a subagent.
// For streaming transports, WriteLine/WriteString forward to root with [name] prefix.
// For non-streaming transports, writes are buffered and only sent on Flush().
// ReadLine reads from the agent's inbox (receives @-routed messages).
type SubAgentIO struct {
	ch        *agentChannel
	root      UserIO // root IO for writes
	caps      Capabilities
	buf       strings.Builder
	mu        sync.Mutex
	lineStart bool // true if next WriteString should be prefixed
}

func (s *SubAgentIO) ReadLine() (string, error) {
	msg, ok := <-s.ch.inputCh
	if !ok {
		return "", io.ErrClosedPipe
	}
	return msg, nil
}

func (s *SubAgentIO) WriteString(msg string) error {
	if msg == "" {
		return nil
	}
	if s.caps.Streaming {
		return s.writePrefixed(msg)
	}
	s.mu.Lock()
	s.writePrefixedToBuf(msg)
	s.mu.Unlock()
	return nil
}

func (s *SubAgentIO) WriteLine(msg string) error {
	s.lineStart = true
	if s.caps.Streaming {
		return s.root.WriteLine("[" + s.ch.name + "] " + msg)
	}
	s.mu.Lock()
	s.buf.WriteString("[" + s.ch.name + "] " + msg + "\n")
	s.mu.Unlock()
	return nil
}

func (s *SubAgentIO) Flush() error {
	s.lineStart = true
	if s.caps.Streaming {
		return s.root.Flush()
	}
	s.mu.Lock()
	content := s.buf.String()
	s.buf.Reset()
	s.mu.Unlock()
	if content == "" {
		return nil
	}
	return s.root.WriteString(content)
}

// writePrefixed writes msg to root with [name] line prefixes for streaming transports.
func (s *SubAgentIO) writePrefixed(msg string) error {
	var sb strings.Builder
	for _, r := range msg {
		if s.lineStart {
			sb.WriteString("[" + s.ch.name + "] ")
			s.lineStart = false
		}
		sb.WriteRune(r)
		if r == '\n' {
			s.lineStart = true
		}
	}
	return s.root.WriteString(sb.String())
}

// writePrefixedToBuf writes msg to internal buffer with [name] line prefixes.
func (s *SubAgentIO) writePrefixedToBuf(msg string) {
	for _, r := range msg {
		if s.lineStart {
			s.buf.WriteString("[" + s.ch.name + "] ")
			s.lineStart = false
		}
		s.buf.WriteRune(r)
		if r == '\n' {
			s.lineStart = true
		}
	}
}

func (s *SubAgentIO) Capabilities() Capabilities { return s.caps }
func (s *SubAgentIO) Context() string            { return "" }
func (s *SubAgentIO) Name() string               { return "[" + s.ch.name + "]" }
