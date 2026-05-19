package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func TestDevModeFullFlow(t *testing.T) {
	homeDir := t.TempDir()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir) // Windows: os.UserHomeDir checks USERPROFILE first
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	// Find project root (where mcp.json and skills.json live)
	rootDir := findProjectRoot(t)

	// Dev demo profile
	profile := &CareerProfile{
		Name:        "demo",
		Skills:      []string{"frontend-expert", "backend-golang"},
		MCP:         []string{"browser-preview", "filesystem"},
		Description: "Demo (integration test)",
	}

	// Use local fallback only — no network required
	fetcher := NewRepoFetcher(filepath.Join(homeDir, UserConfigDir, "cache"))
	fetcher.SetLocalDir(rootDir)

	skills, mcp := AugmentWithRepos(profile, []string{"dolphinv/skills"}, []string{"dolphinv/mcp"})

	t.Logf("matched skills: %d, mcp: %d", len(skills), len(mcp))

	if len(skills) == 0 {
		t.Error("expected at least 1 matched skill from local skills.json")
	}
	if len(mcp) == 0 {
		t.Error("expected at least 1 matched mcp server from local mcp.json")
	}

	for _, s := range skills {
		t.Logf("  skill: %s (url=%s)", s.Name, s.URL)
	}
	for _, m := range mcp {
		t.Logf("  mcp:   %s (cmd=%s args=%v)", m.Name, m.Command, m.Args)
	}

	// Apply tools
	if err := ApplyTools(skills, mcp); err != nil {
		t.Fatalf("ApplyTools: %v", err)
	}

	// Verify skills directory
	skillsDir := filepath.Join(homeDir, UserConfigDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("read skills dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no skill files created")
	}
	for _, e := range entries {
		t.Logf("  skill file: %s", e.Name())
	}

	// Verify MCP config
	configPath := filepath.Join(homeDir, UserConfigDir, ConfigFileName+".yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configStr := string(data)
	t.Logf("config.yaml:\n%s", configStr)

	if !strings.Contains(configStr, "browser-preview") {
		t.Error("config should contain browser-preview server")
	}
	if !strings.Contains(configStr, "stdio") {
		t.Error("MCP server should be stdio type")
	}
}

func TestDevModeNoDuplicates(t *testing.T) {
	homeDir := t.TempDir()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir) // Windows: os.UserHomeDir checks USERPROFILE first
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	rootDir := findProjectRoot(t)

	profile := &CareerProfile{
		Name:        "demo",
		Skills:      []string{"browser-preview", "filesystem"},
		MCP:         []string{"browser-preview", "filesystem"},
		Description: "Demo",
	}

	fetcher := NewRepoFetcher(filepath.Join(homeDir, UserConfigDir, "cache"))
	fetcher.SetLocalDir(rootDir)

	_, mcp := AugmentWithRepos(profile, []string{}, []string{"dolphinv/mcp"})

	// Apply twice — no duplicates in config
	if err := ApplyTools(nil, mcp); err != nil {
		t.Fatalf("ApplyTools 1: %v", err)
	}
	if err := ApplyTools(nil, mcp); err != nil {
		t.Fatalf("ApplyTools 2: %v", err)
	}

	configPath := filepath.Join(homeDir, UserConfigDir, ConfigFileName+".yaml")
	data, _ := os.ReadFile(configPath)
	count := strings.Count(string(data), "browser-preview")
	if count > 2 { // once in servers map key, once in type/command
		t.Errorf("browser-preview appears %d times, expected no duplicates", count)
	}
}

