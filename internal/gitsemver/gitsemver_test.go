package gitsemver_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/internal/gitsemver"
)

func Test_NewVersionStringer_SucceedsNormally(t *testing.T) {
	vs, err := gitsemver.New("git", nil)
	if err != nil {
		t.Error(err)
	}
	if vs == nil {
		t.Error("vs is nil")
	}
}

func Test_NewVersionStringer_FailsWithBadBinary(t *testing.T) {
	vs, err := gitsemver.New("./versionstringer.go", nil)
	if err == nil {
		t.Error("no error")
	}
	if vs != nil {
		t.Error(vs)
	}
}

func isEqual[T comparable](t *testing.T, a, b T) {
	t.Helper()
	if a != b {
		t.Errorf("%v != %v", a, b)
	}
}

func isTrue(t *testing.T, v bool) {
	t.Helper()
	if !v {
		t.Error(v)
	}
}

type MockBatchErrorGitter struct {
	*MockGitter
	batchCalls int
}

func (mg *MockBatchErrorGitter) GetHashesBatch(repo string, tags []string) (hashes []gitsemver.GitTag, err error) {
	mg.batchCalls++
	return nil, errors.New("batch failed")
}

func Test_VersionStringer_IsEnvTrue(t *testing.T) {
	vs := gitsemver.GitSemVer{
		Env: MockEnvironment{
			"TEST_EMPTY":      "",
			"TEST_FALSE":      "false",
			"TEST_TRUE_LOWER": "true",
			"TEST_TRUE_UPPER": "TRUE",
			"TEST_TRUE_1":     "1",
		},
	}
	isEqual(t, vs.IsEnvTrue("TEST_MISSING"), false)
	isEqual(t, vs.IsEnvTrue("TEST_MISSING"), false)
	isEqual(t, vs.IsEnvTrue("TEST_EMPTY"), false)
	isEqual(t, vs.IsEnvTrue("TEST_FALSE"), false)
	isEqual(t, vs.IsEnvTrue("TEST_TRUE_LOWER"), true)
	isEqual(t, vs.IsEnvTrue("TEST_TRUE_UPPER"), true)
	isEqual(t, vs.IsEnvTrue("TEST_TRUE_1"), true)
}

func Test_VersionStringer_IsReleaseBranch(t *testing.T) {
	const branchName = "testbranch"
	env := MockEnvironment{}
	vs := gitsemver.GitSemVer{Env: env}

	isTrue(t, vs.IsReleaseBranch("default"))
	isTrue(t, vs.IsReleaseBranch("main"))
	isTrue(t, vs.IsReleaseBranch("master"))
	isTrue(t, !vs.IsReleaseBranch(branchName))

	env["CI_DEFAULT_BRANCH"] = branchName
	isTrue(t, vs.IsReleaseBranch(branchName))
	delete(env, "CI_DEFAULT_BRANCH")

	env["CI_COMMIT_REF_PROTECTED"] = "true"
	isTrue(t, vs.IsReleaseBranch(branchName))
	delete(env, "CI_COMMIT_REF_PROTECTED")

	env["GITHUB_REF_PROTECTED"] = "true"
	isTrue(t, vs.IsReleaseBranch(branchName))
	delete(env, "GITHUB_REF_PROTECTED")

	isTrue(t, !vs.IsReleaseBranch(branchName))
}

func Test_VersionStringer_GetTag(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}

	var tag string
	var sametree bool
	var err error

	vs := gitsemver.GitSemVer{Git: git, Env: env}
	tag, sametree, err = vs.GetTag("/")
	isEqual(t, "v0.0.0", tag)
	isEqual(t, false, sametree)
	isEqual(t, err, nil)

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	tag, sametree, err = vs.GetTag(".")
	isEqual(t, "v6.0.0", tag)
	isEqual(t, false, sametree)
	isEqual(t, err, nil)

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	git.treehash = "tree-4"
	tag, sametree, err = vs.GetTag(".")
	isEqual(t, "v4.0.0", tag)
	isEqual(t, true, sametree)
	isEqual(t, err, nil)

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	git.treehash = ""
	env["CI_COMMIT_TAG"] = "v3"
	tag, sametree, err = vs.GetTag(".")
	isEqual(t, "v3", tag)
	isEqual(t, true, sametree)
	isEqual(t, err, nil)
	delete(env, "CI_COMMIT_TAG")

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	env["CI_COMMIT_TAG"] = "1foo"
	tag, sametree, err = vs.GetTag(".")
	isEqual(t, "v6.0.0", tag)
	isEqual(t, false, sametree)
	isEqual(t, err, nil)
}

