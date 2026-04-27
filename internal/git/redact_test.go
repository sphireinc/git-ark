package git

import "testing"

func TestRedactURLRedactsHTTPSCredentials(t *testing.T) {
	got := RedactURL("https://user:pass@example.com/org/repo.git")
	want := "https://***@example.com/org/repo.git"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRedactURLLeavesSSHURLsAlone(t *testing.T) {
	got := RedactURL("git@github.com:org/repo.git")
	if got != "git@github.com:org/repo.git" {
		t.Fatalf("got %q", got)
	}
}
