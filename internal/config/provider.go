package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ProviderDiagnostic captures the useful bits of provider-aware remote checks.
type ProviderDiagnostic struct {
	Provider    string
	DisplayName string
	URLHost     string
	URLScheme   string
	Warnings    []string
	Notes       []string
}

type providerProfile struct {
	DisplayName    string
	CompetingHosts []string
	HTTPSNote      string
}

var providerProfiles = map[string]providerProfile{
	"github": {
		DisplayName:    "GitHub",
		CompetingHosts: []string{"gitlab.com", "bitbucket.org", "codeberg.org"},
		HTTPSNote:      "GitHub HTTPS remotes often need a token for unattended backups.",
	},
	"gitlab": {
		DisplayName:    "GitLab",
		CompetingHosts: []string{"github.com", "bitbucket.org", "codeberg.org"},
		HTTPSNote:      "GitLab HTTPS remotes often need a personal access token or deploy token.",
	},
	"bitbucket": {
		DisplayName:    "Bitbucket",
		CompetingHosts: []string{"github.com", "gitlab.com", "codeberg.org"},
		HTTPSNote:      "Bitbucket HTTPS remotes often need an app password for automation.",
	},
	"codeberg": {
		DisplayName:    "Codeberg",
		CompetingHosts: []string{"github.com", "gitlab.com", "bitbucket.org"},
		HTTPSNote:      "Codeberg HTTPS remotes often need an access token for automation.",
	},
	"gitea": {
		DisplayName:    "Gitea",
		CompetingHosts: []string{"github.com", "gitlab.com", "bitbucket.org", "codeberg.org"},
		HTTPSNote:      "Gitea HTTPS remotes may need a token depending on server settings.",
	},
}

// DiagnoseProvider turns a configured provider label and URL into human-friendly hints.
// It stays conservative on purpose: it only warns when the URL looks like it belongs
// to a different major host, and otherwise keeps the output informational.
func DiagnoseProvider(remote RemoteConfig) ProviderDiagnostic {
	provider := strings.ToLower(strings.TrimSpace(remote.Provider))
	diag := ProviderDiagnostic{Provider: provider}

	if provider == "" {
		return diag
	}

	profile, ok := providerProfiles[provider]
	if !ok {
		return diag
	}

	diag.DisplayName = profile.DisplayName
	diag.URLHost, diag.URLScheme = remoteURLHostAndScheme(remote.URL)

	if diag.URLHost != "" {
		for _, competing := range profile.CompetingHosts {
			if strings.EqualFold(diag.URLHost, competing) {
				diag.Warnings = append(diag.Warnings, fmt.Sprintf("provider %s is configured, but the URL host %s looks like %s", profile.DisplayName, diag.URLHost, providerLabelForHost(competing)))
				break
			}
		}
	}

	if profile.HTTPSNote != "" && strings.EqualFold(diag.URLScheme, "https") {
		diag.Notes = append(diag.Notes, profile.HTTPSNote)
	}

	if diag.URLHost == "" {
		diag.Notes = append(diag.Notes, "URL host could not be determined from the remote string")
	}

	return diag
}

// WarningMessages prefixes any provider mismatch warnings with the remote name.
func (d ProviderDiagnostic) WarningMessages(remoteName string) []string {
	if len(d.Warnings) == 0 {
		return nil
	}

	out := make([]string, 0, len(d.Warnings))
	for _, warning := range d.Warnings {
		out = append(out, fmt.Sprintf("remote %q: %s", remoteName, warning))
	}

	return out
}

// DetailLines returns a small, readable provider summary for `doctor`.
func (d ProviderDiagnostic) DetailLines(remoteName string) []string {
	if d.Provider == "" {
		return nil
	}

	lines := []string{fmt.Sprintf("- %s: provider=%s", remoteName, d.DisplayName)}

	if d.URLHost != "" {
		if d.URLScheme != "" {
			lines = append(lines, fmt.Sprintf("  host=%s scheme=%s", d.URLHost, d.URLScheme))
		} else {
			lines = append(lines, fmt.Sprintf("  host=%s", d.URLHost))
		}
	} else if d.URLScheme != "" {
		lines = append(lines, fmt.Sprintf("  scheme=%s", d.URLScheme))
	}

	if len(d.Warnings) == 0 {
		lines = append(lines, "  note: no obvious provider mismatch")
	} else {
		for _, warning := range d.Warnings {
			lines = append(lines, "  warning: "+warning)
		}
	}
	for _, note := range d.Notes {
		lines = append(lines, "  note: "+note)
	}

	return lines
}

// remoteURLHostAndScheme pulls the host and scheme out of a Git remote URL.
// It understands both normal URLs and the common scp-style SSH form.
func remoteURLHostAndScheme(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", ""
		}
		return strings.ToLower(parsed.Hostname()), strings.ToLower(parsed.Scheme)
	}

	if strings.Contains(raw, "@") && strings.Contains(raw, ":") && !strings.Contains(raw, " ") {
		at := strings.Index(raw, "@")
		afterAt := raw[at+1:]
		host := strings.SplitN(afterAt, ":", 2)[0]
		if host != "" && !strings.Contains(host, "/") {
			return strings.ToLower(host), "ssh"
		}
	}

	return "", ""
}

func providerLabelForHost(host string) string {
	switch strings.ToLower(host) {
	case "github.com":
		return "GitHub"
	case "gitlab.com":
		return "GitLab"
	case "bitbucket.org":
		return "Bitbucket"
	case "codeberg.org":
		return "Codeberg"
	default:
		return host
	}
}