func Test_VersionStringer_GetTag_PropagatesClosestTagError(t *testing.T) {
	env := MockEnvironment{}
	expectedErr := errors.New("closest tag lookup failed")
	git := &MockGitter{closestTagErr: expectedErr}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	_, _, err := vs.GetTag(".")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected closest tag error, got %v", err)
	}
}

func Test_VersionStringer_GetTag_BatchFailureFallsBackToPerTagLookup(t *testing.T) {
	env := MockEnvironment{}
	git := &MockBatchErrorGitter{MockGitter: &MockGitter{}}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	tag, sametree, err := vs.GetTag(".")
	if err != nil {
		t.Fatal(err)
	}
	isEqual(t, "v6.0.0", tag)
	isEqual(t, false, sametree)
	isEqual(t, 1, git.batchCalls)
}

func Test_VersionStringer_GetBranch(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	git.branch = "zomg"
	name, err := vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "zomg", name)
	git.branch = ""

	git.branch = "detached"
	env["GITHUB_REF_NAME"] = "github.branch"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "github.branch", name)
	delete(env, "GITHUB_REF_NAME")
	git.branch = ""

	git.branch = "detached"
	env["GITHUB_HEAD_REF"] = "feature/foo"
	env["GITHUB_BASE_REF"] = "main"
	env["GITHUB_REF_NAME"] = "123/merge"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "feature/foo", name)
	delete(env, "GITHUB_HEAD_REF")
	delete(env, "GITHUB_BASE_REF")
	delete(env, "GITHUB_REF_NAME")
	git.branch = ""

	git.branch = "detached"
	env["GITHUB_REF_TYPE"] = "tag"
	env["GITHUB_REF_NAME"] = "v1.0.0"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "main", name)
	delete(env, "GITHUB_REF_TYPE")
	delete(env, "GITHUB_REF_NAME")
	git.branch = ""

	git.branch = "detached"
	env["GITHUB_BASE_REF"] = "foobranch"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "foobranch", name)
	delete(env, "GITHUB_BASE_REF")
	git.branch = ""

	git.branch = "detached"
	env["CI_COMMIT_REF_NAME"] = "gitlab---branch"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "gitlab---branch", name)
	delete(env, "CI_COMMIT_REF_NAME")
	git.branch = ""

	git.branch = "detached"
	env["CI_COMMIT_TAG"] = "v1.0.0"
	env["CI_COMMIT_REF_NAME"] = "v1.0.0"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "main", name)
	delete(env, "CI_COMMIT_TAG")
	delete(env, "CI_COMMIT_REF_NAME")
	git.branch = ""
}

func Test_VersionStringer_GetBranchFromTag_GitLab(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	env["CI_COMMIT_TAG"] = "v1.0.0"
	env["CI_COMMIT_REF_NAME"] = "v1.0.0"
	name, err := vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "main", name)
}

func Test_VersionStringer_GetBranchFromTag_GitHub(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	env["GITHUB_REF_TYPE"] = "tag"
	env["GITHUB_REF_NAME"] = "v1.0.0"
	name, err := vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "main", name)

	git.branch = "detached"
	env["GITHUB_REF_NAME"] = "v1"
	name, err = vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "", name)
}

func Test_VersionStringer_GetBranchFromTag_GitHub_NoContainingBranch(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	git.branch = "detached"
	env["GITHUB_REF_TYPE"] = "tag"
	env["GITHUB_REF_NAME"] = "v2.5.0"
	name, err := vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "", name)
}

func Test_VersionStringer_GetBranchFromTag_GitLab_NoReleaseBranch(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	git.branch = "detached"
	env["CI_COMMIT_TAG"] = "v1"
	env["CI_COMMIT_REF_NAME"] = "v1"
	name, err := vs.GetBranch(".")
	isEqual(t, err, nil)
	isEqual(t, "", name)
}

func Test_VersionStringer_GetBuild(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	build, err := vs.GetBuild(".")
	isEqual(t, err, nil)
	isEqual(t, "build", build)

	env["CI_PIPELINE_IID"] = "456"
	build, err = vs.GetBuild(".")
	isEqual(t, err, nil)
	isEqual(t, "456", build)
	delete(env, "CI_PIPELINE_IID")

	env["GITHUB_RUN_NUMBER"] = "789"
	build, err = vs.GetBuild(".")
	isEqual(t, err, nil)
	isEqual(t, "789", build)
	delete(env, "CI_PIPELINE_IID")
}

