package config

import (
	"strings"
)

func IsValidMode(mode string) bool {
	switch strings.ToLower(mode) {
	case "safe", "mirror", "bundle":
		return true
	default:
		return false
	}
}

func NormalizeMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		return "safe"
	}
	return mode
}
