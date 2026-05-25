package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dolphin/internal/agent/provider"
	"dolphin/internal/config"
	"dolphin/internal/i18n"
	workflowpkg "dolphin/internal/subsystem/workflow"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowShort),
		RunE:  runWorkflowList,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowListUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowListShort),
		RunE:  runWorkflowList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowShowUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowShowShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runWorkflowShow,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowNewUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowNewShort),
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runWorkflowNew,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowInitUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowInitShort),
		Args:  cobra.RangeArgs(0, 2),
		RunE:  runWorkflowInit,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowDeleteUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowDeleteShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runWorkflowDelete,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowDisableUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowDisableShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runWorkflowDisable,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdWorkflowEnableUse),
		Short: i18n.TL(i18n.KeyCmdWorkflowEnableShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runWorkflowEnable,
	})

	return cmd
}

func loadWorkflowCmdConfig() (*config.Config, *workflowpkg.Manager, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	wfDirs := []string{cfg.Workflows.Dir}
	if homeDir, err := os.UserHomeDir(); err == nil {
		userWfDir := filepath.Join(homeDir, config.UserConfigDir, "workflows")
		if userWfDir != cfg.Workflows.Dir {
			wfDirs = append(wfDirs, userWfDir)
		}
	}
	mgr := workflowpkg.NewManager(wfDirs...)
	if err := mgr.Load(); err != nil {
		return nil, nil, fmt.Errorf("load workflows: %w", err)
	}
	return cfg, mgr, nil
}

func runWorkflowList(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadWorkflowCmdConfig()
	if err != nil {
		return err
	}

	wfs := mgr.List()
	if len(wfs) == 0 {
		fmt.Println(i18n.TL(i18n.KeyWorkflowCLINone))
		return nil
	}

	zap.S().Infow("listed workflows", "count", len(wfs))

	fmt.Printf("%-30s %s\n", "NAME", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))
	for _, w := range wfs {
		desc := w.Description
		if len(desc) > 45 {
			desc = desc[:42] + "..."
		}
		fmt.Printf("%-30s %s\n", w.Name, desc)
	}
	fmt.Printf(i18n.TL(i18n.KeyWorkflowCLITotal)+"\n", len(wfs))
	return nil
}

func runWorkflowShow(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadWorkflowCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]
	w, ok := mgr.Get(name)
	if !ok {
		return fmt.Errorf("workflow %q not found", name)
	}

	fmt.Printf("--- %s ---\n", w.Name)
	if w.Description != "" {
		fmt.Printf("Description: %s\n\n", w.Description)
	}
	fmt.Println(w.Content)
	return nil
}

func runWorkflowNew(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	name := args[0]
	description := name
	if len(args) > 1 {
		description = args[1]
	}

	mgr := workflowpkg.NewManager(cfg.Workflows.Dir)
	if err := mgr.Load(); err != nil {
		return fmt.Errorf("load workflows: %w", err)
	}

	if err := mgr.NewTemplate(name, description); err != nil {
		return fmt.Errorf("create workflow: %w", err)
	}

	zap.S().Infow("created workflow", "name", name, "dir", cfg.Workflows.Dir)
	fmt.Printf(i18n.TL(i18n.KeyWorkflowCLICreated)+"\n", name, cfg.Workflows.Dir)
	fmt.Println(i18n.TL(i18n.KeyWorkflowCLIEdit))
	return nil
}

func runWorkflowDelete(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadWorkflowCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]

	if _, ok := mgr.Get(name); !ok {
		return fmt.Errorf("workflow %q not found", name)
	}

	if err := mgr.Unregister(name); err != nil {
		return fmt.Errorf("delete workflow: %w", err)
	}

	zap.S().Infow("deleted workflow", "name", name)
	fmt.Printf(i18n.TL(i18n.KeyWorkflowCLIDeleted)+"\n", name)
	return nil
}

