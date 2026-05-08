package context

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Builder assembles the system prompt from context files.
type Builder struct {
	projectDir string
	userDir    string
	systemDir  string
}

func NewBuilder() *Builder {
	home, _ := os.UserHomeDir()
	return &Builder{
		projectDir: ".dolphinzZ",
		userDir:    filepath.Join(home, ".dolphinzZ"),
		systemDir:  "/etc/dolphinzZ",
	}
}

func (b *Builder) Build() (string, error) {
	var parts []string

	// 1. PREFACE (embedded)
	parts = append(parts, DefaultPreface)

	// 2. AGENTS.md (project > user > system)
	if agents := b.loadFile("AGENTS.md"); agents != "" {
		parts = append(parts, "## Agent Definitions\n"+agents)
	}

	// 3. RULES.md
	if rules := b.loadFile("RULES.md"); rules != "" {
		parts = append(parts, "## Rules\n"+rules)
	}

	// 4. USER.md
	if user := b.loadFile("USER.md"); user != "" {
		parts = append(parts, "## User Context\n"+user)
	}

	return strings.Join(parts, "\n\n"), nil
}

func (b *Builder) loadFile(name string) string {
	for _, dir := range []string{b.projectDir, b.userDir, b.systemDir} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err == nil {
			slog.Debug("loaded context file", "file", path)
			return string(data)
		}
	}
	return ""
}
