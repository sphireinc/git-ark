package cli

import (
	"git-ark/internal/config"

	"github.com/spf13/cobra"
)

func newInitCommand(d deps) *cobra.Command {
	var outputPath string
	var force bool
	var repoPath string
	var mode string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a starter git-ark.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if repoPath == "" {
				repoPath = "."
			}
			isRepo, err := d.git.IsRepo(ctx, repoPath)
			if err != nil {
				return err
			}
			content := []byte(config.ExampleYAML(repoPath, mode, isRepo))
			if err := writeFileExclusive(outputPath, content, force); err != nil {
				return err
			}
			if err := cmdOutLine(cmd, "Wrote", outputPath); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outputPath, "path", "git-ark.yml", "config output path")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config")
	cmd.Flags().StringVar(&repoPath, "repo", ".", "repo path to write into config")
	cmd.Flags().StringVar(&mode, "mode", "safe", "starter mode")
	return cmd
}
