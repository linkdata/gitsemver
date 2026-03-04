package gitsemver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
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
	// FetchTags calls "git fetch --tags". Uses the "--unshallow" option if needed.
	FetchTags(repo string) error
	// CreateTag creates a new lightweight tag. Does nothing if tag is empty.
	CreateTag(repo, tag string) error
	// DeleteTag deletes the given tag. Does nothing if tag is empty.
	DeleteTag(repo, tag string) (err error)
	// PushTag pushes the given tag to the origin. Does nothing if tag is empty.
	PushTag(repo, tag string) (err error)
	// DeleteRemoteTag deletes the given tag from origin. Does nothing if tag is empty.
	DeleteRemoteTag(repo, tag string) (err error)
	// CleanStatus returns true if there are no uncommitted changes in the repo.
	// If includeUntracked is false, untracked files do not affect cleanliness.
	CleanStatus(repo string, includeUntracked bool) (yes bool, err error)
}

type DefaultGitter struct {
	Git      string
	DebugOut io.Writer
}

func MaybeSync(w io.Writer) {
	if syncer, ok := w.(interface{ Sync() error }); ok {
		_ = syncer.Sync()
	}
}

func (dg DefaultGitter) Exec(args ...string) (output []byte, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var sout, serr bytes.Buffer
	cmd := exec.Command(dg.Git, args...) /* #nosec G204 */
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	if dg.DebugOut != nil {
		fmt.Fprintf(dg.DebugOut, "%q =>", strings.Join(cmd.Args, " "))
		MaybeSync(dg.DebugOut)
		defer func(w io.Writer) {
			result := "OK"
			dbgErr := err
			for errors.Unwrap(dbgErr) != nil {
				dbgErr = errors.Unwrap(dbgErr)
			}
			if dbgErr != nil {
				result = dbgErr.Error()
			}
			if serr.Len() > 0 {
				result += fmt.Sprintf(" %q", serr.String())
			}
			fmt.Fprintf(w, " (%v+%v) %v\n", sout.Len(), serr.Len(), result)
			MaybeSync(w)
		}(dg.DebugOut)
	}
	err = cmd.Run()
	output = bytes.TrimSpace(sout.Bytes())
	stderr := bytes.TrimSpace(serr.Bytes())
	if err != nil {
		err = NewErrGitExec(dg.Git, args, err, string(stderr))
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

// hasGitMarker returns true when dir has a valid git marker at ".git":
// either a directory (normal repo) or a file that starts with "gitdir:"
// (worktree).
func hasGitMarker(dir string) (yes bool, err error) {
	gitPath := filepath.Join(dir, ".git")
	var fi os.FileInfo
	if fi, err = os.Stat(gitPath); err != nil {
		return false, err
	}
	if fi.IsDir() {
		return true, nil
	}
	if !fi.Mode().IsRegular() {
		return false, ErrNotDirectory
	}
	var b []byte
	if b, err = os.ReadFile(gitPath); /* #nosec G304 */ err != nil {
		return false, err
	}
	s := strings.TrimSpace(string(b))
	return strings.HasPrefix(s, "gitdir:"), nil
}

// dirOrParentHasGitSubdir returns the name of a directory containing
// a valid '.git' marker or an empty string. It searches starting from
// the given directory and looks in that and it's parents.
func dirOrParentHasGitSubdir(s string) (dir string, err error) {
	for {
		var hasMarker bool
		if hasMarker, err = hasGitMarker(s); err == nil && hasMarker {
			return s, nil
		}
		parent := filepath.Dir(s)
		if parent == s {
			return "", err
		}
		s = parent
	}
}

// CheckGitRepo checks that the given directory is part of a git repository,
// meaning that it or one of its parent directories has a valid '.git' marker.
// If it is, it returns the absolute path of the git repo and a nil error.
func (dg DefaultGitter) CheckGitRepo(dir string) (repo string, err error) {
	if dir, err = filepath.Abs(dir); err == nil {
		if repo, err = dirOrParentHasGitSubdir(dir); err != nil {
			repo = dir
		}
	}
	return
}

// Intentionally accepts partial numeric tags (v1, v1.2, v1.2.3),
// with or without a leading "v".
var reMatchSemver = regexp.MustCompile(`^v?[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?$`)

func normalizeNumericString(s string) string {
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return "0"
	}
	return s
}

func compareNumericStrings(a, b string) int {
	a = normalizeNumericString(a)
	b = normalizeNumericString(b)
	if len(a) != len(b) {
		if len(a) > len(b) {
			return 1
		}
		return -1
	}
	if a == b {
		return 0
	}
	if a > b {
		return 1
	}
	return -1
}

func semverPart(tag string, index int) string {
	core := strings.TrimPrefix(tag, "v")
	parts := strings.Split(core, ".")
	if index >= 0 && index < len(parts) {
		return parts[index]
	}
	return "0"
}

func semverGreater(leftTag, rightTag string) bool {
	for idx := 0; idx < 3; idx++ {
		cmp := compareNumericStrings(semverPart(leftTag, idx), semverPart(rightTag, idx))
		if cmp != 0 {
			return cmp > 0
		}
	}
	return false
}

// GetTags returns all tags, sorted by version descending.
// The latest tag is the first in the list.
func (dg DefaultGitter) GetTags(repo string) (tags []string, err error) {
	var b []byte
	if b, err = dg.Exec("-C", repo, "tag", "--sort=-v:refname", "--list", "v[0-9]*", "[0-9]*"); len(b) > 0 /* #nosec G204 */ {
		for _, tag := range strings.Split(string(b), "\n") {
			if tag = strings.TrimSpace(tag); tag != "" && reMatchSemver.MatchString(tag) {
				tags = append(tags, tag)
			}
		}
		// Git's multi-pattern listing can interleave v-prefixed and non-prefixed
		// tags in a way that is not globally version-sorted. Normalize here.
		sort.SliceStable(tags, func(i, j int) bool {
			return semverGreater(tags[i], tags[j])
		})
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
	var listed []byte
	if listed, err = dg.Exec("-C", repo, "tag", "--merged", commit, "--list", "v[0-9]*", "[0-9]*"); err == nil {
		candidates := map[string]struct{}{}
		for _, listedTag := range strings.Fields(string(listed)) {
			candidates[listedTag] = struct{}{}
		}

		if len(candidates) > 0 {
			seen := map[string]struct{}{}
			for i := 0; err == nil && i < len(candidates); i++ {
				// Ask git for the closest tag by ancestry. If it returns a non-strict
				// semver tag (for example "1foo"), exclude it and try again.
				args := []string{
					"-C", repo,
					"describe", "--tags", "--abbrev=0",
					"--match=v[0-9]*",
					"--match=[0-9]*",
				}
				for pattern := range seen {
					args = append(args, "--exclude="+pattern)
				}
				args = append(args, commit)
				var b []byte
				if b, err = dg.Exec(args...); err == nil {
					candidate := strings.TrimSpace(string(b))
					if candidate == "" || reMatchSemver.MatchString(candidate) {
						tag = candidate
						return
					}
					seen[candidate] = struct{}{}
				}
			}
		}
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
		seen := map[string]struct{}{}
		for _, s := range strings.Split(string(b), "\n") {
			if s = strings.TrimSpace(s); len(s) > 1 {
				if !strings.Contains(s, "HEAD") {
					starred := s[0] == '*'
					s = strings.TrimSpace(strings.TrimPrefix(s, "*"))
					// Skip symbolic refs like "origin/HEAD -> origin/main".
					if len(s) > 0 && !strings.Contains(s, " ") {
						if strings.HasPrefix(s, "remotes/") {
							// Normalize "remotes/<remote>/<branch>" into "<branch>".
							s = strings.TrimPrefix(s, "remotes/")
							if idx := strings.IndexByte(s, '/'); idx > -1 {
								s = s[idx+1:]
							}
						}
						if s != "" {
							if _, ok := seen[s]; !ok {
								seen[s] = struct{}{}
								branches = append(branches, s)
							}
							if starred {
								branches = branches[len(branches)-1:]
								break
							}
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
	var b []byte
	if b, err = dg.Exec("-C", repo, "rev-parse", "--is-shallow-repository"); err == nil {
		args := []string{"-C", repo, "fetch", "--tags", "--unshallow"}
		if strings.TrimSpace(string(b)) != "true" {
			args = args[:len(args)-1]
		}
		_, err = dg.Exec(args...) /* #nosec G204 */
	}
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

func (dg DefaultGitter) DeleteRemoteTag(repo, tag string) (err error) {
	if tag != "" {
		_, err = dg.Exec("-C", repo, "push", "--delete", "origin", tag)
	}
	return
}

func (dg DefaultGitter) CleanStatus(repo string, includeUntracked bool) (yes bool, err error) {
	var b []byte
	args := []string{"-C", repo, "status", "--porcelain"}
	if !includeUntracked {
		args = append(args, "--untracked-files=no")
	}
	if b, err = dg.Exec(args...); err == nil {
		yes = len(b) == 0
	}
	return
}
