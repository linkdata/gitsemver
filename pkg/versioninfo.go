package gitsemver

import (
	"fmt"
	"go/token"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reOnlyWords = regexp.MustCompile(`[^\w]`)
)

type GitTag struct {
	Tag    string
	Commit string
	Tree   string
}

type VersionInfo struct {
	Tag       string   // git tag, e.g. "v1.2.3"
	Branch    string   // git branch, e.g. "Special--Branch"
	Build     string   // git or CI build number, e.g. "456"
	SameTree  bool     // true if tree hash is identical
	IsRelease bool     // true if the branch is a release branch
	Tags      []GitTag // all tags and their tree hashes
}

func findPackageName(repo, s string) (pkgName string, err error) {
	pkgName = s
	if pkgName == "" {
		var b []byte
		if b, err = os.ReadFile(path.Join(repo, "go.mod")); /*#nosec G304*/ err == nil {
			for _, s := range strings.Split(string(b), "\n") {
				s = strings.TrimSpace(s)
				if strings.HasPrefix(s, "module") {
					pkgName = LastName(s)
				}
			}
		}
	}
	if err == nil && !token.IsIdentifier(pkgName) {
		err = fmt.Errorf("%q is not a valid Go identifier", pkgName)
	}
	return
}

const goPackageTemplate = `// Code generated%s at %s UTC DO NOT EDIT.
// branch %q, build %s
package %s

const PkgName = %q
const PkgVersion = %q
`

// GoPackage returns  a small piece of Go code defining global
// variables named "PkgName" and "PkgVersion"
// with the given pkgName in all lower case and the contents of Version.
// If the pkgName isn't a valid Go identifier, an error is returned.
func (vi *VersionInfo) GoPackage(repo, pkgName string) (retv string, err error) {
	pkgName, err = findPackageName(repo, pkgName)
	if err == nil {
		generatedBy := ""
		if executable, err := os.Executable(); err == nil {
			generatedBy = " by " + path.Base(executable)
		}
		retv = fmt.Sprintf(goPackageTemplate,
			generatedBy, time.Now().UTC().Format(time.DateTime),
			vi.Branch, vi.Build,
			strings.ToLower(pkgName),
			pkgName,
			vi.Version())
	}
	return
}

func (vi *VersionInfo) HasTag(tag string) bool {
	for _, gt := range vi.Tags {
		if gt.Tag == tag {
			return true
		}
	}
	return false
}

// IncPatch increments the patch level of the version, returning the new tag.
func (vi *VersionInfo) IncPatch() string {
	for strings.Count(vi.Tag, ".") < 2 {
		vi.Tag += ".0"
	}
	for {
		patchindex := strings.LastIndexByte(vi.Tag, '.') + 1
		if patchlevel, err := strconv.Atoi(vi.Tag[patchindex:]); err == nil {
			vi.Tag = vi.Tag[:patchindex] + strconv.Itoa(patchlevel+1)
			if !vi.HasTag(vi.Tag) {
				break
			}
		}
	}
	vi.SameTree = true
	return vi.Tag
}

func CleanBranch(branch string) string {
	branch = reOnlyWords.ReplaceAllString(branch, "-")
	for {
		if newSuffix := strings.ReplaceAll(branch, "--", "-"); newSuffix != branch {
			branch = newSuffix
			continue
		}
		break
	}
	branch = strings.TrimPrefix(branch, "-")
	branch = strings.TrimSuffix(branch, "-")
	branch = strings.ToLower(branch)
	return branch
}

// Version returns the composite version, e.g. "v1.2.3-mybranch.456"
func (vi *VersionInfo) Version() (version string) {
	if vi.Tag != "" {
		version = vi.Tag
		if !vi.IsRelease || !vi.SameTree {
			suffix := CleanBranch(vi.Branch)
			if vi.Build != "" {
				if suffix != "" {
					suffix += "."
				}
				suffix += vi.Build
			}
			if suffix != "" {
				version += "-" + suffix
			}
		}
	}
	return
}
