package transport

import "context"

// Transport represents a user connection transport.
// Each Transport instance corresponds to one user session.
type Transport interface {
	// Name returns a human-readable name ("stdio", "ssh", "mqtt").
	Name() string

	// Start initiates the transport and blocks until the session ends.
	Start(ctx context.Context) error

	// Close terminates the transport.
	Close() error
}

// UserIO provides readline-based interactive I/O for the agent loop.
// Both StdioTransport and SSHSession implement this interface.
type UserIO interface {
	ReadLine() (string, error)
	WriteLine(string) error
	WriteString(string) error
}
