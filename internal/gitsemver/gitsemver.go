package gitsemver

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GitSemVer holds git metadata used while computing a version.
//
// A GitSemVer instance is intended for one repository per run.
// Reusing a single instance across multiple repositories is unsupported and
// can produce incorrect results because internal tag metadata is cached.
// Reusing a single instance across repeated GetVersion calls after repository
// changes is also unsupported for the same reason.
type GitSemVer struct {
	Git         Gitter      // Git
	Env         Environment // environment
	DebugOut    io.Writer   // if nit nil, write debug output here
	cleanstatus bool        // true if there are no uncommitted changes in current tree
	tags        []GitTag    // cached tags for one repo during one version computation
}

// New returns a GitSemVer ready to examine
// the git repositories using the given Git binary.
func New(gitBin string, debugOut io.Writer) (vs *GitSemVer, err error) {
	var git Gitter
	if git, err = NewDefaultGitter(gitBin, debugOut); err == nil {
		vs = &GitSemVer{
			Git:      git,
			Env:      OsEnvironment{},
			DebugOut: debugOut,
		}
	}
	return
}

// IsEnvTrue returns true if the given environment variable
// exists and is set to something that parses as true.
func (vs *GitSemVer) IsEnvTrue(envvar string) (yes bool) {
	yes, _ = strconv.ParseBool(vs.Env.Getenv(envvar))
	return
}

// IsReleaseBranch returns true if the given branch name should
// be allowed to use 'release mode', where the version string
// doesn't contains build information suffix.
func (vs *GitSemVer) IsReleaseBranch(branchName string) bool {
	// A GitLab or GitHub protected branch allows release mode.
	if vs.IsEnvTrue("CI_COMMIT_REF_PROTECTED") || vs.IsEnvTrue("GITHUB_REF_PROTECTED") {
		return true
	}

	// If the branch isn't protected, we only allow release
	// mode for the 'default' branch.

	// GitLab gives us the default branch name directly.
	if defBranch, ok := vs.Env.LookupEnv("CI_DEFAULT_BRANCH"); ok {
		return branchName == strings.TrimSpace(defBranch)
	}

	// Fallback to common default branch names.
	switch branchName {
	case "": // this is the case for a detached HEAD
		return true
	case "default":
		return true
	case "master":
		return true
	case "main":
		return true
	}

	return false
}

// Debug writes debugging output to DebugOut if it's not nil.
func (vs *GitSemVer) Debug(f string, args ...any) {
	if vs.DebugOut != nil {
		_, _ = fmt.Fprintf(vs.DebugOut, f, args...)
	}
}

func (vs *GitSemVer) cacheTag(gt GitTag) {
	for i := range vs.tags {
		if vs.tags[i].Tag == gt.Tag {
			vs.tags[i] = gt
			return
		}
	}
	vs.tags = append(vs.tags, gt)
}

func (vs *GitSemVer) getTreeHash(repo, tag string) (gt GitTag, err error) {
	for i := range vs.tags {
		if vs.tags[i].Tag == tag {
			return vs.tags[i], nil
		}
	}
	var commit, tree string
	if commit, tree, err = vs.Git.GetHashes(repo, tag); commit != "" && tree != "" && err == nil {
		gt.Tag = tag
		gt.Commit = commit
		gt.Tree = tree
		vs.cacheTag(gt)
	}
	return
}

func (vs *GitSemVer) examineTags(repo string) (err error) {
	// Version detection should ignore CI-generated untracked files.
	if vs.cleanstatus, err = vs.Git.CleanStatus(repo, false); err == nil {
		var headHashes GitTag
		if headHashes, err = vs.getTreeHash(repo, "HEAD"); err == nil {
			vs.Debug("treehash %s: HEAD (clean: %v)\n", headHashes.Tree, vs.cleanstatus)
			var tags []string
			if tags, err = vs.Git.GetTags(repo); err == nil {
				if batched, batchErr := vs.Git.GetHashesBatch(repo, tags); batchErr == nil {
					for _, gt := range batched {
						vs.cacheTag(gt)
					}
				} else {
					vs.Debug("treehash batch lookup failed, falling back to per-tag: %v\n", batchErr)
				}
				for _, testtag := range tags {
					var tagtreehashes GitTag
					if tagtreehashes, err = vs.getTreeHash(repo, testtag); err == nil {
						if tagtreehashes.Tree != "" {
							vs.Debug("treehash %s: %q\n", tagtreehashes.Tree, testtag)
							if vs.cleanstatus && tagtreehashes.Tree == headHashes.Tree {
								return
							}
						}
					}
				}
			}
		}
	}
	return
}

