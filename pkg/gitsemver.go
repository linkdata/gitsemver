package gitsemver

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type GitSemVer struct {
	Git         Gitter      // Git
	Env         Environment // environment
	DebugOut    io.Writer   // if nit nil, write debug output here
	cleanstatus bool        // true if there are no uncommitted changes in current tree
	tags        []GitTag    // tags
}

// New returns a GitSemVer ready to examine
// the git repositories using the given Git binary.
func New(gitBin string) (vs *GitSemVer, err error) {
	var git Gitter
	if git, err = NewDefaultGitter(gitBin); err == nil {
		vs = &GitSemVer{
			Git: git,
			Env: OsEnvironment{},
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

func (vs *GitSemVer) getTreeHash(repo, tag string) (gt GitTag) {
	for i := range vs.tags {
		if vs.tags[i].Tag == tag {
			return vs.tags[i]
		}
	}
	if commit, tree := vs.Git.GetHashes(repo, tag); commit != "" && tree != "" {
		gt.Tag = tag
		gt.Commit = commit
		gt.Tree = tree
		vs.tags = append(vs.tags, gt)
	}
	return
}

func (vs *GitSemVer) examineTags(repo string) {
	vs.cleanstatus = vs.Git.CleanStatus(repo)
	headHashes := vs.getTreeHash(repo, "HEAD")
	vs.Debug("treehash %s: HEAD (clean: %v)\n", headHashes.Tree, vs.cleanstatus)
	for _, testtag := range vs.Git.GetTags(repo) {
		tagtreehashes := vs.getTreeHash(repo, testtag)
		if tagtreehashes.Tree != "" {
			vs.Debug("treehash %s: %q\n", tagtreehashes.Tree, testtag)
			if vs.cleanstatus && tagtreehashes.Tree == headHashes.Tree {
				return
			}
		}
	}
}

// GetTag returns the semver git version tag matching the current tree, or
// the closest semver tag if none match exactly. It also returns a bool
// that is true if the tree hashes match and there are no uncommitted changes.
func (vs *GitSemVer) GetTag(repo string) (string, bool) {
	vs.examineTags(repo)
	if tag := strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_TAG")); tag != "" {
		return tag, true
	}
	head := vs.getTreeHash(repo, "HEAD")
	for _, gt := range vs.tags {
		if gt.Tag != "HEAD" && gt.Tree == head.Tree {
			return gt.Tag, vs.cleanstatus
		}
	}
	if tag := vs.Git.GetClosestTag(repo, "HEAD"); tag != "" {
		found := vs.getTreeHash(repo, tag)
		for _, gt := range vs.tags {
			if gt.Tag != "HEAD" && gt.Tree == found.Tree {
				found = gt
				break
			}
		}
		vs.Debug("treehash %s: %q is closest to HEAD\n", found.Tree, found.Tag)
		return found.Tag, vs.cleanstatus && (found.Tree == head.Tree)
	}
	return "v0.0.0", false
}

func (vs *GitSemVer) getBranchGitHub(repo string) (branchName string) {
	if branchName = strings.TrimSpace(vs.Env.Getenv("GITHUB_BASE_REF")); branchName == "" {
		if branchName = strings.TrimSpace(vs.Env.Getenv("GITHUB_REF_NAME")); branchName != "" {
			if strings.TrimSpace(vs.Env.Getenv("GITHUB_REF_TYPE")) == "tag" {
				for _, branchName = range vs.Git.GetBranchesFromTag(repo, branchName) {
					if vs.IsReleaseBranch(branchName) {
						break
					}
				}
			}
		}
	}
	return
}

func (vs *GitSemVer) getBranchGitLab(repo string) (branchName string) {
	if branchName = strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_REF_NAME")); branchName != "" {
		if strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_TAG")) == branchName {
			for _, branchName = range vs.Git.GetBranchesFromTag(repo, branchName) {
				if vs.IsReleaseBranch(branchName) {
					break
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
func (vs *GitSemVer) GetBranch(repo string) (branchName string) {
	if branchName = vs.Git.GetBranch(repo); branchName == "" {
		if branchName = vs.getBranchGitHub(repo); branchName == "" {
			branchName = vs.getBranchGitLab(repo)
		}
	}
	return
}

// GetBuild returns the build counter. This is taken from the CI system if available,
// otherwise the Git commit count is used. Returns an empty string if no reasonable build
// counter can be found.
func (vs *GitSemVer) GetBuild(repo string) (build string) {
	if build = strings.TrimSpace(vs.Env.Getenv("CI_PIPELINE_IID")); build == "" {
		if build = strings.TrimSpace(vs.Env.Getenv("GITHUB_RUN_NUMBER")); build == "" {
			build = vs.Git.GetBuild(repo)
		}
	}
	return
}

// GetVersion returns a VersionInfo for the source code in the Git repository.
func (vs *GitSemVer) GetVersion(repo string) (vi VersionInfo, err error) {
	if repo, err = vs.Git.CheckGitRepo(repo); err == nil {
		if vi.Tag, vi.SameTree = vs.GetTag(repo); vi.Tag != "" {
			vi.Build = vs.GetBuild(repo)
			vi.Branch = vs.GetBranch(repo)
			vi.IsRelease = vs.IsReleaseBranch(vi.Branch)
			vi.Tags = vs.tags
		}
	}
	return
}
