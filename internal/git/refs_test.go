package git

import (
	"reflect"
	"testing"
)

func TestFilterRefsIncludeAllWhenIncludeEmpty(t *testing.T) {
	got := FilterRefs([]string{"main", "release/1.0"}, nil, nil)
	want := []string{"main", "release/1.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestFilterRefsIncludePatternsWork(t *testing.T) {
	got := FilterRefs([]string{"main", "release/1.0", "wip/foo"}, []string{"release/*"}, nil)
	want := []string{"release/1.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestFilterRefsExcludeWins(t *testing.T) {
	got := FilterRefs([]string{"main", "release/1.0"}, []string{"release/*"}, []string{"release/1.0"})
	want := []string{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestFilterRefsBranchesWithSlashesWork(t *testing.T) {
	got := FilterRefs([]string{"feature/foo", "feature/bar/baz"}, []string{"feature/*"}, nil)
	want := []string{"feature/foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestFilterRefsTagsWithVPatternWork(t *testing.T) {
	got := FilterRefs([]string{"v1.0.0", "test-1"}, []string{"v*"}, nil)
	want := []string{"v1.0.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
