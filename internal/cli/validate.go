package cli

import (
	"fmt"
	"os"

	"git-ark/internal/config"
	"git-ark/internal/git"

	"github.com/spf13/cobra"
)

func newValidateCommand(d deps) *cobra.Command {
	var configPath string
	var repoPath string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config and local environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, warnings, err := loadConfigForCommand(repoPath, configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			_ = path
			if err := printWarnings(cmd, warnings); err != nil {
				return err
			}
			cfgWarnings, err := config.Validate(cfg)
			if err := printWarnings(cmd, cfgWarnings); err != nil {
				return err
			}
			if err != nil {
				return err
			}
			repo := repoPath
			if repo == "" {
				repo = cfg.Repo
			}
			if repo == "" {
				repo = "."
			}
			if _, err := os.Stat(repo); err != nil {
				return fmt.Errorf("repo path %q: %w", repo, err)
			}
			if err := d.git.EnsureInstalled(cmd.Context()); err != nil {
				return err
			}
			isRepo, err := d.git.IsRepo(cmd.Context(), repo)
			if err != nil {
				return err
			}
			if !isRepo {
				return fmt.Errorf("%s is not a git repository", repo)
			}
			lfsDetected, err := git.DetectLFSUsage(repo)
			if err != nil {
				return err
			}
			if lfsDetected {
				if cfg.Options.SkipLFS {
					if err := cmdOutLine(cmd, "LFS: detected, skip_lfs is enabled"); err != nil {
						return err
					}
				} else {
					if err := cmdOutLine(cmd, "Warning: repository appears to use Git LFS; consider enabling skip_lfs"); err != nil {
						return err
					}
				}
			}
			if cfg.Options.VerifyCleanWorktree {
				clean, err := d.git.WorktreeClean(cmd.Context(), repo)
				if err != nil {
					return err
				}
				if !clean {
					return fmt.Errorf("worktree is not clean")
				}
			}
			for name, remote := range cfg.Remotes {
				if err := config.ValidateRemoteURL(remote.URL); err != nil {
					return fmt.Errorf("remote %q URL invalid: %w", name, err)
				}
				if !config.LooksLikeRemoteURL(remote.URL) {
					return fmt.Errorf("remote %q URL does not look valid: %q", name, remote.URL)
				}
			}
			if err := cmdOutLine(cmd, "config valid"); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "config path")
	cmd.Flags().StringVar(&repoPath, "repo", "", "repo path override")
	return cmd
}
