package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const CurrentVersion = 1

type Config struct {
	Version       int                     `yaml:"version"`
	Repo          string                  `yaml:"repo"`
	Mode          string                  `yaml:"mode"`
	Remotes       map[string]RemoteConfig `yaml:"remotes"`
	Options       Options                 `yaml:"options"`
	BranchFilters FilterSet               `yaml:"branch_filters"`
	TagFilters    FilterSet               `yaml:"tag_filters"`
	Bundle        BundleConfig            `yaml:"bundle"`
	Logging       LoggingConfig           `yaml:"logging"`
	Metadata      MetadataConfig          `yaml:"metadata"`
}

type RemoteConfig struct {
	URL         string `yaml:"url"`
	Enabled     bool   `yaml:"enabled"`
	Required    bool   `yaml:"required"`
	Provider    string `yaml:"provider"`
	Description string `yaml:"description"`
}

type Options struct {
	ManageRemotes              bool `yaml:"manage_remotes"`
	PushBranches               bool `yaml:"push_branches"`
	PushTags                   bool `yaml:"push_tags"`
	PushNotes                  bool `yaml:"push_notes"`
	PushAllRefs                bool `yaml:"push_all_refs"`
	Prune                      bool `yaml:"prune"`
	VerifyCleanWorktree        bool `yaml:"verify_clean_worktree"`
	ContinueOnError            bool `yaml:"continue_on_error"`
	ConfirmDangerousOperations bool `yaml:"confirm_dangerous_operations"`
	FetchBeforeBackup          bool `yaml:"fetch_before_backup"`
	PullBeforeBackup           bool `yaml:"pull_before_backup"`
	SkipLFS                    bool `yaml:"skip_lfs"`
	IncludeArchivedBranches    bool `yaml:"include_archived_branches"`
	WriteMetadata              bool `yaml:"write_metadata"`
}

