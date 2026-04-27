package cli

import (
	"fmt"
	"io"
	"sort"
	"time"

	"git-ark/internal/backup"
	"git-ark/internal/config"
	"git-ark/internal/git"

	"github.com/spf13/cobra"
)

// newStatusCommand shows the local repo shape and the latest backup metadata.
func newStatusCommand(d deps) *cobra.Command {
	var configPath string
	var repoPath string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show repository status and backup readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cfg, _, warnings, err := loadConfigForCommand(repoPath, configPath)
			if err != nil {
				return err
			}

			if err := printWarnings(cmd, warnings); err != nil {
				return err
			}

			if _, err := config.Validate(cfg); err != nil {
				return err
			}

			repo := repoPath
			if repo == "" {
				repo = cfg.Repo
			}
			if repo == "" {
				repo = "."
			}

			branch, err := d.git.CurrentBranch(cmd.Context(), repo)
			if err != nil {
				return err
			}
			clean, err := d.git.WorktreeClean(cmd.Context(), repo)
			if err != nil {
				return err
			}

			if branch == "" {
				branch = "detached HEAD"
			}

			branches, err := d.git.LocalBranches(cmd.Context(), repo)
			if err != nil {
				return err
			}
			tags, err := d.git.LocalTags(cmd.Context(), repo)
			if err != nil {
				return err
			}

			remoteMap, err := d.git.RemoteMap(cmd.Context(), repo)
			if err != nil {
				return err
			}

			missing := make([]string, 0)
			names := make([]string, 0, len(cfg.Remotes))
			for name, remote := range cfg.Remotes {
				names = append(names, name)
				if remote.Enabled {
					if _, ok := remoteMap[name]; !ok {
						missing = append(missing, name)
					}
				}
			}

			sort.Strings(names)
			sort.Strings(missing)

			if err := writeStatusLine(out, "Current branch:", branch); err != nil {
				return err
			}
			if err := writeStatusLine(out, "Worktree clean:", clean); err != nil {
				return err
			}
			if err := writeStatusLine(out, "Local branches:", len(branches)); err != nil {
				return err
			}
			if err := writeStatusLine(out, "Local tags:", len(tags)); err != nil {
				return err
			}
			if err := writeStatusLine(out, "Configured remotes:", len(cfg.Remotes)); err != nil {
				return err
			}
			for _, name := range names {
				remote := cfg.Remotes[name]
				localURL, exists := remoteMap[name]
				status := "missing"
				if exists {
					status = "present"
				}
				if err := writeStatusf(out, "- %s: %s (%s)\n", name, localURLOrConfigured(localURL, remote.URL), status); err != nil {
					return err
				}
			}
			if err := writeStatusLine(out, "Missing remotes:", missing); err != nil {
				return err
			}
			meta := backup.MetadataPath(cfg, repo)
			store, err := backup.ReadMetadata(meta)
			if err != nil {
				if err := writeStatusLine(out, "Last backup metadata: none"); err != nil {
					return err
				}
				return nil
			}
			latest, ok := store.LatestReport()
			if !ok {
				if err := writeStatusLine(out, "Last backup metadata: none"); err != nil {
					return err
				}
				return nil
			}
			if err := writeStatusLine(out, "Last backup metadata:", meta); err != nil {
				return err
			}

			if store.Repo != "" {
				if err := writeStatusLine(out, "Metadata repo:", store.Repo); err != nil {
					return err
				}
			}

			if err := writeStatusLine(out, "Recorded backups:", len(store.History)); err != nil {
				return err
			}

			if err := writeStatusLine(out, "Latest backup mode:", latest.Mode); err != nil {
				return err
			}
			if err := writeStatusLine(out, "Latest backup success:", latest.Success); err != nil {
				return err
			}

			recent := store.RecentReports(3)
			if len(recent) > 0 {
				if err := writeStatusLine(out, "Recent backups:"); err != nil {
					return err
				}
				for _, report := range recent {
					if err := writeStatusLine(out, "-", formatBackupHistoryEntry(report)); err != nil {
						return err
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

func localURLOrConfigured(localURL, configured string) string {
	if localURL != "" {
		return git.RedactURL(localURL)
	}
	return git.RedactURL(configured)
}

// formatBackupHistoryEntry keeps the history summary compact and readable.
func formatBackupHistoryEntry(report backup.Report) string {
	when := report.FinishedAt
	if when.IsZero() {
		when = report.StartedAt
	}
	timestamp := "unknown time"
	if !when.IsZero() {
		timestamp = when.UTC().Format(time.RFC3339)
	}
	outcome := "failure"
	if report.Success {
		outcome = "success"
	}
	duration := ""
	if report.DurationMS > 0 {
		duration = fmt.Sprintf(" (%s)", time.Duration(report.DurationMS)*time.Millisecond)
	}
	return fmt.Sprintf("%s mode=%s %s%s", timestamp, report.Mode, outcome, duration)
}

// writeStatusLine wraps fmt.Fprintln so write errors get propagated cleanly.
func writeStatusLine(out io.Writer, args ...any) error {
	_, err := fmt.Fprintln(out, args...)
	return err
}

// writeStatusf wraps fmt.Fprintf so write errors get propagated cleanly.
func writeStatusf(out io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(out, format, args...)
	return err
}