func TestDevModeDemoSkillsDownload(t *testing.T) {
	// This test exercises the full remote flow with dolphinZzv/demo_skills
	homeDir := t.TempDir()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir) // Windows: os.UserHomeDir checks USERPROFILE first
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	profile := &CareerProfile{
		Name:        "demo",
		Skills:      []string{"demo-skill"},
		MCP:         []string{},
		Description: "Demo download test",
	}

	skills, _ := AugmentWithRepos(profile, []string{"dolphinZzv/demo_skills"}, []string{})
	if len(skills) == 0 {
		t.Skip("demo_skills repo not reachable (network issue)")
	}
	t.Logf("matched %d skills from demo_skills repo", len(skills))
	for _, s := range skills {
		t.Logf("  %s: url=%s", s.Name, s.URL)
	}

	if err := ApplyTools(skills, nil); err != nil {
		t.Fatalf("ApplyTools: %v", err)
	}

	// Verify demo-skill/SKILL.md was created in skills dir
	skillsDir := filepath.Join(homeDir, UserConfigDir, "skills")
	skillPath := filepath.Join(skillsDir, "demo-skill", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("demo-skill/SKILL.md not created: %v", err)
	}
	if !strings.Contains(string(data), "Demo Skill") {
		t.Errorf("skill content does not contain expected text, got: %s", string(data)[:200])
	}
	t.Logf("downloaded skill content:\n%s", string(data)[:200])
}

func TestSyncConfigIntegration(t *testing.T) {
	origDir := ProjectConfigDir
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		ProjectConfigDir = origDir
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})

	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir)

	projectDir := t.TempDir()
	ProjectConfigDir = projectDir

	// Write a minimal config — missing most optional sections
	minimal := `sync_config: true
llm:
  type: openai
  base_url: https://api.deepseek.com
  api_key: test-key-for-integration
  model: deepseek-v4-flash
  max_tokens: 4096
  max_context_tokens: 128000
  temperature: 0.7
  max_sub_turns: 10
session:
  max_loop: 50
mcp:
  shell:
    enabled: true
    timeout_seconds: 30
    priority: 10
agent_pool:
  max_concurrency: 5
  default_timeout: 300
  max_pending_results: 10
`
	cfgPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(minimal), 0600); err != nil {
		t.Fatal(err)
	}

	// Load config — triggers fillConfigDefaults
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// 1. Verify loaded config has correct values from the file
	if cfg.LLM.Model != "deepseek-v4-flash" {
		t.Errorf("llm.model = %q, want deepseek-v4-flash", cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("llm.max_tokens = %d, want 4096", cfg.LLM.MaxTokens)
	}
	if cfg.Session.MaxLoop != 50 {
		t.Errorf("session.max_loop = %d, want 50", cfg.Session.MaxLoop)
	}
	if !cfg.SyncConfig {
		t.Error("sync_config should be true")
	}

	// 2. Verify config file was updated with missing sections
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	expectedSections := []string{
		"crontab:",
		"diary:",
		"skills:",
		"pprof:",
		"metrics:",
		"update:",
		"resource:",
		"transport:",
	}
	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("config file missing section: %s", section)
		}
	}

	// 3. Verify existing values were NOT overwritten
	if !strings.Contains(content, "test-key-for-integration") {
		t.Error("api_key should be preserved in config file")
	}
	if !strings.Contains(content, "deepseek-v4-flash") {
		t.Error("model should be preserved in config file")
	}
	// Verify the output is valid YAML with expected structure
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML after fill: %v", err)
	}
	llm, ok := parsed["llm"].(map[string]any)
	if !ok {
		t.Fatal("llm section missing from parsed output")
	}
	if llm["model"] != "deepseek-v4-flash" {
		t.Errorf("llm.model = %v, want deepseek-v4-flash (preserved)", llm["model"])
	}

	// 4. Verify it can be loaded again without error
	cfg2, err := Load("")
	if err != nil {
		t.Fatalf("second Load() error: %v", err)
	}
	if cfg2.LLM.Model != "deepseek-v4-flash" {
		t.Errorf("after re-load: llm.model = %q", cfg2.LLM.Model)
	}
}

