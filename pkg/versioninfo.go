package gitsemver

import (
	"fmt"
	"go/token"
	"os"
	"path"
	"strings"
	"time"
)

type VersionInfo struct {
	Tag     string // git tag, e.g. "v1.2.3"
	Branch  string // git branch, e.g. "mybranch"
	Build   string // git or CI build number, e.g. "456"
	Version string // composite version, e.g. "v1.2.3-mybranch.456"
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
			vi.Version)
	}
	return
}
