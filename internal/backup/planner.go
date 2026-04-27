package backup

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"git-ark/internal/config"
	"git-ark/internal/git"
)

// BuildPlan pulls together the repo state, config, and CLI overrides into a
// single backup plan. The rest of the pipeline just follows this plan.
func (s *Service) BuildPlan(ctx context.Context, cfg config.Config, opts RunOptions) (Plan, error) {
	if s == nil || s.Git == nil {
		return Plan{}, fmt.Errorf("git client is not configured")
	}

	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	repoPath := opts.RepoPath
	if repoPath == "" {
		repoPath = cfg.Repo
	}
	if repoPath == "" {
		repoPath = "."
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return Plan{}, err
	}

	isRepo, err := s.Git.IsRepo(ctx, absRepo)
	if err != nil {
		return Plan{}, err
	}
	if !isRepo {
		return Plan{}, fmt.Errorf("%s is not a git repository", absRepo)
	}

	root, err := s.Git.RepoRoot(ctx, absRepo)
	if err != nil {
		return Plan{}, err
	}

	mode := config.NormalizeMode(cfg.Mode)
	if opts.ModeOverride != "" {
		mode = config.NormalizeMode(opts.ModeOverride)
	}
	if !config.IsValidMode(mode) {
		return Plan{}, fmt.Errorf("invalid mode %q", mode)
	}

	branches, err := s.Git.LocalBranches(ctx, absRepo)
	if err != nil {
		return Plan{}, err
	}
	branches = git.FilterRefs(branches, cfg.BranchFilters.Include, cfg.BranchFilters.Exclude)

	tags, err := s.Git.LocalTags(ctx, absRepo)
	if err != nil {
		return Plan{}, err
	}
	tags = git.FilterRefs(tags, cfg.TagFilters.Include, cfg.TagFilters.Exclude)

	remoteMap, err := s.Git.RemoteMap(ctx, absRepo)
	if err != nil {
		return Plan{}, err
	}

	lfsDetected, err := git.DetectLFSUsage(absRepo)
	if err != nil {
		return Plan{}, err
	}

	desiredNames, requiredNames, targets, err := selectTargets(cfg, remoteMap, opts.SelectedRemotes, opts.IncludeDisabled)
	if err != nil {
		return Plan{}, err
	}

	// These are the operations that might surprise someone skimming the output.
	dangerous := []string{}
	if mode == "mirror" {
		dangerous = append(dangerous, "mirror push may delete remote refs")
	}
	if cfg.Options.Prune {
		dangerous = append(dangerous, "prune may delete remote refs")
	}

	lfsNote := ""
	if lfsDetected {
		if cfg.Options.SkipLFS {
			lfsNote = "Git LFS detected; fetch and pull will skip smudge downloads."
		} else {
			lfsNote = "Git LFS detected; consider enabling skip_lfs if you do not want smudge downloads during fetch or pull."
		}
	}

	bundleEnabled := cfg.Bundle.Enabled
	bundleOnly := mode == "bundle"
	branchPushMode := "disabled"
	if cfg.Options.PushAllRefs {
		branchPushMode = "all-refs"
	} else if cfg.Options.PushBranches {
		if hasBranchFilters(cfg) {
			branchPushMode = "refspecs"
		} else {
			branchPushMode = "all"
		}
	}

	if cfg.Options.Prune && branchPushMode != "disabled" {
		branchPushMode += "+prune"
	}

	tagPushMode := "disabled"
	if cfg.Options.PushAllRefs {
		tagPushMode = "all-refs"
	} else if cfg.Options.PushTags {
		if hasTagFilters(cfg) {
			tagPushMode = "refspecs"
		} else {
			tagPushMode = "all"
		}
	}

	if cfg.Options.Prune && tagPushMode != "disabled" {
		tagPushMode += "+prune"
	}

	return Plan{
		RepoPath:         absRepo,
		ResolvedRepoRoot: root,
		Mode:             mode,
		Targets:          targets,
		SelectedRemotes:  desiredNames,
		RequiredRemotes:  requiredNames,
		Branches:         branches,
		Tags:             tags,
		BranchPushMode:   branchPushMode,
		TagPushMode:      tagPushMode,
		PushAllRefs:      cfg.Options.PushAllRefs,
		PushNotes:        cfg.Options.PushNotes || cfg.Options.PushAllRefs,
		BundleEnabled:    bundleEnabled,
		BundleOnly:       bundleOnly,
		BundlePath:       resolveBundlePath(cfg, absRepo, nowFn().UTC()),
		RemoteManagement: cfg.Options.ManageRemotes,
		LFSNote:          lfsNote,
		LFSDetected:      lfsDetected,
		SkipLFS:          cfg.Options.SkipLFS,
		DangerousOps:     dangerous,
		DryRun:           opts.DryRun,
		WillFetch:        cfg.Options.FetchBeforeBackup,
		WillPull:         cfg.Options.PullBeforeBackup,
	}, nil
}

// selectTargets figures out which remotes the user actually wants to touch.
// It also keeps the result sorted so the CLI output stays stable and easy to scan.
func selectTargets(cfg config.Config, remoteMap map[string]string, selected []string, includeDisabled bool) ([]string, []string, []TargetPlan, error) {
	desired := map[string]struct{}{}
	if len(selected) > 0 {
		for _, name := range selected {
			desired[name] = struct{}{}
		}
	} else {
		for name, remote := range cfg.Remotes {
			if remote.Enabled {
				desired[name] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(desired))
	required := make([]string, 0, len(desired))
	targets := make([]TargetPlan, 0, len(desired))

	for name := range desired {
		remote, ok := cfg.Remotes[name]
		if !ok {
			return nil, nil, nil, fmt.Errorf("unknown remote %q", name)
		}
		if !remote.Enabled && !includeDisabled {
			return nil, nil, nil, fmt.Errorf("remote %q is disabled; pass --include-disabled to use it", name)
		}
		names = append(names, name)
		if remote.Required {
			required = append(required, name)
		}

		targets = append(targets, TargetPlan{
			Name:        name,
			URL:         remote.URL,
			CurrentURL:  remoteMap[name],
			Enabled:     remote.Enabled,
			Required:    remote.Required,
			Selected:    true,
			ExistsLocal: remoteMap[name] != "",
			UseURL:      remoteMap[name] == "",
			WillMutate:  remote.Enabled && remoteMap[name] != "",
			WillPush:    true,
		})
	}

	sort.Strings(names)
	sort.Strings(required)
	sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })

	for i := range targets {
		if targets[i].URL == "" {
			return nil, nil, nil, fmt.Errorf("remote %q has empty URL", targets[i].Name)
		}
		if _, ok := remoteMap[targets[i].Name]; !ok {
			targets[i].UseURL = true
		} else if targets[i].CurrentURL != targets[i].URL {
			targets[i].WillMutate = true
		}
	}

	return names, required, targets, nil
}

// hasBranchFilters keeps the planner readable by hiding the include/exclude check on branches
func hasBranchFilters(cfg config.Config) bool {
	return len(cfg.BranchFilters.Include) > 0 || len(cfg.BranchFilters.Exclude) > 0
}

// hasTagFilters keeps the planner readable by hiding the include/exclude check on tags
func hasTagFilters(cfg config.Config) bool {
	return len(cfg.TagFilters.Include) > 0 || len(cfg.TagFilters.Exclude) > 0
}
