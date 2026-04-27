package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"git-ark/internal/config"
	"git-ark/internal/git"
)

// Run builds a plan and then executes it in one shot.
func (s *Service) Run(ctx context.Context, cfg config.Config, opts RunOptions) (Report, Plan, error) {
	plan, err := s.BuildPlan(ctx, cfg, opts)
	if err != nil {
		return Report{}, Plan{}, err
	}
	return s.RunPlan(ctx, cfg, plan, opts)
}

// RunPlan executes a previously built plan. This is useful for dry runs and tests.
func (s *Service) RunPlan(ctx context.Context, cfg config.Config, plan Plan, opts RunOptions) (Report, Plan, error) {
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	started := nowFn()
	report := Report{
		Repo:      plan.ResolvedRepoRoot,
		Mode:      plan.Mode,
		DryRun:    opts.DryRun,
		StartedAt: started.UTC(),
		Results:   []TargetResult{},
		Errors:    []string{},
	}
	if plan.DryRun || opts.DryRun {
		report.FinishedAt = nowFn().UTC()
		report.DurationMS = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
		report.Success = true
		return report, plan, nil
	}

	if plan.Mode == "mirror" && !opts.Yes && cfg.Options.ConfirmDangerousOperations {
		if !isInteractiveInput() {
			return report, plan, fmt.Errorf("mirror mode requires --yes in non-interactive environments")
		}
		return report, plan, fmt.Errorf("mirror mode requires confirmation")
	}
	if cfg.Options.FetchBeforeBackup {
		var err error
		if cfg.Options.SkipLFS {
			err = s.Git.FetchAllPruneWithEnv(ctx, plan.RepoPath, map[string]string{"GIT_LFS_SKIP_SMUDGE": "1"})
		} else {
			err = s.Git.FetchAllPrune(ctx, plan.RepoPath)
		}
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			if !cfg.Options.ContinueOnError {
				report.FinishedAt = nowFn().UTC()
				report.DurationMS = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
				return report, plan, err
			}
		}
	}
	if cfg.Options.PullBeforeBackup {
		var err error
		if cfg.Options.SkipLFS {
			err = s.Git.PullFFOnlyWithEnv(ctx, plan.RepoPath, map[string]string{"GIT_LFS_SKIP_SMUDGE": "1"})
		} else {
			err = s.Git.PullFFOnly(ctx, plan.RepoPath)
		}
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			if !cfg.Options.ContinueOnError {
				report.FinishedAt = nowFn().UTC()
				report.DurationMS = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
				return report, plan, err
			}
		}
	}
	if !plan.BundleOnly {
		for _, target := range plan.Targets {
			res := TargetResult{Target: target.Name, Type: "remote", URL: git.RedactURL(target.URL), Required: target.Required}
			if err := s.prepareRemote(ctx, cfg, plan, target); err != nil {
				res.Success = false
				res.Error = err.Error()
				report.Results = append(report.Results, res)
				report.Errors = append(report.Errors, err.Error())
				if !cfg.Options.ContinueOnError {
					break
				}
				continue
			}
			err := s.pushTarget(ctx, cfg, plan, target)
			if err != nil {
				res.Success = false
				res.Error = err.Error()
				report.Results = append(report.Results, res)
				report.Errors = append(report.Errors, err.Error())
				if !cfg.Options.ContinueOnError {
					break
				}
				continue
			}
			res.Success = true
			report.Results = append(report.Results, res)
		}
	}
	if plan.BundleEnabled || plan.BundleOnly {
		bundlePath, err := s.createBundle(ctx, cfg, plan)
		if err != nil {
			report.Results = append(report.Results, TargetResult{Target: "bundle", Type: "bundle", URL: bundlePath, Required: true, Success: false, Error: err.Error()})
			report.Errors = append(report.Errors, err.Error())
		} else {
			report.Results = append(report.Results, TargetResult{Target: "bundle", Type: "bundle", URL: bundlePath, Required: true, Success: true})
		}
	}
	report.FinishedAt = nowFn().UTC()
	report.DurationMS = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
	report.Success = successFromResults(report.Results)
	if !cfg.Options.ContinueOnError && len(report.Errors) > 0 {
		report.Success = false
	}

	return report, plan, nil
}

// prepareRemote makes sure the destination remote exists and points at the URL we expect.
func (s *Service) prepareRemote(ctx context.Context, cfg config.Config, plan Plan, target TargetPlan) error {
	if target.URL == "" {
		return fmt.Errorf("remote %q has no URL", target.Name)
	}
	_ = ctx
	_ = cfg
	_ = plan
	return nil
}

