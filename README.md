[![build](https://github.com/linkdata/gitsemver/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/gitsemver/actions/workflows/go.yml)
[![coverage](https://coveralls.io/repos/github/linkdata/gitsemver/badge.svg?branch=main)](https://coveralls.io/github/linkdata/gitsemver?branch=main)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/gitsemver)](https://goreportcard.com/report/github.com/linkdata/gitsemver)
[![Docs](https://godoc.org/github.com/linkdata/gitsemver?status.svg)](https://godoc.org/github.com/linkdata/gitsemver)

# gitsemver

Build a [semver](https://semver.org/) compliant version string for a git repository.

Using tree hashes it returns the latest matching semver tag. If no tree hash
match exactly, it falls back to the latest semver tag reachable from the
current HEAD.

If the match is not exact or the current branch is not the default branch,
it creates a work-in-progress semver string like "v0.1.2-myfeature.123".
