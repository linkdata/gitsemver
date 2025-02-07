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

// GoPackage returns  a small piece of Go code defining global
// variables named "PkgName" and "PkgVersion"
// with the given pkgName in all lower case and the contents of Version.
// If the pkgName isn't a valid Go identifier, an error is returned.
func (vi *VersionInfo) GoPackage(pkgName string) (string, error) {
	if !token.IsIdentifier(pkgName) {
		return "", fmt.Errorf("%q is not a valid Go identifier", pkgName)
	}
	generatedBy := ""
	if executable, err := os.Executable(); err == nil {
		generatedBy = " by " + path.Base(executable)
	}
	return fmt.Sprintf(`// Code generated%s at %s UTC DO NOT EDIT.
// branch %q, build %s
package %s

const PkgName = %q
const PkgVersion = %q
`,
		generatedBy, time.Now().UTC().Format(time.DateTime),
		vi.Branch, vi.Build,
		strings.ToLower(pkgName),
		pkgName,
		vi.Version), nil
}
