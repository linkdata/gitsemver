package gitsemver

import (
	"strings"

	xmodsemver "golang.org/x/mod/semver"
)

// canonicalSemverTag converts relaxed git tag forms like "1", "1.2", "v1.2.3"
// into strict canonical semver for validation/comparison.
func canonicalSemverTag(tag string) (canonical string, ok bool) {
	if tag = strings.TrimSpace(tag); tag != "" {
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag
		}
		// Git tags are intentionally limited to numeric MAJOR[.MINOR][.PATCH] forms.
		if xmodsemver.Prerelease(tag) == "" && xmodsemver.Build(tag) == "" {
			canonical = xmodsemver.Canonical(tag)
		}
	}
	ok = canonical != ""
	return
}

func isSemverTag(tag string) bool {
	_, ok := canonicalSemverTag(tag)
	return ok
}

func semverTagGreater(leftTag, rightTag string) bool {
	leftCanonical, _ := canonicalSemverTag(leftTag)
	rightCanonical, _ := canonicalSemverTag(rightTag)
	return xmodsemver.Compare(leftCanonical, rightCanonical) > 0
}
