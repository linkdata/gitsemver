package gitsemver_test

import (
	"os"
	"strings"

	gitsemver "github.com/linkdata/gitsemver/pkg"
)

type MockEnvironment map[string]string

func (me MockEnvironment) Getenv(key string) string {
	return me[key]
}

func (me MockEnvironment) LookupEnv(key string) (val string, ok bool) {
	val, ok = me[key]
	return
}

type mockCommit struct {
	commithash string
	treehash   string
	tag        string
}

var mockHistory = []mockCommit{
	{"HEAD", "tree-HEAD", ""},
	{"commit-6", "tree-6", "v6.0.0"},
	{"commit-5", "tree-5", ""},
	{"commit-4", "tree-4", "v4.0.0"},
	{"commit-3", "tree-3", ""},
	{"commit-2", "tree-2", "v2.0.0"},
	{"commit-1", "tree-1", ""},
}

type MockGitter struct {
	branch   string
	treehash string
	TopTag   string
	dirty    bool
}

func (mg *MockGitter) CheckGitRepo(dir string) (repo string, err error) {
	if dir == "." {
		return ".", nil
	}
	return dir, os.ErrNotExist
}

func (mg *MockGitter) GetCommits(repo string) (commits []string) {
	if repo == "." {
		for _, h := range mockHistory {
			commits = append(commits, h.commithash)
		}
	}
	return
}

func (mg *MockGitter) GetTags(repo string) (tags []string) {
	if repo == "." {
		for _, h := range mockHistory {
			if h.tag != "" {
				tags = append(tags, h.tag)
			}
		}
	}
	return
}

func (mg *MockGitter) GetCurrentTreeHash(repo string) string {
	if repo == "." {
		if mg.treehash == "" {
			return "tree-HEAD"
		}
		return mg.treehash
	}
	return ""
}

func (mg *MockGitter) GetTreeHash(repo, tag string) string {
	if repo == "." {
		for _, h := range mockHistory {
			if h.commithash == tag || h.tag == tag {
				return h.treehash
			}
		}
	}
	return ""
}

func (mg *MockGitter) GetClosestTag(repo, commit string) (tag string) {
	if repo == "." {
		for i := range mockHistory {
			if mockHistory[i].commithash == commit {
				for i < len(mockHistory) {
					if mockHistory[i].tag != "" {
						return mockHistory[i].tag
					}
					i++
				}
			}
		}
	}
	return
}

func (mg *MockGitter) GetBranch(repo string) string {
	if repo == "." {
		if mg.branch == "detached" {
			return ""
		}
		if mg.branch == "" {
			return "main"
		}
		return mg.branch
	}
	return ""
}

func (mg *MockGitter) GetBranchesFromTag(repo, tag string) (branches []string) {
	if strings.HasPrefix(tag, "v1.0") {
		branches = append(branches, "main")
	}
	if strings.HasPrefix(tag, "v1") {
		branches = append(branches, "onepointoh")
	}
	return
}

func (mg *MockGitter) GetBuild(repo string) string {
	if repo == "." {
		return "build"
	}
	return ""
}

func (mg *MockGitter) FetchTags(repo string) error {
	return nil
}

func (mg *MockGitter) CreateTag(repo, tag string) (err error) {
	return
}

func (mg *MockGitter) DeleteTag(repo, tag string) (err error) {
	return
}

func (mg *MockGitter) PushTag(repo, tag string) (err error) {
	return
}

func (mg *MockGitter) CleanStatus(repo string) bool {
	return !mg.dirty
}

var _ gitsemver.Gitter = &MockGitter{}