// GetTag returns the semver git version tag matching the current tree, or
// the closest semver tag if none match exactly. It also returns a bool
// that is true if the tree hashes match and there are no uncommitted changes.
func (vs *GitSemVer) GetTag(repo string) (tag string, match bool, err error) {
	if ciTag := strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_TAG")); ciTag != "" {
		if isSemverTag(ciTag) {
			return ciTag, true, nil
		}
	}
	tag = "v0.0.0"
	if err = vs.examineTags(repo); err == nil {
		var head GitTag
		if head, err = vs.getTreeHash(repo, "HEAD"); err == nil {
			for _, gt := range vs.tags {
				if gt.Tag != "HEAD" && gt.Tree == head.Tree {
					return gt.Tag, vs.cleanstatus, nil
				}
			}
		}
		var closeToHEAD string
		if closeToHEAD, err = vs.Git.GetClosestTag(repo, "HEAD"); err == nil && closeToHEAD != "" {
			var found GitTag
			if found, err = vs.getTreeHash(repo, closeToHEAD); err == nil {
				vs.Debug("treehash %s: %q is closest to HEAD\n", found.Tree, found.Tag)
				return found.Tag, vs.cleanstatus && (found.Tree == head.Tree), nil
			}
		}
	}
	return
}

type gitHubPullRequestEvent struct {
	PullRequest struct {
		Merged bool `json:"merged"`
	} `json:"pull_request"`
}

func (vs *GitSemVer) getBranchGitHub(repo string) (branchName string, err error) {
	if branchName = strings.TrimSpace(vs.Env.Getenv("GITHUB_BASE_REF")); branchName == "" {
		if refName := strings.TrimSpace(vs.Env.Getenv("GITHUB_REF_NAME")); refName != "" {
			if strings.TrimSpace(vs.Env.Getenv("GITHUB_REF_TYPE")) == "tag" {
				var branches []string
				if branches, err = vs.Git.GetBranchesFromTag(repo, refName); err == nil {
					for _, branchName = range branches {
						if vs.IsReleaseBranch(branchName) {
							return
						}
					}
				}
			}
		}
		branchName = strings.TrimSpace(vs.Env.Getenv("GITHUB_HEAD_REF"))
	}
	return
}

func (vs *GitSemVer) getBranchGitLab(repo string) (branchName string, err error) {
	if branchName = strings.TrimSpace(vs.Env.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME")); branchName == "" {
		if branchName = strings.TrimSpace(vs.Env.Getenv("CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_NAME")); branchName == "" {
			if branchName = strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_REF_NAME")); branchName != "" {
				if strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_TAG")) == branchName {
					var branches []string
					if branches, err = vs.Git.GetBranchesFromTag(repo, branchName); err == nil {
						for _, branchName = range branches {
							if vs.IsReleaseBranch(branchName) {
								return
							}
						}
					}
					branchName = ""
				}
			}
		}
	}
	return
}

// GetBranch returns the current branch as a string suitable
// for inclusion in the semver text as well as the actual
// branch name in the build system or Git. If no branch name
// can be found (for example, in detached HEAD state),
// then an empty string is returned.
func (vs *GitSemVer) GetBranch(repo string) (branchName string, err error) {
	if branchName, err = vs.Git.GetBranch(repo); branchName == "" {
		if branchName, err = vs.getBranchGitHub(repo); branchName == "" {
			branchName, err = vs.getBranchGitLab(repo)
		}
	}
	return
}

// GetBuild returns the build counter. This is taken from the CI system if available,
// otherwise the Git commit count is used. Returns an empty string if no reasonable build
// counter can be found.
func (vs *GitSemVer) GetBuild(repo string) (build string, err error) {
	if build = strings.TrimSpace(vs.Env.Getenv("CI_PIPELINE_IID")); build == "" {
		if build = strings.TrimSpace(vs.Env.Getenv("GITHUB_RUN_NUMBER")); build == "" {
			build, err = vs.Git.GetBuild(repo)
		}
	}
	return
}

// GetVersion returns a VersionInfo for the source code in the Git repository.
// A GitSemVer instance should be treated as single-snapshot state: if the repo
// changes, create a new GitSemVer before calling GetVersion again.
func (vs *GitSemVer) GetVersion(repo string) (vi VersionInfo, err error) {
	if repo, err = vs.Git.CheckGitRepo(repo); err == nil {
		if vi.Tag, vi.SameTree, err = vs.GetTag(repo); vi.Tag != "" && err == nil {
			var e error
			vi.Build, e = vs.GetBuild(repo)
			err = errors.Join(err, e)
			vi.Branch, e = vs.GetBranch(repo)
			err = errors.Join(err, e)
			vi.IsRelease = vs.IsReleaseBranch(vi.Branch)
			vi.Tags = vs.tags
		}
	}
	return
}
