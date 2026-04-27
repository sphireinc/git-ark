package git

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectLFSUsage checks the repo for simple Git LFS markers.
// It is intentionally conservative and only looks at the usual attributes files.
func DetectLFSUsage(repo string) (bool, error) {
	candidates := []string{
		filepath.Join(repo, ".gitattributes"),
		filepath.Join(repo, ".git", "info", "attributes"),
	}

	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false, err
		}

		if looksLikeLFS(raw) {
			return true, nil
		}
	}

	return false, nil
}

func looksLikeLFS(raw []byte) bool {
	text := strings.ToLower(string(raw))
	return strings.Contains(text, "filter=lfs") || strings.Contains(text, "diff=lfs") || strings.Contains(text, "merge=lfs")
}
