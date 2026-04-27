package cli

import (
	"os/exec"
	"sort"
	"strings"

	"git-ark/internal/config"
	gitpkg "git-ark/internal/git"

	"github.com/spf13/cobra"
)

// newDoctorCommand prints a quick health report for the repo and its remotes.
func newDoctorCommand(d deps) *cobra.Command {
	var configPath string
	var repoPath string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, warnings, err := loadConfigForCommand(repoPath, configPath)
			if err != nil {
				return err
			}

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

			if err := d.git.EnsureInstalled(cmd.Context()); err != nil {
				if err := cmdOutLine(cmd, "Git: missing"); err != nil {
					return err
				}
			} else {
				if err := cmdOutLine(cmd, "Git: available"); err != nil {
					return err
				}
			}
			if _, err := exec.LookPath("ssh"); err != nil {
				if err := cmdOutLine(cmd, "SSH: unavailable"); err != nil {
					return err
				}
			} else {
				if err := cmdOutLine(cmd, "SSH: available"); err != nil {
					return err
				}
			}
			isRepo, err := d.git.IsRepo(cmd.Context(), repo)
			if err != nil {
				return err
			}

			if err := cmdOutLine(cmd, "Repo valid:", isRepo); err != nil {
				return err
			}

			lfsDetected, err := gitpkg.DetectLFSUsage(repo)
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

			branches, _ := d.git.LocalBranches(cmd.Context(), repo)
			tags, _ := d.git.LocalTags(cmd.Context(), repo)
			commits, _ := d.git.HasCommits(cmd.Context(), repo)

			if len(branches) == 0 {
				if err := cmdOutLine(cmd, "Warning: no local branches found"); err != nil {
					return err
				}
			}
			if len(tags) == 0 {
				if err := cmdOutLine(cmd, "Warning: no tags found"); err != nil {
					return err
				}
			}
			if !commits {
				if err := cmdOutLine(cmd, "Warning: repository has no commits"); err != nil {
					return err
				}
			}
			if cfg.Mode == "mirror" {
				if err := cmdOutLine(cmd, "Warning: mirror mode enabled"); err != nil {
					return err
				}
			}
			if cfg.Options.Prune {
				if err := cmdOutLine(cmd, "Warning: prune enabled"); err != nil {
					return err
				}
			}
			if cfg.Options.PushAllRefs {
				if err := cmdOutLine(cmd, "Warning: push_all_refs enabled"); err != nil {
					return err
				}
			}

			names := make([]string, 0, len(cfg.Remotes))
			providerNames := make([]string, 0, len(cfg.Remotes))
			for name, remote := range cfg.Remotes {
				names = append(names, name)

				if !config.ValidRemoteName(name) {
					if err := cmdOutLine(cmd, "Warning: invalid remote name", name); err != nil {
						return err
					}
				}
				if !config.LooksLikeRemoteURL(remote.URL) {
					if err := cmdOutLine(cmd, "Warning: remote URL looks malformed for", name); err != nil {
						return err
					}
				}
				if strings.TrimSpace(remote.Provider) != "" {
					providerNames = append(providerNames, name)
				}
			}

			sort.Strings(names)
			sort.Strings(providerNames)

			if err := cmdOutLine(cmd, "Remote health:"); err != nil {
				return err
			}
			for _, name := range names {
				remote := cfg.Remotes[name]
				health := d.git.ProbeRemote(cmd.Context(), remote.URL)
				redacted := gitpkg.RedactURL(remote.URL)
				if health.Reachable {
					if health.Empty {
						if err := cmdOutf(cmd, "- %s: reachable, no refs advertised (%s)\n", name, redacted); err != nil {
							return err
						}
					} else {
						if err := cmdOutf(cmd, "- %s: reachable, refs=%d (%s)\n", name, health.RefCount, redacted); err != nil {
							return err
						}
					}
					continue
				}
				if health.Hint != "" {
					if err := cmdOutf(cmd, "- %s: unreachable - %s (%s)\n", name, health.Hint, redacted); err != nil {
						return err
					}
				} else {
					if err := cmdOutf(cmd, "- %s: unreachable - %s (%s)\n", name, health.Error, redacted); err != nil {
						return err
					}
				}
				if strings.HasPrefix(strings.ToLower(remote.URL), "https://") {
					if err := cmdOutLine(cmd, "Warning: HTTPS remote may require credentials for", name); err != nil {
						return err
					}
				}
			}

			if len(providerNames) > 0 {
				if err := cmdOutLine(cmd, "Provider diagnostics:"); err != nil {
					return err
				}
				for _, name := range providerNames {
					diag := config.DiagnoseProvider(cfg.Remotes[name])
					for _, line := range diag.DetailLines(name) {
						if err := cmdOutLine(cmd, line); err != nil {
							return err
						}
					}
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "config path")
	cmd.Flags().StringVar(&repoPath, "repo", "", "repo path override")
	return cmd
}
