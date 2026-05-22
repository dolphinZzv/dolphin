package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			wfDirs = append([]string{userWfDir}, wfDirs...)
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
