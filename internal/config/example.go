package config

import (
	"fmt"
	"strings"
)

func ExampleYAML(repoPath string, mode string, isGitRepo bool) string {
	if mode == "" {
		mode = "safe"
	}
	if repoPath == "" {
		repoPath = "."
	}
	repoLine := ""
	if isGitRepo || repoPath != "." {
		repoLine = fmt.Sprintf("repo: %q\n", repoPath)
	}
	var b strings.Builder
	b.WriteString("# git-ark v1 config\n")
	b.WriteString("# Use safe mode by default.\n")
	b.WriteString("version: 1\n\n")
	b.WriteString(repoLine)
	if repoLine != "" {
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("mode: %q\n\n", mode))
	b.WriteString("# Backup remotes to push to.\n")
	b.WriteString("remotes:\n")
	b.WriteString("  github:\n")
	b.WriteString("    url: \"git@github.com:example/example-backup.git\"\n")
	b.WriteString("    enabled: true\n")
	b.WriteString("    required: true\n")
	b.WriteString("    provider: \"github\"\n")
	b.WriteString("    description: \"GitHub backup mirror\"\n\n")
	b.WriteString("# Safe defaults keep git-ark conservative.\n")
	b.WriteString("options:\n")
	b.WriteString("  manage_remotes: true\n")
	b.WriteString("  push_branches: true\n")
	b.WriteString("  push_tags: true\n")
	b.WriteString("  push_notes: false\n")
	b.WriteString("  push_all_refs: false\n")
	b.WriteString("  prune: false\n")
	b.WriteString("  verify_clean_worktree: false\n")
	b.WriteString("  continue_on_error: true\n")
	b.WriteString("  confirm_dangerous_operations: true\n")
	b.WriteString("  fetch_before_backup: false\n")
	b.WriteString("  pull_before_backup: false\n")
	b.WriteString("  skip_lfs: false\n")
	b.WriteString("  include_archived_branches: true\n")
	b.WriteString("  write_metadata: true\n\n")
	b.WriteString("# Optional filters narrow the refs that are pushed.\n")
	b.WriteString("branch_filters:\n")
	b.WriteString("  include: []\n")
	b.WriteString("  exclude: []\n\n")
	b.WriteString("tag_filters:\n")
	b.WriteString("  include: []\n")
	b.WriteString("  exclude: []\n\n")
	b.WriteString("# Local bundle archive settings.\n")
	b.WriteString("bundle:\n")
	b.WriteString("  enabled: false\n")
	b.WriteString("  path: \"./backups\"\n")
	b.WriteString("  filename_template: \"{{repo}}-{{timestamp}}.bundle\"\n")
	b.WriteString("  include_all_refs: true\n\n")
	b.WriteString("# Output formatting.\n")
	b.WriteString("logging:\n")
	b.WriteString("  level: \"info\"\n")
	b.WriteString("  format: \"text\"\n\n")
	b.WriteString("# Metadata written after successful backups.\n")
	b.WriteString("metadata:\n")
	b.WriteString("  path: \".git/git-ark-last-backup.json\"\n")
	return b.String()
}
