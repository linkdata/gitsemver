package gitsemver

import (
	"regexp"
	"strings"
)

var (
	// reCheckTag  = regexp.MustCompile(`^v\d+(\.\d+(\.\d+)?)?$`)
	reOnlyWords = regexp.MustCompile(`[^\w]`)
)

type GitSemVer struct {
	Git Gitter      // Git
	Env Environment // environment
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
// exists and is set to the string "true" (not case sensitive).
func (vs *GitSemVer) IsEnvTrue(envvar string) bool {
	return strings.ToLower(strings.TrimSpace(vs.Env.Getenv(envvar))) == "true"
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
		defBranch = strings.TrimSpace(defBranch)
		return branchName == defBranch
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

// GetTag returns the semver git version tag matching the current tree, or
// the closest semver tag if none match.
func (vs *GitSemVer) GetTag(repo string) (string, bool) {
	if tag := strings.TrimSpace(vs.Env.Getenv("CI_COMMIT_TAG")); tag != "" {
		return tag, true
	}
	if currtreehash := vs.Git.GetCurrentTreeHash(repo); currtreehash != "" {
		for _, testtag := range vs.Git.GetTags(repo) {
			if vs.Git.GetTreeHash(repo, testtag) == currtreehash {
				return testtag, true
			}
		}
	}
	if tag := vs.Git.GetClosestTag(repo, "HEAD"); tag != "" {
		return tag, false
	}
	return "v0.0.0", false
}

func (vs *GitSemVer) getBranchGitHub(repo string) (branchName string) {
	if branchName = strings.TrimSpace(vs.Env.Getenv("GITHUB_REF_NAME")); branchName != "" {
		if strings.TrimSpace(vs.Env.Getenv("GITHUB_REF_TYPE")) == "tag" {
			for _, branchName = range vs.Git.GetBranchesFromTag(repo, branchName) {
				if vs.IsReleaseBranch(branchName) {
					break
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
func (vs *GitSemVer) GetBranch(repo string) (branchText, branchName string) {
	if branchName = vs.getBranchGitHub(repo); branchName == "" {
		if branchName = vs.getBranchGitLab(repo); branchName == "" {
			branchName = vs.Git.GetBranch(repo)
		}
	}
	branchText = branchName
	if branchText != "" {
		branchText = reOnlyWords.ReplaceAllString(branchText, "-")
		for {
			if newBranchText := strings.ReplaceAll(branchText, "--", "-"); newBranchText != branchText {
				branchText = newBranchText
				continue
			}
			break
		}
		branchText = strings.TrimPrefix(branchText, "-")
		branchText = strings.TrimSuffix(branchText, "-")
		branchText = strings.ToLower(branchText)
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
		var sametree bool
		if vi.Tag, sametree = vs.GetTag(repo); vi.Tag != "" {
			vi.Version = vi.Tag
			vi.Build = vs.GetBuild(repo)
			branchText, branchName := vs.GetBranch(repo)
			vi.Branch = branchName

			if vs.IsReleaseBranch(branchName) && sametree {
				return
			}

			suffix := branchText
			if vi.Build != "" {
				if suffix != "" {
					suffix += "."
				}
				suffix += vi.Build
			}
			if suffix != "" {
				vi.Version += "-" + suffix
			}
		}
	}
	return
}
