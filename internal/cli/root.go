package cli

import (
	"context"
	"fmt"
	"os"

	"git-ark/internal/backup"
	"git-ark/internal/config"
	gitpkg "git-ark/internal/git"
	"git-ark/internal/log"
	"git-ark/internal/platform"

	"github.com/spf13/cobra"
)

type Root struct {
	*cobra.Command
}

type deps struct {
	git    *gitpkg.Client
	backup *backup.Service
	logger *log.Logger
}

func NewRoot() *Root {
	d := newDeps()
	root := &cobra.Command{
		Use:           "git-ark",
		Short:         "Back up a local Git repository to multiple remotes",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	addCommands(root, d)
	return &Root{Command: root}
}

func (r *Root) Execute() error {
	if r == nil || r.Command == nil {
		return fmt.Errorf("root command not initialized")
	}
	return r.Command.ExecuteContext(context.Background())
}

func newDeps() deps {
	gg := gitpkg.NewClient(nil)
	return deps{
		git:    gg,
		backup: backup.New(gg),
		logger: log.New(os.Stdout, os.Stderr, "info", "text"),
	}
}

func addCommands(root *cobra.Command, d deps) {
	root.AddCommand(newInitCommand(d))
	root.AddCommand(newValidateCommand(d))
	root.AddCommand(newBackupCommand(d))
	root.AddCommand(newRemotesCommand(d))
	root.AddCommand(newStatusCommand(d))
	root.AddCommand(newDoctorCommand(d))
	root.AddCommand(newVersionCommand())
}

func loadConfigForCommand(repoOverride, configPath string) (config.Config, string, []string, error) {
	_ = repoOverride
	return config.LoadSearch(".", configPath)
}

func ensureRepoPath(repoOverride string, cfg config.Config) (string, error) {
	repo := repoOverride
	if repo == "" {
		repo = cfg.Repo
	}
	if repo == "" {
		repo = "."
	}
	return repo, nil
}

// printWarnings keeps config warnings on stderr and fails if the write does.
func printWarnings(cmd *cobra.Command, warnings []string) error {
	for _, warning := range warnings {
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "WARN:", warning); err != nil {
			return err
		}
	}
	return nil
}

func newContextCommand(cmd *cobra.Command) context.Context {
	return cmd.Context()
}

func redactURL(raw string) string {
	return gitpkg.RedactURL(raw)
}

func versionInfo() string {
	info := platform.Current()
	return fmt.Sprintf("git-ark %s (%s, %s) %s/%s built %s", info.Version, info.Commit, info.GoVersion, info.GOOS, info.GOARCH, info.BuildDate)
}