type FilterSet struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type BundleConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Path             string `yaml:"path"`
	FilenameTemplate string `yaml:"filename_template"`
	IncludeAllRefs   bool   `yaml:"include_all_refs"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type MetadataConfig struct {
	Path string `yaml:"path"`
}

type rawConfig struct {
	Version       *int                 `yaml:"version"`
	Repo          *string              `yaml:"repo"`
	Mode          *string              `yaml:"mode"`
	Remotes       map[string]rawRemote `yaml:"remotes"`
	Options       *rawOptions          `yaml:"options"`
	BranchFilters *FilterSet           `yaml:"branch_filters"`
	TagFilters    *FilterSet           `yaml:"tag_filters"`
	Bundle        *rawBundle           `yaml:"bundle"`
	Logging       *LoggingConfig       `yaml:"logging"`
	Metadata      *MetadataConfig      `yaml:"metadata"`
}

type rawRemote struct {
	URL         *string `yaml:"url"`
	Enabled     *bool   `yaml:"enabled"`
	Required    *bool   `yaml:"required"`
	Provider    *string `yaml:"provider"`
	Description *string `yaml:"description"`
}

type rawOptions struct {
	ManageRemotes              *bool `yaml:"manage_remotes"`
	PushBranches               *bool `yaml:"push_branches"`
	PushTags                   *bool `yaml:"push_tags"`
	PushNotes                  *bool `yaml:"push_notes"`
	PushAllRefs                *bool `yaml:"push_all_refs"`
	Prune                      *bool `yaml:"prune"`
	VerifyCleanWorktree        *bool `yaml:"verify_clean_worktree"`
	ContinueOnError            *bool `yaml:"continue_on_error"`
	ConfirmDangerousOperations *bool `yaml:"confirm_dangerous_operations"`
	FetchBeforeBackup          *bool `yaml:"fetch_before_backup"`
	PullBeforeBackup           *bool `yaml:"pull_before_backup"`
	SkipLFS                    *bool `yaml:"skip_lfs"`
	IncludeArchivedBranches    *bool `yaml:"include_archived_branches"`
	WriteMetadata              *bool `yaml:"write_metadata"`
}

type rawBundle struct {
	Enabled          *bool   `yaml:"enabled"`
	Path             *string `yaml:"path"`
	FilenameTemplate *string `yaml:"filename_template"`
	IncludeAllRefs   *bool   `yaml:"include_all_refs"`
}

func Default() Config {
	cfg := Config{
		Version: CurrentVersion,
		Repo:    ".",
		Mode:    "safe",
		Remotes: map[string]RemoteConfig{},
		Options: Options{
			ManageRemotes:              true,
			PushBranches:               true,
			PushTags:                   true,
			PushNotes:                  false,
			PushAllRefs:                false,
			Prune:                      false,
			VerifyCleanWorktree:        false,
			ContinueOnError:            true,
			ConfirmDangerousOperations: true,
			FetchBeforeBackup:          false,
			PullBeforeBackup:           false,
			SkipLFS:                    false,
			IncludeArchivedBranches:    true,
			WriteMetadata:              true,
		},
		BranchFilters: FilterSet{Include: []string{}, Exclude: []string{}},
		TagFilters:    FilterSet{Include: []string{}, Exclude: []string{}},
		Bundle: BundleConfig{
			Enabled:          false,
			Path:             "./backups",
			FilenameTemplate: "{{repo}}-{{timestamp}}.bundle",
			IncludeAllRefs:   true,
		},
		Logging:  LoggingConfig{Level: "info", Format: "text"},
		Metadata: MetadataConfig{Path: ".git/git-ark-last-backup.json"},
	}
	return cfg
}

func ApplyDefaults(raw []byte) (Config, []string, error) {
	cfg := Default()
	warnings := unknownTopLevelWarnings(raw)
	var rc rawConfig
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(false)
	if err := dec.Decode(&rc); err != nil {
		return Config{}, warnings, err
	}
	if rc.Version != nil {
		cfg.Version = *rc.Version
	}
	if rc.Repo != nil && *rc.Repo != "" {
		cfg.Repo = *rc.Repo
	}
	if rc.Mode != nil && *rc.Mode != "" {
		cfg.Mode = *rc.Mode
	}
	if rc.Options != nil {
		applyRawOptions(&cfg.Options, rc.Options)
	}
	if rc.BranchFilters != nil {
		cfg.BranchFilters = *rc.BranchFilters
	}
	if rc.TagFilters != nil {
		cfg.TagFilters = *rc.TagFilters
	}
	if rc.Bundle != nil {
		applyRawBundle(&cfg.Bundle, rc.Bundle)
	}
	if rc.Logging != nil {
		if rc.Logging.Level != "" {
			cfg.Logging.Level = rc.Logging.Level
		}
		if rc.Logging.Format != "" {
			cfg.Logging.Format = rc.Logging.Format
		}
	}
	if rc.Metadata != nil && rc.Metadata.Path != "" {
		cfg.Metadata.Path = rc.Metadata.Path
	}
	for name, remote := range rc.Remotes {
		cfg.Remotes[name] = resolveRemote(remote)
	}
	return cfg, warnings, nil
}

func Load(path string) (Config, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, nil, err
	}
	return ApplyDefaults(raw)
}

func LoadSearch(repoDir string, explicitPath string) (Config, string, []string, error) {
	candidates := []string{}
	if explicitPath != "" {
		candidates = append(candidates, explicitPath)
	} else {
		candidates = append(candidates,
			filepath.Join(repoDir, "git-ark.yml"),
			filepath.Join(repoDir, "git-ark.yaml"),
			filepath.Join(repoDir, ".git-ark.yml"),
			filepath.Join(repoDir, ".git-ark.yaml"),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			cfg, warnings, err := Load(candidate)
			return cfg, candidate, warnings, err
		}
	}
	if explicitPath != "" {
		return Config{}, explicitPath, nil, os.ErrNotExist
	}
	return Config{}, "", nil, errors.New("no git-ark config found")
}

func resolveRemote(raw rawRemote) RemoteConfig {
	remote := RemoteConfig{
		Enabled:  true,
		Required: true,
	}
	if raw.URL != nil {
		remote.URL = *raw.URL
	}
	if raw.Enabled != nil {
		remote.Enabled = *raw.Enabled
	}
	if raw.Required != nil {
		remote.Required = *raw.Required
	}
	if raw.Provider != nil {
		remote.Provider = *raw.Provider
	}
	if raw.Description != nil {
		remote.Description = *raw.Description
	}
	return remote
}

func applyRawOptions(dst *Options, src *rawOptions) {
	if src.ManageRemotes != nil {
		dst.ManageRemotes = *src.ManageRemotes
	}
	if src.PushBranches != nil {
		dst.PushBranches = *src.PushBranches
	}
	if src.PushTags != nil {
		dst.PushTags = *src.PushTags
	}
	if src.PushNotes != nil {
		dst.PushNotes = *src.PushNotes
	}
	if src.PushAllRefs != nil {
		dst.PushAllRefs = *src.PushAllRefs
	}
	if src.Prune != nil {
		dst.Prune = *src.Prune
	}
	if src.VerifyCleanWorktree != nil {
		dst.VerifyCleanWorktree = *src.VerifyCleanWorktree
	}
	if src.ContinueOnError != nil {
		dst.ContinueOnError = *src.ContinueOnError
	}
	if src.ConfirmDangerousOperations != nil {
		dst.ConfirmDangerousOperations = *src.ConfirmDangerousOperations
	}
	if src.FetchBeforeBackup != nil {
		dst.FetchBeforeBackup = *src.FetchBeforeBackup
	}
	if src.PullBeforeBackup != nil {
		dst.PullBeforeBackup = *src.PullBeforeBackup
	}
	if src.SkipLFS != nil {
		dst.SkipLFS = *src.SkipLFS
	}
	if src.IncludeArchivedBranches != nil {
		dst.IncludeArchivedBranches = *src.IncludeArchivedBranches
	}
	if src.WriteMetadata != nil {
		dst.WriteMetadata = *src.WriteMetadata
	}
}

func applyRawBundle(dst *BundleConfig, src *rawBundle) {
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
	if src.Path != nil && *src.Path != "" {
		dst.Path = *src.Path
	}
	if src.FilenameTemplate != nil && *src.FilenameTemplate != "" {
		dst.FilenameTemplate = *src.FilenameTemplate
	}
	if src.IncludeAllRefs != nil {
		dst.IncludeAllRefs = *src.IncludeAllRefs
	}
}

func unknownTopLevelWarnings(raw []byte) []string {
	var node yaml.Node
	if err := yaml.Unmarshal(raw, &node); err != nil {
		return nil
	}
	if len(node.Content) == 0 {
		return nil
	}
	root := node.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	allowed := map[string]struct{}{
		"version": {}, "repo": {}, "mode": {}, "remotes": {}, "options": {}, "branch_filters": {},
		"tag_filters": {}, "bundle": {}, "logging": {}, "metadata": {},
	}
	var warnings []string
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if _, ok := allowed[key]; !ok {
			warnings = append(warnings, fmt.Sprintf("unknown top-level field %q", key))
		}
	}
	sort.Strings(warnings)
	return warnings
}
