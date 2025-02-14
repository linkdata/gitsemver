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

var mockHistory = []gitsemver.GitTag{
	{"HEAD", "commit-7", "tree-7"},
	{"v6.0.0", "commit-6", "tree-6"},
	{"", "commit-5", "tree-5"},
	{"v4.0.0", "commit-4", "tree-4"},
	{"", "commit-3", "tree-3"},
	{"v2.0.0", "commit-2", "tree-2"},
	{"", "commit-1", "tree-1"},
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

func (mg *MockGitter) GetTags(repo string) (tags []string) {
	if repo == "." {
		for _, h := range mockHistory {
			if h.Tag != "" && h.Tag != "HEAD" {
				tags = append(tags, h.Tag)
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

func (mg *MockGitter) GetHashes(repo, tag string) (commit, tree string) {
	if repo == "." {
		for _, h := range mockHistory {
			if h.Tag == tag {
				tree := h.Tree
				if tag == "HEAD" && mg.treehash != "" {
					tree = mg.treehash
				}
				return h.Commit, tree
			}
		}
	}
	return "", ""
}

func (mg *MockGitter) GetClosestTag(repo, from string) (tag string) {
	if repo == "." {
		if from == "HEAD" {
			from = mg.treehash
			if from == "" {
				from = mockHistory[0].Tree
			}
		}
		for i := range mockHistory {
			if mockHistory[i].Tree == from {
				for i < len(mockHistory) {
					if mockHistory[i].Tag != "" && mockHistory[i].Tag != "HEAD" {
						return mockHistory[i].Tag
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
