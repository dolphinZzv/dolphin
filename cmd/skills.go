package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dolphin/internal/config"
	"dolphin/internal/i18n"
	"dolphin/internal/skill"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type skillResult struct {
	Name        string
	Description string
	Source      string // "local" or repo name
	Installed   bool
}

func NewSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsUse),
		Short: i18n.TL(i18n.KeyCmdSkillsShort),
		RunE:  runSkillsList,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsListUse),
		Short: i18n.TL(i18n.KeyCmdSkillsListShort),
		RunE:  runSkillsList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsSearchUse),
		Short: i18n.TL(i18n.KeyCmdSkillsSearchShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsSearch,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsInstallUse),
		Short: i18n.TL(i18n.KeyCmdSkillsInstallShort),
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runSkillsInstall,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsNewUse),
		Short: i18n.TL(i18n.KeyCmdSkillsNewShort),
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runSkillsNew,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsDisableUse),
		Short: i18n.TL(i18n.KeyCmdSkillsDisableShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsDisable,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsEnableUse),
		Short: i18n.TL(i18n.KeyCmdSkillsEnableShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsEnable,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   i18n.TL(i18n.KeyCmdSkillsUninstallUse),
		Short: i18n.TL(i18n.KeyCmdSkillsUninstallShort),
		Args:  cobra.ExactArgs(1),
		RunE:  runSkillsUninstall,
	})

	return cmd
}

func loadSkillsCmdConfig() (*config.Config, *skill.Manager, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	// Build skill dirs the same way as initSkillManager in root.go
	skillDirs := []string{cfg.Skills.Dir}
	if homeDir, err := os.UserHomeDir(); err == nil {
		userSkillsDir := filepath.Join(homeDir, config.UserConfigDir, "skills")
		if userSkillsDir != cfg.Skills.Dir {
			skillDirs = append([]string{userSkillsDir}, skillDirs...)
		}
	}
	mgr := skill.NewManager(skillDirs...)
	if err := mgr.Load(); err != nil {
		return nil, nil, fmt.Errorf("load skills: %w", err)
	}
	return cfg, mgr, nil
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadSkillsCmdConfig()
	if err != nil {
		return err
	}

	skills := mgr.List()
	if len(skills) == 0 {
		fmt.Println(i18n.TL(i18n.KeySkillsCLINone))
		return nil
	}

	zap.S().Infow("listed skills", "count", len(skills))

	fmt.Printf("%-30s %s\n", "NAME", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))
	for _, s := range skills {
		desc := s.Description
		if len(desc) > 45 {
			desc = desc[:42] + "..."
		}
		fmt.Printf("%-30s %s\n", s.Name, desc)
	}
	fmt.Printf(i18n.TL(i18n.KeySkillsCLITotal)+"\n", len(skills))
	return nil
}