func runWorkflowDisable(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadWorkflowCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]

	if err := mgr.Disable(name); err != nil {
		return fmt.Errorf("disable workflow: %w", err)
	}

	zap.S().Infow("disabled workflow", "name", name)
	fmt.Printf(i18n.TL(i18n.KeyWorkflowCLIDisabled)+"\n", name)
	return nil
}

func runWorkflowEnable(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadWorkflowCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]

	if err := mgr.Enable(name); err != nil {
		return fmt.Errorf("enable workflow: %w", err)
	}

	zap.S().Infow("enabled workflow", "name", name)
	fmt.Printf(i18n.TL(i18n.KeyWorkflowCLIEnabled)+"\n", name)
	return nil
}

func runWorkflowInit(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Get LLM provider
	providers := cfg.LLM.EffectiveProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no LLM provider configured")
	}
	provCfg := &providers[0]
	prov := provider.NewProviderFromConfig(provCfg)

	// ---- Gather project context ----
	var ctxBuf strings.Builder

	// 1. Project info
	ctxBuf.WriteString("## Project\n\n")
	if moduleBytes, err := os.ReadFile("go.mod"); err == nil {
		for _, line := range strings.Split(string(moduleBytes), "\n") {
			if strings.HasPrefix(line, "module ") {
				ctxBuf.WriteString(fmt.Sprintf("- Module: %s\n", strings.TrimPrefix(line, "module ")))
				break
			}
		}
	}
	absRoot, _ := filepath.Abs(".")
	ctxBuf.WriteString(fmt.Sprintf("- Root: %s\n\n", absRoot))

	// 2. Codebase directory tree (3 levels)
	ctxBuf.WriteString("## Codebase Structure\n\n")
	ctxBuf.WriteString(buildTree(".", 0, 3))
	ctxBuf.WriteString("\n")

	// 3. Reference workflow docs
	ctxBuf.WriteString("## Reference Workflow Docs\n\n")
	if entries, err := os.ReadDir("workflow"); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				data, err := os.ReadFile(filepath.Join("workflow", e.Name()))
				if err == nil {
					ctxBuf.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", e.Name(), string(data)))
				}
			}
		}
	}

	// 4. Existing workflows as format examples
	ctxBuf.WriteString("## Existing Workflows\n\n")
	if wfDir := cfg.Workflows.Dir; wfDir != "" {
		if wfEntries, err := os.ReadDir(wfDir); err == nil {
			for _, wfEntry := range wfEntries {
				if !wfEntry.IsDir() {
					continue
				}
				wfData, err := os.ReadFile(filepath.Join(wfDir, wfEntry.Name(), "WORKFLOW.md"))
				if err != nil {
					continue
				}
				ctxBuf.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", wfEntry.Name(), string(wfData)))
			}
		}
	}

	// 5. CLAUDE.md guidelines
	if data, err := os.ReadFile("CLAUDE.md"); err == nil {
		ctxBuf.WriteString("## Project Guidelines (CLAUDE.md)\n\n")
		ctxBuf.Write(data)
		ctxBuf.WriteString("\n\n")
	}

	projectContext := ctxBuf.String()

	// ---- Build prompt ----
	var systemPrompt string
	var userPrompt string

	if len(args) == 0 {
		// No-name mode: auto-discover multiple workflows
		systemPrompt = `You are a workflow generator for the Dolphin AI agent project.
Generate WORKFLOW.md files with YAML frontmatter (name, description) and detailed markdown steps.
Steps must be actionable and specific to the project context — not generic placeholders.`

		userPrompt = fmt.Sprintf(`Analyze this project and suggest 3-5 useful workflows that would be valuable.

For each workflow, provide:
1. A short kebab-case name
2. A brief description
3. Full WORKFLOW.md content with YAML frontmatter and detailed steps

Separate each workflow with "---WORKFLOW---" on its own line.

Project context:
%s`, projectContext)
	} else {
		// Named mode: generate a single workflow
		name := args[0]
		description := name
		if len(args) > 1 {
			description = args[1]
		}

		systemPrompt = `You are a workflow generator for the Dolphin AI agent project.
Generate a WORKFLOW.md file with YAML frontmatter (name, description) and detailed markdown steps.
Steps must be actionable and specific to the project context — not generic placeholders.`

		userPrompt = fmt.Sprintf(`Generate a workflow named "%s" with description "%s".

Output only the WORKFLOW.md content with YAML frontmatter (name, description fields).

Project context:
%s`, name, description, projectContext)
	}

	// ---- Call LLM ----
	zap.S().Info("calling LLM for workflow generation")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := provider.ProviderRequest{
		Messages: []provider.Message{
			{Role: "user", Content: provider.TextContent(userPrompt)},
		},
		System:    systemPrompt,
		MaxTokens: 8192,
		Model:     provCfg.Model,
	}

	resp, err := prov.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract text from response
	var blocks []map[string]any
	responseText := ""
	if err := json.Unmarshal(resp.Content, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if t, ok := b["text"].(string); ok {
				parts = append(parts, t)
			}
		}
		responseText = strings.Join(parts, "")
	} else {
		responseText = string(resp.Content)
	}
	responseText = strings.TrimSpace(responseText)
	if responseText == "" {
		return fmt.Errorf("LLM returned empty response")
	}
	zap.S().Debugw("LLM response", "text", responseText[:min(len(responseText), 500)])

	// ---- Save results ----
	mgr := workflowpkg.NewManager(cfg.Workflows.Dir)
	if err := mgr.Load(); err != nil {
		return fmt.Errorf("load workflows: %w", err)
	}

	if len(args) == 0 {
		// Parse and save multiple workflows
		parts := strings.Split(responseText, "---WORKFLOW---")
		var generated int
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			name := extractWorkflowName(part)
			if name == "" {
				continue
			}
			desc := extractWorkflowDescription(part)
			if err := mgr.Register(name, desc, part); err != nil {
				zap.S().Warnw("failed to save generated workflow", "name", name, "error", err)
				fmt.Printf("  ✗ %s: %v\n", name, err)
				continue
			}
			zap.S().Infow("generated workflow", "name", name)
			fmt.Printf("  ✓ %s: %s\n", name, desc)
			generated++
		}
		if generated == 0 {
			return fmt.Errorf("no valid workflows generated")
		}
		fmt.Printf(i18n.TL(i18n.KeyWorkflowCLITotal)+"\n", generated)
	} else {
		name := args[0]
		description := name
		if len(args) > 1 {
			description = args[1]
		}
		if err := mgr.Register(name, description, responseText); err != nil {
			return fmt.Errorf("save workflow: %w", err)
		}
		zap.S().Infow("generated workflow", "name", name)
		fmt.Printf(i18n.TL(i18n.KeyCmdWorkflowInitComplete)+"\n", name)
	}

	return nil
}

// buildTree returns a directory tree string up to maxDepth levels.
func buildTree(dir string, depth, maxDepth int) string {
	if depth > maxDepth {
		return ""
	}
	var sb strings.Builder
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden and common generated directories
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}
		sb.WriteString(strings.Repeat("  ", depth))
		sb.WriteString(fmt.Sprintf("- %s/\n", name))
		sb.WriteString(buildTree(filepath.Join(dir, name), depth+1, maxDepth))
	}
	return sb.String()
}

// extractWorkflowName parses the name field from YAML frontmatter.
func extractWorkflowName(content string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	rest := content[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx <= 0 {
		return ""
	}
	frontmatter := rest[:endIdx]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			return strings.TrimSpace(line[5:])
		}
	}
	return ""
}

// extractWorkflowDescription parses the description field from YAML frontmatter.
func extractWorkflowDescription(content string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	rest := content[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx <= 0 {
		return ""
	}
	frontmatter := rest[:endIdx]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(line[12:])
		}
	}
	return ""
}
