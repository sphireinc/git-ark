package cli

import (
	"encoding/json"
	"sort"

	"git-ark/internal/config"
	"git-ark/internal/git"

	"github.com/spf13/cobra"
)

func newRemotesCommand(d deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remotes",
		Short: "Inspect configured remotes",
	}
	cmd.AddCommand(newRemotesListCommand(d))
	cmd.AddCommand(newRemotesSyncCommand(d))
	return cmd
}

func newRemotesListCommand(d deps) *cobra.Command {
	var configPath string
	var repoPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured remotes and whether they exist locally",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			local, err := d.git.RemoteMap(cmd.Context(), repo)
			if err != nil {
				return err
			}
			type row struct {
				Name   string `json:"name"`
				URL    string `json:"url"`
				Exists bool   `json:"exists"`
			}
			rows := make([]row, 0, len(cfg.Remotes))
			names := make([]string, 0, len(cfg.Remotes))
			for name := range cfg.Remotes {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				remote := cfg.Remotes[name]
				_, exists := local[name]
				rows = append(rows, row{Name: name, URL: git.RedactURL(remote.URL), Exists: exists})
			}
			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}
			for _, row := range rows {
				if err := cmdOutf(cmd, "%s\t%s\t%t\n", row.Name, row.URL, row.Exists); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "config path")
	cmd.Flags().StringVar(&repoPath, "repo", "", "repo path override")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func newRemotesSyncCommand(d deps) *cobra.Command {
	var configPath string
	var repoPath string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize local Git remotes with config",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			remoteMap, err := d.git.RemoteMap(cmd.Context(), repo)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Remotes))
			for name := range cfg.Remotes {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				remote := cfg.Remotes[name]
				if current, ok := remoteMap[name]; !ok {
					if dryRun {
						if err := cmdOutf(cmd, "would add remote %s %s\n", name, git.RedactURL(remote.URL)); err != nil {
							return err
						}
						continue
					}
					if err := d.git.AddRemote(cmd.Context(), repo, name, remote.URL); err != nil {
						return err
					}
					if err := cmdOutf(cmd, "added remote %s\n", name); err != nil {
						return err
					}
				} else if current != remote.URL {
					if dryRun {
						if err := cmdOutf(cmd, "would set remote %s %s\n", name, git.RedactURL(remote.URL)); err != nil {
							return err
						}
						continue
					}
					if err := d.git.SetRemoteURL(cmd.Context(), repo, name, remote.URL); err != nil {
						return err
					}
					if err := cmdOutf(cmd, "updated remote %s\n", name); err != nil {
						return err
					}
				} else {
					if err := cmdOutf(cmd, "remote %s already up to date\n", name); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "config path")
	cmd.Flags().StringVar(&repoPath, "repo", "", "repo path override")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show changes without mutating")
	return cmd
}
