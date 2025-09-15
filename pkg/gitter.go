package gitsemver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Gitter is an interface exposing the required Git functionality
type Gitter interface {
	Exec(args ...string) (output []byte, err error)
	// CheckGitRepo checks that the given directory is part of a git repository.
	CheckGitRepo(dir string) (repo string, err error)
	// GetTags returns all tags, sorted by version descending.
	GetTags(repo string) (tags []string, err error)
	// GetCurrentTreeHash returns the current tree hash.
	GetCurrentTreeHash(repo string) (string, error)
	// GetHashes returns the commit and tree hashes for the given tag.
	GetHashes(repo, tag string) (commit string, tree string, err error)
	// GetClosestTag returns the closest semver tag for the given commit hash.
	GetClosestTag(repo, commit string) (tag string, err error)
	// GetBranch returns the current branch in the repository or an empty string.
	GetBranch(repo string) (branch string, err error)
	// GetBranchesFromTag returns the non-HEAD branches in the repository that have the tag, otherwise an empty string.
	GetBranchesFromTag(repo, tag string) (branches []string, err error)
	// GetBuild returns the number of commits in the currently checked out branch as a string, or an empty string
	GetBuild(repo string) (string, error)
	// FetchTags calls "git fetch --tags"
	FetchTags(repo string) error
	// CreateTag creates a new lightweight tag. Does nothing if tag is empty.
	CreateTag(repo, tag string) error
	// DeleteTag deletes the given tag. Does nothing if tag is empty.
	DeleteTag(repo, tag string) (err error)
	// PushTag pushes the given tag to the origin. Does nothing if tag is empty.
	PushTag(repo, tag string) (err error)
	// CleanStatus returns true if there are no uncommitted changes in the repo
	CleanStatus(repo string) (yes bool, err error)
}

type DefaultGitter struct {
	Git      string
	DebugOut io.Writer
}

func (dg DefaultGitter) Exec(args ...string) (output []byte, err error) {
	var sout, serr bytes.Buffer
	cmd := exec.Command(dg.Git, args...) /* #nosec G204 */
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	err = cmd.Run()
	output = bytes.TrimSpace(sout.Bytes())
	stderr := bytes.TrimSpace(serr.Bytes())
	if err != nil {
		err = NewErrGitExec(dg.Git, args, err, string(stderr))
	} else {
		output = append(output, stderr...)
	}
	if dg.DebugOut != nil {
		result := "OK"
		if err != nil {
			result = err.Error()
		}
		fmt.Fprintf(dg.DebugOut, "%q => (%v+%v) %v\n", strings.Join(cmd.Args, " "), len(output), len(stderr), result)
	}
	return
}

func NewDefaultGitter(gitBin string, debugOut io.Writer) (gitter Gitter, err error) {
	if gitBin, err = exec.LookPath(gitBin); err == nil {
		gitter = DefaultGitter{Git: gitBin, DebugOut: debugOut}
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

var reMatchSemver = regexp.MustCompile(`^v?[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?$`)

// GetTags returns all tags, sorted by version descending.
// The latest tag is the first in the list.
func (dg DefaultGitter) GetTags(repo string) (tags []string, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "tag", "--sort=-v:refname"); len(b) > 0 /* #nosec G204 */ {
		for _, tag := range strings.Split(string(b), "\n") {
			if tag = strings.TrimSpace(tag); len(tag) > 1 {
				if reMatchSemver.MatchString(tag) {
					tags = append(tags, tag)
				}
			}
		}
	}
	return
}

// GetCurrentTreeHash returns the current tree hash.
func (dg DefaultGitter) GetCurrentTreeHash(repo string) (hash string, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "write-tree"); len(b) > 0 /* #nosec G204 */ {
		hash = string(b)
	}
	return
}

// GetHashes returns the commit and tree hashes for the given tag.
func (dg DefaultGitter) GetHashes(repo, tag string) (commit, tree string, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "rev-parse", tag, tag+"^{tree}"); err == nil && len(b) > 0 /* #nosec G204 */ {
		hashes := strings.Split(strings.TrimSpace(string(b)), "\n")
		if len(hashes) == 2 {
			commit, tree = hashes[0], hashes[1]
		}
	}
	return
}

// GetClosestTag returns the closest semver tag for the given commit hash.
func (dg DefaultGitter) GetClosestTag(repo, commit string) (tag string, err error) {
	_, _ = dg.Exec("-C", repo, "fetch", "--unshallow", "--tags") // ignore "unshallow on a complete repository does not make sense"
	var b []byte
	if commit == "HEAD" {
		if b, err = dg.Exec("-C", repo, "rev-list", "--tags", "--max-count=1"); err == nil && len(b) > 0 /* #nosec G204 */ {
			if tag, err = dg.GetClosestTag(repo, strings.TrimSpace(string(b))); tag != "" {
				return
			}
		}
	}
	if b, err = dg.Exec("-C", repo, "describe", "--tags", "--match=v[0-9]*", "--match=[0-9]*", "--abbrev=0", commit); len(b) > 0 /* #nosec G204 */ {
		tag = strings.TrimSpace(string(b))
	}
	return
}

func LastName(s string) string {
	if idx := strings.LastIndexByte(s, '/'); idx > -1 {
		s = s[idx+1:]
	}
	return s
}

func (dg DefaultGitter) GetBranchesFromTag(repo, tag string) (branches []string, err error) {
	tag = strings.TrimPrefix(tag, "refs/")
	tag = strings.TrimPrefix(tag, "tags/")
	var b []byte
	if b, err = dg.Exec("-C", repo, "branch", "--all", "--no-color", "--contains", "tags/"+tag); len(b) > 0 /* #nosec G204 */ {
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

func (dg DefaultGitter) GetBranch(repo string) (branch string, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "branch", "--show-current"); len(b) > 0 /* #nosec G204 */ {
		branch = strings.TrimSpace(string(b))
	}
	return
}

func (dg DefaultGitter) GetBuild(repo string) (buildnum string, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "rev-list", "HEAD", "--count"); err == nil && len(b) > 0 /* #nosec G204 */ {
		str := strings.TrimSpace(string(b))
		var num int
		if num, err = strconv.Atoi(str); err == nil && num > 0 {
			buildnum = str
		}
	}
	return
}

func (dg DefaultGitter) FetchTags(repo string) (err error) {
	err = exec.Command(dg.Git, "-C", repo, "fetch", "--tags").Run() /* #nosec G204 */
	return
}

func (dg DefaultGitter) CreateTag(repo, tag string) (err error) {
	if tag != "" {
		_, err = dg.Exec("-C", repo, "tag", tag)
	}
	return
}

func (dg DefaultGitter) DeleteTag(repo, tag string) (err error) {
	if tag != "" {
		_, err = dg.Exec("-C", repo, "tag", "-d", tag)
	}
	return
}

func (dg DefaultGitter) PushTag(repo, tag string) (err error) {
	if tag != "" {
		_, err = dg.Exec("-C", repo, "push", "origin", tag)
	}
	return
}

func (dg DefaultGitter) CleanStatus(repo string) (yes bool, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "status", "--untracked-files=no", "--porcelain"); err == nil {
		yes = len(b) == 0
	}
	return
}
