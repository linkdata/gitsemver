package gitsemver

import "testing"

func Test_GitSemVer_cacheTag_ReplacesExistingTag(t *testing.T) {
	vs := &GitSemVer{
		tags: []GitTag{
			{Tag: "v1.2.3", Commit: "old-commit", Tree: "old-tree"},
			{Tag: "v2.0.0", Commit: "other-commit", Tree: "other-tree"},
		},
	}

	vs.cacheTag(GitTag{Tag: "v1.2.3", Commit: "new-commit", Tree: "new-tree"})

	if got, want := len(vs.tags), 2; got != want {
		t.Fatalf("len(tags) = %d, want %d", got, want)
	}
	if got, want := vs.tags[0].Commit, "new-commit"; got != want {
		t.Fatalf("tags[0].Commit = %q, want %q", got, want)
	}
	if got, want := vs.tags[0].Tree, "new-tree"; got != want {
		t.Fatalf("tags[0].Tree = %q, want %q", got, want)
	}
}