func TestSyncConfigIntegrationDisabled(t *testing.T) {
	origDir := ProjectConfigDir
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		ProjectConfigDir = origDir
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})

	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir)

	projectDir := t.TempDir()
	ProjectConfigDir = projectDir

	// Write config with sync_config: false — missing sections should NOT be added
	config := `sync_config: false
llm:
  type: openai
  api_key: test-key
  model: gpt-4
  max_tokens: 4096
  max_context_tokens: 128000
  temperature: 0.7
  max_sub_turns: 10
session:
  max_loop: 50
mcp:
  shell:
    enabled: true
    timeout_seconds: 30
    priority: 10
agent_pool:
  max_concurrency: 5
  default_timeout: 300
  max_pending_results: 10
`
	cfgPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.SyncConfig {
		t.Error("SyncConfig should be false")
	}

	// Config file should NOT have been modified
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "crontab:") {
		t.Error("crontab section should NOT have been added when sync_config=false")
	}
}

func TestSyncConfigIntegrationNoFile(t *testing.T) {
	// Load with no config file — should use all defaults
	origDir := ProjectConfigDir
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		ProjectConfigDir = origDir
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})

	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir)

	projectDir := t.TempDir()
	ProjectConfigDir = projectDir

	// No config file at all
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() with no config file error: %v", err)
	}

	// Default values should be used
	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("default llm.model = %q, want gpt-4o (no config file)", cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("default llm.max_tokens = %d, want 4096", cfg.LLM.MaxTokens)
	}
	if cfg.Session.MaxLoop != 50 {
		t.Errorf("default session.max_loop = %d, want 50", cfg.Session.MaxLoop)
	}
	if !cfg.Transport.Stdio.Enabled {
		t.Error("default transport.stdio.enabled should be true")
	}
	// fillConfigDefaults should not create a file if none existed
	if _, err := os.Stat(filepath.Join(projectDir, "config.yaml")); err == nil {
		t.Error("config.yaml should not have been created when it didn't exist")
	}
}

func TestSyncConfigIntegrationFullConfig(t *testing.T) {
	origDir := ProjectConfigDir
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		ProjectConfigDir = origDir
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})

	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir)

	projectDir := t.TempDir()
	ProjectConfigDir = projectDir

	// Build a full config from viper defaults (all keys present)
	v := viper.New()
	setDefaults(v)

	// Override with test-specific values
	v.Set("llm.api_key", "test-key")
	v.Set("llm.base_url", "https://api.deepseek.com")
	v.Set("llm.model", "deepseek-v4-flash")
	v.Set("llm.type", "openai")

	full := v.AllSettings()
	data, err := yaml.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Capture content before Load
	before, _ := os.ReadFile(cfgPath)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.SyncConfig {
		t.Error("SyncConfig should be true")
	}

	// Config should not have been modified (all keys already present)
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// Compare YAML content (unmarshal both for semantic comparison)
	var beforeMap, afterMap map[string]any
	yaml.Unmarshal(before, &beforeMap)
	yaml.Unmarshal(after, &afterMap)

	beforeJSON, _ := json.Marshal(beforeMap)
	afterJSON, _ := json.Marshal(afterMap)
	if string(beforeJSON) != string(afterJSON) {
		t.Error("config file should not be modified when it already has all defaults")
	}
}

func TestSyncConfigIntegrationRepeatedLoads(t *testing.T) {
	origDir := ProjectConfigDir
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		ProjectConfigDir = origDir
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})

	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	os.Setenv("USERPROFILE", homeDir)

	projectDir := t.TempDir()
	ProjectConfigDir = projectDir

	// Write minimal config
	minimal := `sync_config: true
llm:
  type: openai
  api_key: test-key
  model: gpt-4
  max_tokens: 4096
  max_context_tokens: 128000
  temperature: 0.7
  max_sub_turns: 10
session:
  max_loop: 50
mcp:
  shell:
    enabled: true
    timeout_seconds: 30
    priority: 10
agent_pool:
  max_concurrency: 5
  default_timeout: 300
  max_pending_results: 10
`
	cfgPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(minimal), 0600); err != nil {
		t.Fatal(err)
	}

	// Load 3 times — each should succeed without error
	for i := 0; i < 3; i++ {
		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load() iteration %d error: %v", i, err)
		}
		if cfg.LLM.Model != "gpt-4" {
			t.Errorf("iteration %d: llm.model = %q", i, cfg.LLM.Model)
		}
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "mcp.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project root not found")
		}
		dir = parent
	}
}
