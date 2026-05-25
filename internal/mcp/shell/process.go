package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"dolphin/internal/mcp"

	"go.uber.org/zap"
)

const (
	cleanupInterval  = 1 * time.Minute // how often to check for stale processes
	processRetention = 5 * time.Minute // keep completed process for this long before cleanup
	defOutputCap     = 64 * 1024       // default per-stream cap
)

// bgProcess represents a background shell process.
type bgProcess struct {
	PID     int
	Command string
	Cmd     *exec.Cmd

	mu       sync.Mutex
	stdout   cappedWriter
	stderr   cappedWriter
	done     chan struct{}
	exitErr  error
	startAt  time.Time
	exitedAt time.Time
}

// cappedWriter is an io.Writer that stops accepting data after a limit.
type cappedWriter struct {
	buf   bytes.Buffer
	cap   int
	trunc bool // set to true once limit is hit
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	if w.trunc {
		return len(p), nil
	}
	available := w.cap - w.buf.Len()
	if available <= 0 {
		w.trunc = true
		return len(p), nil
	}
	if len(p) > available {
		w.buf.Write(p[:available])
		w.trunc = true
		return len(p), nil
	}
	return w.buf.Write(p)
}

// processManager manages all background processes.
type processManager struct {
	mu     sync.Mutex
	procs  map[int]*bgProcess
	nextID int
	done   chan struct{} // closed on Shutdown
}

var pm = &processManager{
	procs:  make(map[int]*bgProcess),
	nextID: 1,
	done:   make(chan struct{}),
}

func init() {
	go pm.cleanupLoop()
}

// maxOutput returns the configured per-stream cap or default.
func maxOutput() int {
	return defOutputCap
}

// startBackground starts a command in the background and returns the PID.
func startBackground(cmd *exec.Cmd, command string, outputCap int) (int, error) {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	if outputCap <= 0 {
		outputCap = defOutputCap
	}

	p := &bgProcess{
		Command: command,
		Cmd:     cmd,
		done:    make(chan struct{}),
		startAt: time.Now(),
		stdout:  cappedWriter{cap: outputCap},
		stderr:  cappedWriter{cap: outputCap},
	}

	if err := cmd.Start(); err != nil {
		stdoutW.Close()
		stderrW.Close()
		return 0, fmt.Errorf("start process: %w", err)
	}

	pm.mu.Lock()
	p.PID = pm.nextID
	pm.nextID++
	pm.procs[p.PID] = p
	pm.mu.Unlock()

	// Capture stdout in background
	go func() {
		io.Copy(&p.stdout, stdoutR)
		stdoutW.Close()
	}()

	// Capture stderr in background
	go func() {
		io.Copy(&p.stderr, stderrR)
		stderrW.Close()
	}()

	// Wait for completion in background
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.exitErr = err
		p.exitedAt = time.Now()
		p.mu.Unlock()
		close(p.done)
		zap.S().Infow("background process exited", "pid", p.PID, "command", truncateCommand(command), "error", err)
	}()

	return p.PID, nil
}

// cleanupLoop periodically removes completed processes past retention.
func (pm *processManager) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-pm.done:
			return
		case <-ticker.C:
			pm.mu.Lock()
			now := time.Now()
			for pid, p := range pm.procs {
				select {
				case <-p.done:
					if now.Sub(p.exitedAt) > processRetention {
						delete(pm.procs, pid)
						zap.S().Debugw("cleaned up background process", "pid", pid)
					}
				default:
				}
			}
			pm.mu.Unlock()
		}
	}
}

// Shutdown kills all running background processes and stops the cleanup loop.
// Safe to call multiple times.
func Shutdown() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Stop the cleanup loop
	select {
	case <-pm.done:
	default:
		close(pm.done)
	}

	for pid, p := range pm.procs {
		select {
		case <-p.done:
			// already exited
		default:
			if p.Cmd != nil && p.Cmd.Process != nil {
				zap.S().Infow("killing background process", "pid", pid, "command", truncateCommand(p.Command))
				p.Cmd.Process.Kill()
			}
		}
		delete(pm.procs, pid)
	}
}

// ProcessReaderTool implements the read_process_output MCP tool.
type ProcessReaderTool struct{}

func NewProcessReaderTool() *ProcessReaderTool {
	return &ProcessReaderTool{}
}

func (t *ProcessReaderTool) Definition() mcp.ToolDefinition {
	schema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pid": map[string]any{
				"type":        "integer",
				"description": "Process ID returned by shell with background=true",
			},
			"wait": map[string]any{
				"type":        "boolean",
				"description": "If true, wait for the process to complete before returning (default: false)",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Max seconds to wait when wait=true (default: 30)",
			},
		},
		"required": []string{"pid"},
	})
	return mcp.ToolDefinition{
		Name:        "read_process_output",
		Description: "Read output from a background process started with shell(background=true). Returns stdout/stderr accumulated so far, and the process exit status if it has completed.",
		InputSchema: schema,
		Priority:    100,
		Source:      "built-in",
	}
}

func (t *ProcessReaderTool) Execute(_ context.Context, input json.RawMessage) (*mcp.ToolResult, error) {
	var params struct {
		PID     int  `json:"pid"`
		Wait    bool `json:"wait,omitempty"`
		Timeout int  `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return &mcp.ToolResult{Content: "invalid input: " + err.Error(), IsError: true}, nil
	}

	pm.mu.Lock()
	p, ok := pm.procs[params.PID]
	pm.mu.Unlock()

	if !ok {
		return &mcp.ToolResult{Content: fmt.Sprintf("process %d not found", params.PID), IsError: true}, nil
	}

	// Optionally wait for completion
	if params.Wait {
		timeout := params.Timeout
		if timeout <= 0 {
			timeout = 30
		}
		timer := time.NewTimer(time.Duration(timeout) * time.Second)
		defer timer.Stop()

		select {
		case <-p.done:
		case <-timer.C:
		}
	}

	p.mu.Lock()
	stdout := p.stdout.buf.String()
	stderr := p.stderr.buf.String()
	stdoutTrunc := p.stdout.trunc
	stderrTrunc := p.stderr.trunc
	var exitErrStr string
	select {
	case <-p.done:
		if p.exitErr != nil {
			exitErrStr = p.exitErr.Error()
		}
	default:
	}
	running := true
	select {
	case <-p.done:
		running = false
	default:
	}
	elapsed := time.Since(p.startAt).Round(time.Second).String()
	p.mu.Unlock()

	result := fmt.Sprintf("PID: %d\nCommand: %s\nRunning: %v\nElapsed: %s\n\nstdout:\n%s",
		params.PID, p.Command, running, elapsed, stdout)
	if stdoutTrunc {
		result += "\n... [stdout truncated]"
	}
	result += fmt.Sprintf("\nstderr:\n%s", stderr)
	if stderrTrunc {
		result += "\n... [stderr truncated]"
	}
	if exitErrStr != "" {
		result += fmt.Sprintf("\nexit error: %s", exitErrStr)
	}

	return &mcp.ToolResult{Content: result}, nil
}
