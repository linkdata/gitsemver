package gitsemver_test

import (
	"os"
	"slices"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/pkg"
)

func Test_NewDefaultGitter_SucceedsNormally(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if dg == nil {
		t.Error("dg is nil")
	}
}

func Test_CheckGitRepo_SucceedsForCurrent(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	repo, err := dg.CheckGitRepo(".")
	if err != nil {
		t.Error(err)
	}
	if repo == "" {
		t.Error(repo)
	}
}

func Test_CheckGitRepo_SucceedsForSubdir(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	repo, err := dg.CheckGitRepo("../pkg/subdir")
	if err != nil {
		t.Error(err)
	}
	if repo == "" {
		t.Error("repo is empty")
	}
}

func Test_CheckGitRepo_FailsForRoot(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	repo, err := dg.CheckGitRepo("/")
	if err == nil {
		t.Error("no error")
	}
	if repo != "/" {
		t.Error(repo)
	}
}

func Test_CheckGitRepo_IgnoresFileNamedGit(t *testing.T) {
	const fileNamedGit = "./subdir/.git"
	if _, err := os.Stat(fileNamedGit); err != nil {
		if f, err := os.Create(fileNamedGit); err == nil {
			defer f.Close()
			defer os.Remove(fileNamedGit)
			dg, err := gitsemver.NewDefaultGitter("git")
			if err != nil {
				t.Error(err)
			}
			repo, err := dg.CheckGitRepo("./subdir")
			if err != nil {
				t.Error(err)
			}
			if gitsemver.LastName(repo) != "gitsemver" {
				t.Error(repo)
			}
		}
	} else {
		t.Logf("warning: '%s' already exists\n", fileNamedGit)
	}
}

func Test_DefaultGitter_GetBranch(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetBranch("."); x == "" {
		t.Error("x is empty")
	}
	if x := dg.GetBranch("/"); x != "" {
		t.Error(x)
	}
}

func Test_LastName(t *testing.T) {
	if x := gitsemver.LastName("foo"); x != "foo" {
		t.Error(x)
	}
	if x := gitsemver.LastName("foo/bar"); x != "bar" {
		t.Error(x)
	}
}

func Test_DefaultGitter_GetTags(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetTags("/"); x != nil {
		t.Error(x)
	}
	alltags := dg.GetTags(".")
	if len(alltags) == 0 {
		t.Error("no tags")
	}
}

func Test_DefaultGitter_GetCurrentTreeHash(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetCurrentTreeHash("/"); x != "" {
		t.Error(x)
	}
	s := dg.GetCurrentTreeHash(".")
	if len(s) == 0 {
		t.Error("no tree hash")
	}
}

func Test_DefaultGitter_GetTreeHash(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetTreeHash("/", "v1.0.0"); x != "" {
		t.Error(x)
	}
	if x := dg.GetTreeHash(".", "v0.0.2"); x != "57562d5fc36ef21a9785fb6afd128e87ab302fae" {
		t.Error(x)
	}
}

func Test_DefaultGitter_GetClosestTag(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetClosestTag("/", ""); x != "" {
		t.Error(x)
	}
	tag := dg.GetClosestTag(".", "f9a1633a72ca04515d517a830a2e2835a98767f6")
	if tag != "v0.0.2" {
		t.Error(tag)
	}
}

func Test_DefaultGitter_GetBranchFromTag(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetBranchesFromTag("/", "refs/tags/v1.0.0"); x != nil {
		t.Error(x)
	}
	if x := dg.GetBranchesFromTag(".", "refs/tags/v0.0.2"); slices.Compare(x, []string{"main"}) != 0 {
		t.Error(x)
	}
}

func Test_DefaultGitter_GetBuild(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetBuild("/"); x != "" {
		t.Error(x)
	}
	if x := dg.GetBuild("."); x == "" {
		t.Error(x)
	}
}

func Test_DefaultGitter_FetchTags(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	dg.FetchTags(".")
}
