package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"git-ark/internal/backup"
	"git-ark/internal/config"
	"git-ark/internal/git"

	"github.com/spf13/cobra"
)

func newBackupCommand(d deps) *cobra.Command {
	var configPath string
	var repoPath string
	var dryRun bool
	var mode string
	var selectedRemotes []string
	var includeDisabled bool
	var yes bool
	var jsonOut bool
	var verbose bool
	var quiet bool
	var prune bool
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Run a backup to configured remotes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if verbose && quiet {
				return fmt.Errorf("--verbose and --quiet cannot be used together")
			}
			cfg, _, warnings, err := loadConfigForCommand(repoPath, configPath)
			if err != nil {
				return err
			}
			if err := printWarnings(cmd, warnings); err != nil {
				return err
			}
			if cmd.Flags().Changed("prune") {
				cfg.Options.Prune = prune
			}
			cfgWarnings, err := config.Validate(cfg)
			if err := printWarnings(cmd, cfgWarnings); err != nil {
				return err
			}
			if err != nil {
				return err
			}
			service := backup.New(d.git)
			opts := backup.RunOptions{
				RepoPath:        repoPath,
				ModeOverride:    mode,
				SelectedRemotes: selectedRemotes,
				IncludeDisabled: includeDisabled,
				DryRun:          dryRun,
				Yes:             yes,
			}
			plan, err := service.BuildPlan(cmd.Context(), cfg, opts)
			if err != nil {
				return err
			}
			if !jsonOut && !quiet {
				if err := printPlan(cmd, plan); err != nil {
					return err
				}
				if verbose {
					if err := printVerbosePlan(cmd, plan); err != nil {
						return err
					}
				}
			}
			if !dryRun && plan.Mode == "mirror" && cfg.Options.ConfirmDangerousOperations && !yes {
				if !isInteractiveInput() {
					return fmt.Errorf("mirror mode requires --yes in non-interactive environments")
				}
				if !confirmMirror(cmd) {
					return fmt.Errorf("mirror mode confirmation declined")
				}
			}
			if dryRun {
				report, _, err := service.RunPlan(cmd.Context(), cfg, plan, opts)
				if err != nil {
					return err
				}
				if jsonOut {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					if err := enc.Encode(report); err != nil {
						return err
					}
					return nil
				}
				if !quiet {
					if err := cmdOutLine(cmd, "Dry run complete"); err != nil {
						return err
					}
				}
				return nil
			}
			report, plan2, err := service.RunPlan(cmd.Context(), cfg, plan, opts)
			if err != nil {
				return err
			}
			_ = plan2
			if err := service.WriteMetadata(cfg, report, plan.RepoPath); err != nil {
				if warnErr := cmdErrLine(cmd, "WARN:", err); warnErr != nil {
					return warnErr
				}
			}
			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(report); err != nil {
					return err
				}
				if !report.Success {
					return fmt.Errorf("backup completed with failures")
				}
				return nil
			}
			if !quiet {
				if err := printSummary(cmd, report); err != nil {
					return err
				}
			}
			if !quiet && report.Success && hasOptionalFailures(report) {
				if err := cmdErrLine(cmd, "WARN: some optional remotes failed"); err != nil {
					return err
				}
			}
			if !report.Success {
				return fmt.Errorf("backup completed with failures")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "config path")
	cmd.Flags().StringVar(&repoPath, "repo", "", "repo path override")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would happen without mutating")
	cmd.Flags().StringVar(&mode, "mode", "", "override mode")
	cmd.Flags().StringSliceVar(&selectedRemotes, "remote", nil, "backup only selected remotes")
	cmd.Flags().BoolVar(&includeDisabled, "include-disabled", false, "allow selected disabled remotes")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip dangerous operation confirmation")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON summary")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "quiet output")
	cmd.Flags().BoolVar(&prune, "prune", false, "prune remote refs not present locally")
	return cmd
}

