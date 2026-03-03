package gitsemver_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/pkg"
)

func runGit(t *testing.T, repo string, env map[string]string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %q failed: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b))
}

func commitAt(t *testing.T, repo, fileName, content, message, timestamp string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, fileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, nil, "add", fileName)
	runGit(t, repo, map[string]string{
		"GIT_AUTHOR_DATE":    timestamp,
		"GIT_COMMITTER_DATE": timestamp,
	}, "commit", "-q", "-m", message)
}

func Test_NewDefaultGitter_SucceedsNormally(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if dg == nil {
		t.Error("dg is nil")
	}
}

func Test_CheckGitRepo_SucceedsForCurrent(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
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
	dg, err := gitsemver.NewDefaultGitter("git", nil)
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
	dg, err := gitsemver.NewDefaultGitter("git", nil)
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
			dg, err := gitsemver.NewDefaultGitter("git", nil)
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

func Test_CheckGitRepo_SucceedsForWorktree(t *testing.T) {
	baseRepo := t.TempDir()
	runGit(t, baseRepo, nil, "init", "-q")
	runGit(t, baseRepo, nil, "config", "user.email", "test@example.com")
	runGit(t, baseRepo, nil, "config", "user.name", "Test")
	commitAt(t, baseRepo, "a.txt", "a\n", "c1", "2020-01-01T00:00:00Z")

	worktreePath := filepath.Join(t.TempDir(), "wt")
	runGit(t, baseRepo, nil, "worktree", "add", "-q", worktreePath, "-b", "feature")

	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := dg.CheckGitRepo(worktreePath)
	if err != nil {
		t.Fatal(err)
	}
	if repo != worktreePath {
		t.Fatalf("expected repo %q, got %q", worktreePath, repo)
	}
}

func Test_DefaultGitter_GetBranch(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, err := dg.GetBranch("."); x == "" {
		t.Error("x is empty", err)
	}
	if x, err := dg.GetBranch("/"); x != "" {
		t.Error(x, err)
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
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, err := dg.GetTags("/"); x != nil {
		t.Error(x, err)
	}
	alltags, err := dg.GetTags(".")
	if len(alltags) == 0 {
		t.Error("no tags")
	}
	if err != nil {
		t.Error(err)
	}
}

func Test_DefaultGitter_GetCurrentTreeHash(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, err := dg.GetCurrentTreeHash("/"); x != "" {
		t.Error(x, err)
	}
	s, err := dg.GetCurrentTreeHash(".")
	if len(s) == 0 {
		t.Error("no tree hash")
	}
	if err != nil {
		t.Error(err)
	}
}

func Test_DefaultGitter_GetTreeHash(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, y, err := dg.GetHashes("/", "v1.0.0"); x != "" || y != "" {
		t.Error(x, y, err)
	}
	if x, y, err := dg.GetHashes(".", "v0.0.2"); x != "f9a1633a72ca04515d517a830a2e2835a98767f6" || y != "57562d5fc36ef21a9785fb6afd128e87ab302fae" {
		t.Error(x, y, err)
	}
}

func Test_DefaultGitter_GetClosestTag(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, err := dg.GetClosestTag("/", ""); x != "" {
		t.Error(x, err)
	}
	tag, err := dg.GetClosestTag(".", "f9a1633a72ca04515d517a830a2e2835a98767f6")
	if err != nil {
		t.Error(err)
	}
	if tag != "v0.0.2" {
		t.Error(tag)
	}
	tag, err = dg.GetClosestTag(".", "HEAD")
	if err != nil {
		t.Error(err)
	}
	if tag == "" {
		t.Error("no closest tag for HEAD")
	}
}

func Test_DefaultGitter_GetClosestTag_HEADUsesReachableTag(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, nil, "init", "-q")
	runGit(t, repo, nil, "config", "user.email", "test@example.com")
	runGit(t, repo, nil, "config", "user.name", "Test")

	commitAt(t, repo, "a.txt", "a\n", "c1", "2020-01-01T00:00:00Z")
	runGit(t, repo, nil, "tag", "v1.0.0")

	runGit(t, repo, nil, "checkout", "-q", "-b", "feature")
	commitAt(t, repo, "a.txt", "a\nb\n", "c2", "2020-01-02T00:00:00Z")

	runGit(t, repo, nil, "checkout", "-q", "-b", "other", "HEAD~1")
	commitAt(t, repo, "a.txt", "a\nc\n", "c3", "2020-01-03T00:00:00Z")
	runGit(t, repo, nil, "tag", "v9.0.0")

	runGit(t, repo, nil, "checkout", "-q", "feature")

	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Fatal(err)
	}
	tag, err := dg.GetClosestTag(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.0.0" {
		t.Fatalf("expected closest reachable tag v1.0.0, got %q", tag)
	}
}

func Test_DefaultGitter_GetBranchFromTag(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, err := dg.GetBranchesFromTag("/", "refs/tags/v1.0.0"); x != nil {
		t.Error(x, err)
	}
	if x, err := dg.GetBranchesFromTag(".", "refs/tags/v0.0.2"); slices.Compare(x, []string{"main"}) != 0 {
		t.Error(x, err)
	}
}

func Test_DefaultGitter_GetBuild(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	if x, err := dg.GetBuild("/"); x != "" {
		t.Error(x, err)
	}
	if x, err := dg.GetBuild("."); x == "" {
		t.Error(x, err)
	}
}

func Test_maybeSync(t *testing.T) {
	if f, err := os.CreateTemp("", ""); err == nil {
		defer os.Remove(f.Name())
		gitsemver.MaybeSync(f)
	}
}

func Test_DefaultGitter_FetchTags(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	dg.FetchTags(".")
}

func Test_DefaultGitter_CreateDeleteTag(t *testing.T) {
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Error(err)
	}
	err = dg.CreateTag(".", "test-tag")
	if err != nil {
		t.Error(err)
	}
	err = dg.DeleteTag(".", "test-tag")
	if err != nil {
		t.Error(err)
	}
}

func Test_DefaultGitter_PushTag(t *testing.T) {
	var buf bytes.Buffer
	dg, err := gitsemver.NewDefaultGitter("git", &buf)
	if err != nil {
		t.Error(err)
	}
	err = dg.PushTag(".", "v1.0.0")
	if err != nil {
		t.Error(err)
	}
	err = dg.PushTag(".", "test-tag")
	if err == nil {
		t.Error("no error")
	} else {
		t.Log(err)
		if buf.Len() == 0 {
			t.Error("no log?")
		}
	}
}

func Test_DefaultGitter_CleanStatus(t *testing.T) {
	var buf bytes.Buffer
	dg, err := gitsemver.NewDefaultGitter("git", &buf)
	if err != nil {
		t.Error(err)
	}
	isclean, err := dg.CleanStatus(".")
	if err != nil {
		t.Error(err)
	}
	if !isclean {
		t.Log("git status reports uncommitted changes")
	} else {
		t.Log("git status reports clean")
	}
	if buf.Len() == 0 {
		t.Error("no log?")
	}
}
