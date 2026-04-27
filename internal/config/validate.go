package config

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var providerAllowList = map[string]struct{}{
	"": {}, "github": {}, "gitlab": {}, "bitbucket": {}, "codeberg": {}, "gitea": {}, "generic": {}, "ssh": {}, "https": {},
}

var remoteNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

type ValidationResult struct {
	Warnings []string
}

// Validate checks the config for obvious problems and returns soft warnings for
// settings that are valid but worth a second look.
func Validate(cfg Config) ([]string, error) {
	var warnings []string

	if cfg.Version != CurrentVersion {
		warnings = append(warnings, fmt.Sprintf("config version %d is not the expected version %d", cfg.Version, CurrentVersion))
	}

	cfg.Mode = NormalizeMode(cfg.Mode)
	if !IsValidMode(cfg.Mode) {
		return warnings, fmt.Errorf("invalid mode %q", cfg.Mode)
	}

	if len(cfg.Remotes) == 0 {
		return warnings, fmt.Errorf("no remotes configured")
	}

	enabled := 0
	urls := map[string]string{}
	for name, remote := range cfg.Remotes {
		if !ValidRemoteName(name) {
			return warnings, fmt.Errorf("invalid remote name %q", name)
		}
		if strings.TrimSpace(remote.URL) == "" {
			return warnings, fmt.Errorf("remote %q has empty URL", name)
		}
		if _, ok := providerAllowList[strings.ToLower(remote.Provider)]; !ok {
			return warnings, fmt.Errorf("remote %q has unsupported provider %q", name, remote.Provider)
		}

		diag := DiagnoseProvider(remote)
		warnings = append(warnings, diag.WarningMessages(name)...)

		if remote.Enabled {
			enabled++
		}

		if prior, ok := urls[remote.URL]; ok {
			warnings = append(warnings, fmt.Sprintf("duplicate remote URL used by %q and %q", prior, name))
		} else {
			urls[remote.URL] = name
		}
	}

	if enabled == 0 {
		return warnings, fmt.Errorf("no enabled remotes configured")
	}

	if cfg.Options.Prune {
		warnings = append(warnings, "prune is enabled and may delete refs on backup remotes")
	}

	if cfg.Options.PushAllRefs {
		warnings = append(warnings, "push_all_refs is enabled; this is broader than safe mode and should be reviewed carefully")
	}

	if !cfg.Options.ConfirmDangerousOperations && cfg.Mode == "mirror" {
		warnings = append(warnings, "mirror mode confirmation is disabled")
	}

	if !cfg.Options.ManageRemotes {
		warnings = append(warnings, "remote management is disabled; missing remotes will not be added or updated")
	}

	if cfg.Mode == "mirror" {
		warnings = append(warnings, "mirror mode can delete remote refs")
	}

	return uniqueStrings(warnings), nil
}

// ValidRemoteName keeps the remote names close to what Git itself accepts.
func ValidRemoteName(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".lock") {
		return false
	}
	if strings.Contains(name, " ") || strings.Contains(name, "\t") || strings.Contains(name, "\n") {
		return false
	}
	if strings.Contains(name, "..") || strings.Contains(name, "@{") || strings.Contains(name, "//") {
		return false
	}
	if strings.HasSuffix(name, "/") || strings.HasPrefix(name, "/") || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}
	return remoteNameRE.MatchString(name)
}

// ValidateRemoteURL only checks the URL shape; the backing remote is probed elsewhere.
func ValidateRemoteURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("empty remote URL")
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid URL %q", raw)
		}
	}
	return nil
}

// LooksLikeRemoteURL is a softer check used for warnings and diagnostics.
func LooksLikeRemoteURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.Contains(raw, "://") {
		return ValidateRemoteURL(raw) == nil
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, ".") {
		return true
	}
	if strings.Contains(raw, "@") && strings.Contains(raw, ":") {
		return true
	}
	return false
}

// uniqueStrings keeps warning output tidy without changing the original meaning.
func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
