package git

import (
	"path"
	"strings"
)

func FilterRefs(items []string, include []string, exclude []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if matchesFilters(item, include, exclude) {
			out = append(out, item)
		}
	}
	return out
}

func matchesFilters(value string, include []string, exclude []string) bool {
	if len(include) > 0 {
		matched := false
		for _, pattern := range include {
			ok, _ := path.Match(pattern, value)
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, pattern := range exclude {
		ok, _ := path.Match(pattern, value)
		if ok {
			return false
		}
	}
	return true
}

func BranchRefspecs(branches []string) []string {
	out := make([]string, 0, len(branches))
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}
		out = append(out, "refs/heads/"+branch+":refs/heads/"+branch)
	}
	return out
}

func TagRefspecs(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		out = append(out, "refs/tags/"+tag+":refs/tags/"+tag)
	}
	return out
}