func Test_VersionStringer_GetVersion(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}

	vs := gitsemver.GitSemVer{Git: git, Env: env}
	vi, err := vs.GetVersion("/") // invalid repo
	if err == nil {
		t.Error("no error")
	}
	isEqual(t, "", vi.Version())

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-main.build", vi.Version())

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	git.treehash = "tree-6"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0", vi.Version())
	git.treehash = ""

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	git.branch = "detached"
	env["CI_COMMIT_REF_NAME"] = "HEAD"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-head.build", vi.Version())
	delete(env, "CI_COMMIT_REF_NAME")
	git.branch = ""

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	env["GITHUB_RUN_NUMBER"] = "789"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-main.789", vi.Version())

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	git.branch = "detached"
	env["CI_COMMIT_REF_NAME"] = "*Branch--.--ONE*-*"
	env["GITHUB_RUN_NUMBER"] = "789"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-branch-one.789", vi.Version())

	vs = gitsemver.GitSemVer{Git: git, Env: env}
	env["CI_COMMIT_REF_NAME"] = "main"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-main.789", vi.Version())
}

func Test_VersionStringer_GetVersionDetachedHEAD(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{branch: "detached", treehash: "tree-2"}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	vi, err := vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v2.0.0", vi.Version())
}

func TestGitSemVer_Debug(t *testing.T) {
	vs, err := gitsemver.New("git", nil)
	if err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	vs.DebugOut = &buf
	vs.Debug("foo")
	if x := buf.String(); x != "foo" {
		t.Error(x)
	}
}

func Test_VersionStringer_GetTag_ClosestTagNotOverriddenByUnreachableSameTreeTag(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, nil, "init", "-q")
	runGit(t, repo, nil, "config", "user.email", "test@example.com")
	runGit(t, repo, nil, "config", "user.name", "Test")

	commitAt(t, repo, "a.txt", "root\n", "c1", "2020-01-01T00:00:00Z")
	commitAt(t, repo, "a.txt", "shared\n", "c2", "2020-01-02T00:00:00Z")
	runGit(t, repo, nil, "tag", "v1.0.0")
	commitAt(t, repo, "a.txt", "head\n", "c3", "2020-01-03T00:00:00Z")

	runGit(t, repo, nil, "checkout", "-q", "-b", "side", "HEAD~2")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("shared\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, nil, "commit", "-qam", "side-shared")
	runGit(t, repo, nil, "tag", "v9.0.0")
	runGit(t, repo, nil, "checkout", "-q", "-")

	vs, err := gitsemver.New("git", nil)
	if err != nil {
		t.Fatal(err)
	}
	tag, sameTree, err := vs.GetTag(repo)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.0.0" {
		t.Fatalf("expected closest reachable tag v1.0.0, got %q", tag)
	}
	if sameTree {
		t.Fatalf("expected sameTree false, got true")
	}
}

func Test_VersionStringer_GetTag_PicksHighestMixedPrefixTagOnSameTree(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, nil, "init", "-q")
	runGit(t, repo, nil, "config", "user.email", "test@example.com")
	runGit(t, repo, nil, "config", "user.name", "Test")

	commitAt(t, repo, "a.txt", "a\n", "c1", "2020-01-01T00:00:00Z")
	runGit(t, repo, nil, "tag", "v1.2.3")
	runGit(t, repo, nil, "tag", "2.0.0")

	vs, err := gitsemver.New("git", nil)
	if err != nil {
		t.Fatal(err)
	}
	tag, sameTree, err := vs.GetTag(repo)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "2.0.0" {
		t.Fatalf("expected highest mixed-prefix tag 2.0.0, got %q", tag)
	}
	if !sameTree {
		t.Fatalf("expected sameTree true, got false")
	}
}

func Test_VersionStringer_GetTag_BatchesTagHashLookup(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, nil, "init", "-q")
	runGit(t, repo, nil, "config", "user.email", "test@example.com")
	runGit(t, repo, nil, "config", "user.name", "Test")
	commitAt(t, repo, "a.txt", "a\n", "c1", "2020-01-01T00:00:00Z")

	for i := 1; i <= 10; i++ {
		runGit(t, repo, nil, "tag", "v0.0."+strconv.Itoa(i))
	}

	var buf bytes.Buffer
	vs, err := gitsemver.New("git", &buf)
	if err != nil {
		t.Fatal(err)
	}
	tag, sameTree, err := vs.GetTag(repo)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v0.0.10" {
		t.Fatalf("expected highest tag v0.0.10, got %q", tag)
	}
	if !sameTree {
		t.Fatalf("expected sameTree true, got false")
	}

	revParseCalls := 0
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, " rev-parse ") {
			revParseCalls++
		}
	}
	if revParseCalls != 2 {
		t.Fatalf("expected 2 rev-parse calls (HEAD + batch), got %d\nlog:\n%s", revParseCalls, buf.String())
	}
}