// pushTarget performs the actual ref push for one remote.
func (s *Service) pushTarget(ctx context.Context, cfg config.Config, plan Plan, target TargetPlan) error {
	remoteRef := target.Name
	if !cfg.Options.ManageRemotes && !target.ExistsLocal {
		remoteRef = target.URL
	}
	if plan.Mode == "mirror" {
		return git.WrapGitError("mirror push", target.Name, s.Git.PushMirror(ctx, plan.RepoPath, remoteRef), "")
	}
	if plan.Mode == "bundle" {
		return nil
	}
	if cfg.Options.ManageRemotes {
		if err := s.ensureRemoteState(ctx, plan, target); err != nil {
			return err
		}
	}
	if cfg.Options.PushAllRefs {
		refspecs := append([]string{}, git.BranchRefspecs(plan.Branches)...)
		refspecs = append(refspecs, git.TagRefspecs(plan.Tags)...)
		refspecs = append(refspecs, "refs/notes/*")
		if len(refspecs) == 0 {
			return nil
		}
		if cfg.Options.Prune {
			if err := s.Git.PushRefspecsPrune(ctx, plan.RepoPath, remoteRef, refspecs); err != nil {
				return git.WrapGitError("push all refs", target.Name, err, "")
			}
			return nil
		}
		if err := s.Git.PushRefspecs(ctx, plan.RepoPath, remoteRef, refspecs); err != nil {
			return git.WrapGitError("push all refs", target.Name, err, "")
		}
		return nil
	}
	if cfg.Options.PushBranches {
		if len(plan.Branches) == 0 {
			// Nothing to do.
		} else if hasBranchFilters(cfg) {
			if cfg.Options.Prune {
				if err := s.Git.PushRefspecsPrune(ctx, plan.RepoPath, remoteRef, git.BranchRefspecs(plan.Branches)); err != nil {
					return git.WrapGitError("branch push", target.Name, err, "")
				}
			} else if err := s.Git.PushRefspecs(ctx, plan.RepoPath, remoteRef, git.BranchRefspecs(plan.Branches)); err != nil {
				return git.WrapGitError("branch push", target.Name, err, "")
			}
		} else if cfg.Options.Prune {
			if err := s.Git.PushAllPrune(ctx, plan.RepoPath, remoteRef); err != nil {
				return git.WrapGitError("branch push", target.Name, err, "")
			}
		} else if err := s.Git.PushAll(ctx, plan.RepoPath, remoteRef); err != nil {
			return git.WrapGitError("branch push", target.Name, err, "")
		}
	}
	if cfg.Options.PushTags {
		if len(plan.Tags) > 0 {
			if hasTagFilters(cfg) {
				if cfg.Options.Prune {
					if err := s.Git.PushRefspecsPrune(ctx, plan.RepoPath, remoteRef, git.TagRefspecs(plan.Tags)); err != nil {
						return git.WrapGitError("tag push", target.Name, err, "")
					}
				} else if err := s.Git.PushRefspecs(ctx, plan.RepoPath, remoteRef, git.TagRefspecs(plan.Tags)); err != nil {
					return git.WrapGitError("tag push", target.Name, err, "")
				}
			} else if cfg.Options.Prune {
				if err := s.Git.PushTagsPrune(ctx, plan.RepoPath, remoteRef); err != nil {
					return git.WrapGitError("tag push", target.Name, err, "")
				}
			} else if err := s.Git.PushTags(ctx, plan.RepoPath, remoteRef); err != nil {
				return git.WrapGitError("tag push", target.Name, err, "")
			}
		}
	}
	if cfg.Options.PushNotes {
		if err := s.Git.PushNotes(ctx, plan.RepoPath, remoteRef); err != nil {
			return git.WrapGitError("notes push", target.Name, err, "")
		}
	}
	return nil
}

// ensureRemoteState keeps the local remote record aligned with the config.
func (s *Service) ensureRemoteState(ctx context.Context, plan Plan, target TargetPlan) error {
	current, err := s.Git.RemoteURL(ctx, plan.RepoPath, target.Name)
	if err != nil {
		return s.Git.AddRemote(ctx, plan.RepoPath, target.Name, target.URL)
	}
	if current != target.URL {
		return s.Git.SetRemoteURL(ctx, plan.RepoPath, target.Name, target.URL)
	}
	return nil
}

// createBundle writes the optional bundle archive if the plan asked for one.
func (s *Service) createBundle(ctx context.Context, cfg config.Config, plan Plan) (string, error) {
	if !cfg.Bundle.Enabled && !plan.BundleOnly {
		return "", nil
	}
	path := plan.BundlePath
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	if err := s.Git.BundleCreate(ctx, plan.RepoPath, path); err != nil {
		return path, git.WrapGitError("bundle creation", path, err, "")
	}
	return path, nil
}

// isInteractiveInput checks whether stdin looks like a terminal.
func isInteractiveInput() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// RemoteSummary returns a quick view of the remotes without executing a backup.
func (s *Service) RemoteSummary(ctx context.Context, cfg config.Config, repo string) ([]TargetPlan, error) {
	remoteMap, err := s.Git.RemoteMap(ctx, repo)
	if err != nil {
		return nil, err
	}
	targets := make([]TargetPlan, 0, len(cfg.Remotes))
	for name, remote := range cfg.Remotes {
		targets = append(targets, TargetPlan{
			Name:        name,
			URL:         remote.URL,
			CurrentURL:  remoteMap[name],
			Enabled:     remote.Enabled,
			Required:    remote.Required,
			ExistsLocal: remoteMap[name] != "",
		})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })
	return targets, nil
}
