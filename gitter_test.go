package makeversion

import (
	"os"
	"slices"
	"testing"
)

func Test_NewDefaultGitter_SucceedsNormally(t *testing.T) {
	dg, err := NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if dg == nil {
		t.Error("dg is nil")
	}
}

func Test_CheckGitRepo_SucceedsForCurrent(t *testing.T) {
	dg, err := NewDefaultGitter("git")
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
	dg, err := NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	repo, err := dg.CheckGitRepo("./subdir")
	if err != nil {
		t.Error(err)
	}
	if repo == "" {
		t.Error("repo is empty")
	}
}

func Test_CheckGitRepo_FailsForRootAndFile(t *testing.T) {
	dg, err := NewDefaultGitter("git")
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
	repo, err = dg.CheckGitRepo("./subdir/empty.txt")
	if err == nil {
		t.Error("no error")
	}
	if repo != "" {
		t.Error(repo)
	}
}

func Test_CheckGitRepo_IgnoresFileNamedGit(t *testing.T) {
	const fileNamedGit = "./subdir/.git"
	if _, err := os.Stat(fileNamedGit); err != nil {
		if f, err := os.Create(fileNamedGit); err == nil {
			defer f.Close()
			defer os.Remove(fileNamedGit)
			dg, err := NewDefaultGitter("git")
			if err != nil {
				t.Error(err)
			}
			repo, err := dg.CheckGitRepo("./subdir")
			if err != nil {
				t.Error(err)
			}
			if lastName(repo) != "gitsemver" {
				t.Error(repo)
			}
		}
	} else {
		t.Logf("warning: '%s' already exists\n", fileNamedGit)
	}
}

func Test_DefaultGitter_GetBranch(t *testing.T) {
	dg, err := NewDefaultGitter("git")
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

func Test_lastName(t *testing.T) {
	if x := lastName("foo"); x != "foo" {
		t.Error(x)
	}
	if x := lastName("foo/bar"); x != "bar" {
		t.Error(x)
	}
}

func Test_DefaultGitter_GetTags(t *testing.T) {
	dg, err := NewDefaultGitter("git")
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
	dg, err := NewDefaultGitter("git")
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
	dg, err := NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetTreeHash("/", "v1.0.0"); x != "" {
		t.Error(x)
	}
	if x := dg.GetTreeHash(".", "v0.0.1"); x != "43a03af5130432f9bffee7447e771bf9c91adb08" {
		t.Error(x)
	}
}

func Test_DefaultGitter_GetClosestTag(t *testing.T) {
	dg, err := NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetClosestTag("/", ""); x != "" {
		t.Error(x)
	}
	tag := dg.GetClosestTag(".", "b1803c4de50c416bf07b873e18ba71cc53fdfb66")
	if tag != "v0.0.1" {
		t.Error(tag)
	}
}

func Test_DefaultGitter_GetBranchFromTag(t *testing.T) {
	dg, err := NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	if x := dg.GetBranchesFromTag("/", "refs/tags/v1.0.0"); x != nil {
		t.Error(x)
	}
	if x := dg.GetBranchesFromTag(".", "refs/tags/v0.0.1"); slices.Compare(x, []string{"main"}) != 0 {
		t.Error(x)
	}
}

func Test_DefaultGitter_GetBuild(t *testing.T) {
	dg, err := NewDefaultGitter("git")
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
	dg, err := NewDefaultGitter("git")
	if err != nil {
		t.Error(err)
	}
	dg.FetchTags(".")
}
