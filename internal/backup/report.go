package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git-ark/internal/config"
)

// successFromResults treats optional failures as warnings and required failures as hard failures.
func successFromResults(results []TargetResult) bool {
	if len(results) == 0 {
		return true
	}
	for _, result := range results {
		if result.Required && !result.Success {
			return false
		}
	}
	return true
}

// resolveBundlePath keeps bundle names stable and human-readable.
func resolveBundlePath(cfg config.Config, repo string, now time.Time) string {
	baseDir := cfg.Bundle.Path
	if baseDir == "" {
		baseDir = "./backups"
	}
	name := filepath.Base(repo)
	timestamp := now.UTC().Format("20060102T150405Z")
	filename := cfg.Bundle.FilenameTemplate
	if filename == "" {
		filename = "{{repo}}-{{timestamp}}.bundle"
	}
	filename = strings.ReplaceAll(filename, "{{repo}}", name)
	filename = strings.ReplaceAll(filename, "{{timestamp}}", timestamp)
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(baseDir, filename)
}

// MetadataPath resolves the on-disk path for the last-backup metadata file.
func MetadataPath(cfg config.Config, repo string) string {
	metaPath := cfg.Metadata.Path
	if metaPath == "" {
		metaPath = ".git/git-ark-last-backup.json"
	}
	if !filepath.IsAbs(metaPath) {
		metaPath = filepath.Join(repo, metaPath)
	}
	return metaPath
}

// ReadMetadata loads and normalizes whatever metadata format is already on disk.
func ReadMetadata(path string) (MetadataFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MetadataFile{}, err
	}
	return decodeMetadata(raw)
}

// WriteMetadata appends the latest report to the persistent metadata history.
func (s *Service) WriteMetadata(cfg config.Config, report Report, repo string) error {
	if !cfg.Options.WriteMetadata {
		return nil
	}

	metaPath := MetadataPath(cfg, repo)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return err
	}

	store, err := loadMetadataStore(metaPath)
	if err != nil {
		return err
	}

	store.prepend(report)
	content, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, content, 0o644)
}

func loadMetadataStore(path string) (MetadataFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return MetadataFile{SchemaVersion: MetadataSchemaVersion, History: []Report{}}, nil
		}
		return MetadataFile{}, err
	}
	return decodeMetadata(raw)
}

func decodeMetadata(raw []byte) (MetadataFile, error) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return MetadataFile{}, err
	}
	if _, ok := keys["schema_version"]; ok || hasMetadataKey(keys, "history") {
		var store MetadataFile
		if err := json.Unmarshal(raw, &store); err != nil {
			return MetadataFile{}, err
		}
		return normalizeMetadataStore(store), nil
	}

	if hasMetadataKey(keys, "repo") || hasMetadataKey(keys, "mode") || hasMetadataKey(keys, "started_at") || hasMetadataKey(keys, "finished_at") || hasMetadataKey(keys, "results") || hasMetadataKey(keys, "success") || hasMetadataKey(keys, "errors") {
		var report Report
		if err := json.Unmarshal(raw, &report); err != nil {
			return MetadataFile{}, err
		}
		return normalizeMetadataStore(MetadataFile{
			SchemaVersion: MetadataSchemaVersion,
			Repo:          report.Repo,
			History:       []Report{report},
		}), nil
	}

	return MetadataFile{}, fmt.Errorf("unsupported metadata format")
}

func normalizeMetadataStore(store MetadataFile) MetadataFile {
	if store.SchemaVersion == 0 {
		store.SchemaVersion = MetadataSchemaVersion
	}
	if store.Repo == "" {
		for _, report := range store.History {
			if report.Repo != "" {
				store.Repo = report.Repo
				break
			}
		}
	}
	if len(store.History) > MetadataHistoryLimit {
		store.History = append([]Report{}, store.History[:MetadataHistoryLimit]...)
	}
	return store
}

func hasMetadataKey(keys map[string]json.RawMessage, key string) bool {
	_, ok := keys[key]
	return ok
}