func confirmMirror(cmd *cobra.Command) bool {
	if err := cmdErrLine(cmd, "Mirror mode can delete refs on backup remotes."); err != nil {
		return false
	}
	if err := cmdErrf(cmd, "Type \"mirror\" to continue: "); err != nil {
		return false
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.TrimSpace(line) == "mirror"
}

func isInteractiveInput() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func printPlan(cmd *cobra.Command, plan backup.Plan) error {
	if err := cmdOutLine(cmd, "Backup plan:"); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Repo:", plan.RepoPath); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Root:", plan.ResolvedRepoRoot); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Mode:", plan.Mode); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Selected remotes:", strings.Join(plan.SelectedRemotes, ", ")); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Required remotes:", strings.Join(plan.RequiredRemotes, ", ")); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Branches:", len(plan.Branches)); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Tags:", len(plan.Tags)); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Branch push:", plan.BranchPushMode); err != nil {
		return err
	}
	if err := cmdOutLine(cmd, "Tag push:", plan.TagPushMode); err != nil {
		return err
	}
	if plan.PushNotes {
		if err := cmdOutLine(cmd, "Notes: enabled"); err != nil {
			return err
		}
	} else {
		if err := cmdOutLine(cmd, "Notes: disabled"); err != nil {
			return err
		}
	}
	if plan.BundleEnabled || plan.BundleOnly {
		if err := cmdOutLine(cmd, "Bundle: enabled"); err != nil {
			return err
		}
		if err := cmdOutLine(cmd, "Bundle path:", plan.BundlePath); err != nil {
			return err
		}
	} else {
		if err := cmdOutLine(cmd, "Bundle: disabled"); err != nil {
			return err
		}
	}
	if plan.RemoteManagement {
		if err := cmdOutLine(cmd, "Remote management: enabled"); err != nil {
			return err
		}
	} else {
		if err := cmdOutLine(cmd, "Remote management: disabled"); err != nil {
			return err
		}
	}
	if plan.LFSNote != "" {
		if err := cmdOutLine(cmd, "LFS:", plan.LFSNote); err != nil {
			return err
		}
	}
	if len(plan.DangerousOps) > 0 {
		if err := cmdOutLine(cmd, "Dangerous operations:"); err != nil {
			return err
		}
		for _, op := range plan.DangerousOps {
			if err := cmdOutLine(cmd, "-", op); err != nil {
				return err
			}
		}
	} else {
		if err := cmdOutLine(cmd, "Dangerous operations: none"); err != nil {
			return err
		}
	}
	if err := cmdOutLine(cmd, "Remotes:"); err != nil {
		return err
	}
	for _, target := range plan.Targets {
		if err := cmdOutf(cmd, "- %s: %s\n", target.Name, git.RedactURL(target.URL)); err != nil {
			return err
		}
		if err := cmdOutf(cmd, "  action: %s\n", target.SyncAction(plan.RemoteManagement)); err != nil {
			return err
		}
	}
	return cmdOutLine(cmd, "Dry run:", plan.DryRun)
}

func printVerbosePlan(cmd *cobra.Command, plan backup.Plan) error {
	for _, target := range plan.Targets {
		if err := cmdOutf(cmd, "target %s current=%s desired=%s\n", target.Name, git.RedactURL(target.CurrentURL), git.RedactURL(target.URL)); err != nil {
			return err
		}
	}
	return nil
}

func printSummary(cmd *cobra.Command, report backup.Report) error {
	if err := cmdOutLine(cmd, "Backup summary:"); err != nil {
		return err
	}
	for _, result := range report.Results {
		if result.Success {
			if result.Type == "bundle" {
				if err := cmdOutf(cmd, "bundle: success - %s\n", result.URL); err != nil {
					return err
				}
			} else {
				if err := cmdOutf(cmd, "%s: success\n", result.Target); err != nil {
					return err
				}
			}
			continue
		}
		if err := cmdOutf(cmd, "%s: failed - %s\n", result.Target, result.Error); err != nil {
			return err
		}
	}
	return nil
}

func hasOptionalFailures(report backup.Report) bool {
	for _, result := range report.Results {
		if !result.Required && !result.Success {
			return true
		}
	}
	return false
}
