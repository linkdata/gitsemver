package gitsemver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/internal/gitsemver"
)

func Test_VersionInfo_GoPackage(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "v1.2.3", Branch: "mybranch", Build: "456"}

	txt, err := vi.GoPackage("../..", "")
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(txt, "package gitsemver") || !strings.Contains(txt, "const PkgName = \"gitsemver\"") {
		t.Error(txt)
	}
	t.Log(txt)
	txt, err = vi.GoPackage("../..", "123")
	if err == nil {
		t.Error("no error")
	}
	if txt != "" {
		t.Error(txt)
	}
}

func Test_VersionInfo_GoPackage_ModuleWithInlineComment(t *testing.T) {
	repo := t.TempDir()
	goMod := "module example.com/my_pkg // inline comment\n\ngo 1.25\n"
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	vi := &gitsemver.VersionInfo{Tag: "v1.2.3", Branch: "mybranch", Build: "456"}
	txt, err := vi.GoPackage(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(txt, "package my_pkg") || !strings.Contains(txt, "const PkgName = \"my_pkg\"") {
		t.Fatalf("unexpected generated package text: %s", txt)
	}
}

func Test_VersionInfo_IncPatch(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "v1.2", Tags: []gitsemver.GitTag{{Tag: "v1.2"}}}
	if !vi.HasTag("v1.2") {
		t.Error("!v1.2")
	}
	if vi.HasTag("v1.2.1") {
		t.Error("v1.2.1")
	}
	vi.IncPatch()
	if vi.Tag != "v1.2.1" {
		t.Error(vi.Tag)
	}
	vi.IncPatch()
	if vi.Tag != "v1.2.2" {
		t.Error(vi.Tag)
	}
}

func Test_VersionInfo_IncPatch_PrereleaseTag(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "v1.2.3-rc.1"}
	if got := vi.IncPatch(); got != "v1.2.4" {
		t.Fatalf("expected v1.2.4, got %q", got)
	}
}

func Test_VersionInfo_IncPatch_InvalidTagNoLoop(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "not-a-semver-tag"}
	if got := vi.IncPatch(); got != "not-a-semver-tag" {
		t.Fatalf("expected unchanged tag, got %q", got)
	}
}

func Test_VersionInfo_IncPatch_OverflowPatchLevel(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "v1.2.9999999999999999999999999"}
	if got := vi.IncPatch(); got != "v1.2.9999999999999999999999999" {
		t.Fatalf("expected unchanged overflow patch tag, got %q", got)
	}
}

func Test_CleanBranch(t *testing.T) {
	isEqual(t, "branch-with-dots", gitsemver.CleanBranch("-branch.with..dots"))
	isEqual(t, "gitlab-branch", gitsemver.CleanBranch("gitlab---branch"))
	isEqual(t, "github-branch", gitsemver.CleanBranch("github.branch."))
	isEqual(t, "feature-x", gitsemver.CleanBranch("feature_x"))
}

func Test_VersionInfo_Version_UsesSemverSafeBranch(t *testing.T) {
	vi := &gitsemver.VersionInfo{
		Tag:       "v1.2.3",
		Branch:    "feature_x",
		Build:     "42",
		IsRelease: false,
		SameTree:  false,
	}
	if got := vi.Version(); got != "v1.2.3-feature-x.42" {
		t.Fatalf("expected semver-safe branch in version, got %q", got)
	}
}
