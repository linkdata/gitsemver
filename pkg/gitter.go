package gitsemver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Gitter is an interface exposing the required Git functionality
type Gitter interface {
	// CheckGitRepo checks that the given directory is part of a git repository.
	CheckGitRepo(dir string) (repo string, err error)
	// GetTags returns all tags, sorted by version descending.
	GetTags(repo string) (tags []string)
	// GetCurrentTreeHash returns the current tree hash.
	GetCurrentTreeHash(repo string) string
	// GetTreeHash returns the tree hash for the given tag or commit.
	GetTreeHash(repo, tag string) string
	// GetClosestTag returns the closest semver tag for the given commit hash.
	GetClosestTag(repo, commit string) (tag string)
	// GetBranch returns the current branch in the repository or an empty string.
	GetBranch(repo string) string
	// GetBranchesFromTag returns the non-HEAD branches in the repository that have the tag, otherwise an empty string.
	GetBranchesFromTag(repo, tag string) []string
	// GetBuild returns the number of commits in the currently checked out branch as a string, or an empty string
	GetBuild(repo string) string
	// FetchTags calls "git fetch --tags"
	FetchTags(repo string) error
	// CreateTag creates a new lightweight tag. Does nothing if tag is empty.
	CreateTag(repo, tag string) error
	// DeleteTag deletes the given tag. Does nothing if tag is empty.
	DeleteTag(repo, tag string) (err error)
	// PushTag pushes the given tag to the origin. Does nothing if tag is empty.
	PushTag(repo, tag string) (err error)
}

type DefaultGitter string

func NewDefaultGitter(gitBin string) (gitter Gitter, err error) {
	if gitBin, err = exec.LookPath(gitBin); err == nil {
		gitter = DefaultGitter(gitBin)
	}
	return
}

var ErrNotDirectory = errors.New("not a directory")

// checkDir checks that the given path is accessible and is a directory.
// Returns nil if it is, else an error.
func checkDir(dir string) (err error) {
	_, err = os.ReadDir(dir)
	return
}

// dirOrParentHasGitSubdir returns the name of a directory containing
// a '.git' subdirectory or an empty string. It searches starting from
// the given directory and looks in that and it's parents.
func dirOrParentHasGitSubdir(s string) (dir string, err error) {
	if err = checkDir(path.Join(s, ".git")); err != nil {
		s = path.Dir(s)
		if s != "/" {
			if s, e := dirOrParentHasGitSubdir(s); e == nil {
				return s, nil
			}
		}
	} else {
		dir = s
	}
	return
}

// CheckGitRepo checks that the given directory is part of a git repository,
// meaning that it or one of it's parent directories has a '.git' subdirectory.
// If it is, it returns the absolute path of the git repo and a nil error.
func (dg DefaultGitter) CheckGitRepo(dir string) (repo string, err error) {
	if dir, err = filepath.Abs(dir); err == nil {
		if repo, err = dirOrParentHasGitSubdir(dir); err != nil {
			repo = dir
		}
	}
	return
}

// GetTags returns all tags, sorted by version descending.
// The latest tag is the first in the list.
func (dg DefaultGitter) GetTags(repo string) (tags []string) {
	if b, _ := exec.Command(string(dg), "-C", repo, "tag", "--sort=-v:refname").Output(); len(b) > 0 /* #nosec G204 */ {
		for _, tag := range strings.Split(string(b), "\n") {
			if tag = strings.TrimSpace(tag); len(tag) > 1 {
				tags = append(tags, tag)
			}
		}
	}
	return
}

// GetCurrentTreeHash returns the current tree hash.
func (dg DefaultGitter) GetCurrentTreeHash(repo string) string {
	if b, _ := exec.Command(string(dg), "-C", repo, "write-tree").Output(); len(b) > 0 /* #nosec G204 */ {
		return strings.TrimSpace(string(b))
	}
	return ""
}

// GetTagTreeHash returns the tree hash for the given tag or commit hash.
func (dg DefaultGitter) GetTreeHash(repo, tag string) string {
	if b, _ := exec.Command(string(dg), "-C", repo, "rev-parse", tag+"^{tree}").Output(); len(b) > 0 /* #nosec G204 */ {
		return strings.TrimSpace(string(b))
	}
	return ""
}

// GetClosestTag returns the closest semver tag for the given commit hash.
func (dg DefaultGitter) GetClosestTag(repo, commit string) (tag string) {
	_ = exec.Command(string(dg), "-C", repo, "fetch", "--unshallow").Run() //#nosec G204
	if b, _ := exec.Command(string(dg), "-C", repo, "describe", "--tags", "--match=v[0-9]*", "--abbrev=0", commit).Output(); len(b) > 0 /* #nosec G204 */ {
		return strings.TrimSpace(string(b))
	}
	return ""
}

func LastName(s string) string {
	if idx := strings.LastIndexByte(s, '/'); idx > -1 {
		s = s[idx+1:]
	}
	return s
}

func (dg DefaultGitter) GetBranchesFromTag(repo, tag string) (branches []string) {
	tag = strings.TrimPrefix(tag, "refs/")
	tag = strings.TrimPrefix(tag, "tags/")
	if b, _ := exec.Command(string(dg), "-C", repo, "branch", "--all", "--no-color", "--contains", "tags/"+tag).Output(); len(b) > 0 /* #nosec G204 */ {
		for _, s := range strings.Split(string(b), "\n") {
			if s = strings.TrimSpace(s); len(s) > 1 {
				if !strings.Contains(s, "HEAD") {
					starred := s[0] == '*'
					s = strings.TrimSpace(strings.TrimPrefix(s, "*"))
					if len(s) > 0 && !strings.Contains(s, " ") {
						branches = append(branches, LastName(s))
						if starred {
							branches = branches[len(branches)-1:]
							break
						}
					}
				}
			}
		}
	}
	return
}

func (dg DefaultGitter) GetBranch(repo string) (branch string) {
	if b, _ := exec.Command(string(dg), "-C", repo, "branch", "--show-current").Output(); len(b) > 0 /* #nosec G204 */ {
		branch = strings.TrimSpace(string(b))
	}
	return
}

func (dg DefaultGitter) GetBuild(repo string) string {
	if b, _ := exec.Command(string(dg), "-C", repo, "rev-list", "HEAD", "--count").Output(); len(b) > 0 /* #nosec G204 */ {
		str := strings.TrimSpace(string(b))
		if num, err := strconv.Atoi(str); err == nil && num > 0 {
			return str
		}
	}
	return ""
}

func (dg DefaultGitter) FetchTags(repo string) (err error) {
	err = exec.Command(string(dg), "-C", repo, "fetch", "--tags").Run() /* #nosec G204 */
	return
}

func (dg DefaultGitter) CreateTag(repo, tag string) (err error) {
	if tag != "" {
		err = exec.Command(string(dg), "-C", repo, "tag", tag).Run() /* #nosec G204 */
	}
	return
}

func (dg DefaultGitter) DeleteTag(repo, tag string) (err error) {
	if tag != "" {
		err = exec.Command(string(dg), "-C", repo, "tag", "-d", tag).Run() /* #nosec G204 */
	}
	return
}

func (dg DefaultGitter) PushTag(repo, tag string) (err error) {
	if tag != "" {
		var b []byte
		b, err = exec.Command(string(dg), "-C", repo, "push", "origin", tag).CombinedOutput() /* #nosec G204 */
		if err != nil {
			err = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(b)))
		}
	}
	return
}
