package gitsemver_test

import (
	"strings"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/pkg"
)

func Test_VersionInfo_GoPackage(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "v1.2.3", Branch: "mybranch", Build: "456"}

	txt, err := vi.GoPackage("..", "")
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(txt, "package gitsemver") || !strings.Contains(txt, "const PkgName = \"gitsemver\"") {
		t.Error(txt)
	}
	t.Log(txt)
	txt, err = vi.GoPackage("..", "123")
	if err == nil {
		t.Error("no error")
	}
	if txt != "" {
		t.Error(txt)
	}
}

func Test_VersionInfo_IncPatch(t *testing.T) {
	vi := &gitsemver.VersionInfo{Tag: "v1.2"}
	vi.IncPatch()
	if vi.Tag != "v1.2.1" {
		t.Error(vi.Tag)
	}
	vi.IncPatch()
	if vi.Tag != "v1.2.2" {
		t.Error(vi.Tag)
	}
}

func Test_CleanBranch(t *testing.T) {
	isEqual(t, "branch-with-dots", gitsemver.CleanBranch("-branch.with..dots"))
	isEqual(t, "gitlab-branch", gitsemver.CleanBranch("gitlab---branch"))
	isEqual(t, "github-branch", gitsemver.CleanBranch("github.branch."))
}
