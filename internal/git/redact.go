package git

import (
	"net/url"
	"strings"
)

func RedactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if !strings.Contains(raw, "://") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return raw
	}
	if parsed.User.Username() == "" {
		return raw
	}
	parsed.User = nil
	redacted := parsed.String()
	base := parsed.Scheme + "://"
	if strings.HasPrefix(redacted, base) {
		return base + "***@" + strings.TrimPrefix(redacted, base)
	}
	return raw
}
