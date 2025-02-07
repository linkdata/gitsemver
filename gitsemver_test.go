package gitsemver_test

import (
	"testing"

	"github.com/linkdata/gitsemver"
)

func Test_NewVersionStringer_SucceedsNormally(t *testing.T) {
	vs, err := gitsemver.New("git")
	if err != nil {
		t.Error(err)
	}
	if vs == nil {
		t.Error("vs is nil")
	}
}

func Test_NewVersionStringer_FailsWithBadBinary(t *testing.T) {
	vs, err := gitsemver.New("./versionstringer.go")
	if err == nil {
		t.Error("no error")
	}
	if vs != nil {
		t.Error(vs)
	}
}

func isEqual[T comparable](t *testing.T, a, b T) {
	if a != b {
		t.Errorf("%v != %v", a, b)
	}
}

func isTrue(t *testing.T, v bool) {
	if !v {
		t.Error(v)
	}
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
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	var tag string
	var sametree bool

	tag, sametree = vs.GetTag("/")
	isEqual(t, "v0.0.0", tag)
	isEqual(t, false, sametree)

	tag, sametree = vs.GetTag(".")
	isEqual(t, "v6.0.0", tag)
	isEqual(t, false, sametree)

	git.treehash = "tree-4"
	tag, sametree = vs.GetTag(".")
	isEqual(t, "v4.0.0", tag)
	isEqual(t, true, sametree)

	git.treehash = ""
	env["CI_COMMIT_TAG"] = "v3"
	tag, sametree = vs.GetTag(".")
	isEqual(t, "v3", tag)
	isEqual(t, true, sametree)
}

func Test_VersionStringer_GetBranch(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	git.branch = "zomg"
	text, name := vs.GetBranch(".")
	isEqual(t, "zomg", text)
	isEqual(t, "zomg", name)

	git.branch = "branch.with..dots"
	text, name = vs.GetBranch(".")
	isEqual(t, "branch-with-dots", text)
	isEqual(t, "branch.with..dots", name)

	env["CI_COMMIT_REF_NAME"] = "gitlab---branch"
	text, name = vs.GetBranch(".")
	isEqual(t, "gitlab-branch", text)
	isEqual(t, "gitlab---branch", name)
	delete(env, "CI_COMMIT_REF_NAME")

	env["GITHUB_REF_NAME"] = "github.branch"
	text, name = vs.GetBranch(".")
	isEqual(t, "github.branch", name)
	isEqual(t, "github-branch", text)
	delete(env, "GITHUB_REF_NAME")
}

func Test_VersionStringer_GetBranchFromTag_GitLab(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	env["CI_COMMIT_TAG"] = "v1.0.0"
	env["CI_COMMIT_REF_NAME"] = "v1.0.0"
	text, name := vs.GetBranch(".")
	isEqual(t, "main", name)
	isEqual(t, "main", text)
}

func Test_VersionStringer_GetBranchFromTag_GitHub(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	env["GITHUB_REF_TYPE"] = "tag"
	env["GITHUB_REF_NAME"] = "v1.0.0"
	text, name := vs.GetBranch(".")
	isEqual(t, "main", name)
	isEqual(t, "main", text)

	env["GITHUB_REF_NAME"] = "v1"
	text, name = vs.GetBranch(".")
	isEqual(t, "onepointoh", name)
	isEqual(t, "onepointoh", text)
}

func Test_VersionStringer_GetBuild(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	build := vs.GetBuild(".")
	isEqual(t, "build", build)

	env["CI_PIPELINE_IID"] = "456"
	build = vs.GetBuild(".")
	isEqual(t, "456", build)
	delete(env, "CI_PIPELINE_IID")

	env["GITHUB_RUN_NUMBER"] = "789"
	build = vs.GetBuild(".")
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
	isEqual(t, "", vi.Version)

	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-main.build", vi.Version)

	git.treehash = "tree-6"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0", vi.Version)

	git.treehash = ""
	env["CI_COMMIT_REF_NAME"] = "HEAD"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-head.build", vi.Version)

	delete(env, "CI_COMMIT_REF_NAME")
	env["GITHUB_RUN_NUMBER"] = "789"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-main.789", vi.Version)

	env["CI_COMMIT_REF_NAME"] = "*Branch--.--ONE*-*"
	env["GITHUB_RUN_NUMBER"] = "789"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-branch-one.789", vi.Version)

	env["CI_COMMIT_REF_NAME"] = "main"
	vi, err = vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v6.0.0-main.789", vi.Version)
}

func Test_VersionStringer_GetVersionDetachedHEAD(t *testing.T) {
	env := MockEnvironment{}
	git := &MockGitter{branch: "detached", treehash: "tree-2"}
	vs := gitsemver.GitSemVer{Git: git, Env: env}

	vi, err := vs.GetVersion(".")
	if err != nil {
		t.Error(err)
	}
	isEqual(t, "v2.0.0", vi.Version)
}