func runSkillsSearch(cmd *cobra.Command, args []string) error {
	cfg, mgr, err := loadSkillsCmdConfig()
	if err != nil {
		return err
	}

	query := strings.ToLower(args[0])
	installed := mgr.List()

	// Build results, de-duplicating by name
	seen := make(map[string]bool)
	var results []skillResult

	// Local skills first
	for _, s := range installed {
		if strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Description), query) {
			results = append(results, skillResult{
				Name:        s.Name,
				Description: s.Description,
				Source:      "local",
				Installed:   true,
			})
			seen[s.Name] = true
		}
	}

	// Determine repos: prefer configured, fall back to default
	repos := cfg.Skills.Repos
	if len(repos) == 0 {
		repos = []string{"https://raw.githubusercontent.com/dolphinZzv/dolphin/main/skills.json"}
	}

	// Fetch remote repos and search their manifests
	if len(repos) > 0 {
		homeDir, err := os.UserHomeDir()
		cacheDir := ""
		if err == nil {
			cacheDir = filepath.Join(homeDir, config.UserConfigDir, "cache")
		}
		fetcher := config.NewRepoFetcher(cacheDir)
		if ex, err := os.Executable(); err == nil {
			fetcher.SetLocalDir(filepath.Dir(ex))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		var manifests []*config.ToolManifest
		for _, repo := range repos {
			m, err := fetcher.FetchSkillsManifest(ctx, repo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[skills] fetch %s: %v\n", repo, err)
				continue
			}
			manifests = append(manifests, m)
		}
		cancel()

		for _, m := range manifests {
			for _, t := range m.Tools {
				if seen[t.Name] {
					continue
				}
				haystack := strings.ToLower(t.Name + " " + t.Description)
				if strings.Contains(haystack, query) {
					results = append(results, skillResult{
						Name:        t.Name,
						Description: t.Description,
						Source:      m.Name,
						Installed:   false,
					})
					seen[t.Name] = true
				}
			}
		}
	}

	zap.S().Infow("searched skills", "query", args[0], "results", len(results))

	if len(results) == 0 {
		fmt.Printf(i18n.TL(i18n.KeySkillsCLISearchNone)+"\n", args[0])
		return nil
	}

	fmt.Printf("%-30s %-18s %s\n", "NAME", "SOURCE", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))
	for _, r := range results {
		desc := r.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		mark := " "
		if r.Installed {
			mark = "*"
		}
		fmt.Printf("%s%-29s %-18s %s\n", mark, r.Name, r.Source, desc)
	}
	fmt.Printf(i18n.TL(i18n.KeySkillsCLIFound)+"\n", len(results), args[0])
	return nil
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	name := args[0]

	// Use project skills dir as install target
	installDir := cfg.Skills.Dir
	mgr := skill.NewManager(installDir)
	if err := mgr.Load(); err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	// Check if already installed
	if _, ok := mgr.Get(name); ok {
		return fmt.Errorf("skill %q is already installed", name)
	}
	// Check if disabled
	if _, err := os.Stat(filepath.Join(installDir, name+".disabled")); err == nil {
		return fmt.Errorf("skill %q is installed but disabled. Use 'dolphin skills enable %s' to restore it", name, name)
	}

	// Determine repos
	repos := cfg.Skills.Repos
	if len(repos) == 0 {
		repos = []string{"https://raw.githubusercontent.com/dolphinZzv/dolphin/main/skills.json"}
	}

	// Fetch skills manifest from repos
	homeDir, err := os.UserHomeDir()
	cacheDir := ""
	if err == nil {
		cacheDir = filepath.Join(homeDir, config.UserConfigDir, "cache")
	}
	fetcher := config.NewRepoFetcher(cacheDir)
	if ex, err := os.Executable(); err == nil {
		fetcher.SetLocalDir(filepath.Dir(ex))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var found *config.ToolEntry
	for _, repo := range repos {
		m, err := fetcher.FetchSkillsManifest(ctx, repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[skills] fetch %s: %v\n", repo, err)
			continue
		}
		for _, t := range m.Tools {
			if t.Name == name {
				found = &t
				break
			}
		}
		if found != nil {
			break
		}
	}

	if found == nil {
		return fmt.Errorf("skill %q not found in any configured repo", name)
	}

	description := found.Description
	if len(args) > 1 {
		description = args[1]
	}

	// Download skill content from URL, or use description as fallback
	var content string
	if found.URL != "" {
		content, err = downloadSkillContent(found.URL)
		if err != nil {
			return fmt.Errorf("download skill content: %w", err)
		}
	} else {
		content = fmt.Sprintf("# %s\n\n%s\n", name, description)
	}

	if err := mgr.Register(name, description, content); err != nil {
		return fmt.Errorf("install skill: %w", err)
	}

	zap.S().Infow("installed skill", "name", name, "dir", installDir)
	fmt.Printf(i18n.TL(i18n.KeySkillsCLIInstalled)+"\n", name, installDir)
	fmt.Println(i18n.TL(i18n.KeySkillsCLIEdit))
	return nil
}

func downloadSkillContent(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(data), nil
}

func runSkillsNew(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	name := args[0]
	description := name
	if len(args) > 1 {
		description = args[1]
	}

	mgr := skill.NewManager(cfg.Skills.Dir)
	if err := mgr.Load(); err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	if err := mgr.NewTemplate(name, description); err != nil {
		return fmt.Errorf("create skill: %w", err)
	}

	zap.S().Infow("created skill", "name", name, "dir", cfg.Skills.Dir)
	fmt.Printf(i18n.TL(i18n.KeySkillsCLICreated)+"\n", name, cfg.Skills.Dir)
	fmt.Println(i18n.TL(i18n.KeySkillsCLIEdit))
	return nil
}

func runSkillsDisable(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadSkillsCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]

	if err := mgr.Disable(name); err != nil {
		return fmt.Errorf("disable skill: %w", err)
	}

	zap.S().Infow("disabled skill", "name", name)
	fmt.Printf(i18n.TL(i18n.KeySkillsCLIDisabled)+"\n", name)
	return nil
}

func runSkillsEnable(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadSkillsCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]

	if err := mgr.Enable(name); err != nil {
		return fmt.Errorf("enable skill: %w", err)
	}

	zap.S().Infow("enabled skill", "name", name)
	fmt.Printf(i18n.TL(i18n.KeySkillsCLIEnabled)+"\n", name)
	return nil
}

func runSkillsUninstall(cmd *cobra.Command, args []string) error {
	_, mgr, err := loadSkillsCmdConfig()
	if err != nil {
		return err
	}

	name := args[0]

	if _, ok := mgr.Get(name); !ok {
		return fmt.Errorf("skill %q not found", name)
	}

	if err := mgr.Unregister(name); err != nil {
		return fmt.Errorf("uninstall skill: %w", err)
	}

	zap.S().Infow("uninstalled skill", "name", name)
	fmt.Printf(i18n.TL(i18n.KeySkillsCLIUninstalled)+"\n", name)
	return nil
}
