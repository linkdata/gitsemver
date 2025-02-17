[![build](https://github.com/linkdata/gitsemver/actions/workflows/build.yml/badge.svg)](https://github.com/linkdata/gitsemver/actions/workflows/build.yml)
[![coverage](https://coveralls.io/repos/github/linkdata/gitsemver/badge.svg?branch=main)](https://coveralls.io/github/linkdata/gitsemver?branch=main)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/gitsemver)](https://goreportcard.com/report/github.com/linkdata/gitsemver)
[![Docs](https://godoc.org/github.com/linkdata/gitsemver?status.svg)](https://godoc.org/github.com/linkdata/gitsemver)

# gitsemver

Build a [semver](https://semver.org/) compliant version string for a git repository.

Using tree hashes it returns the latest matching semver tag. If no tree hash
match exactly, it falls back to the latest semver tag reachable from the
current HEAD.

If the match is not exact or the current branch is not the default branch
or a protected branch, it creates a work-in-progress semver string like `v0.1.2-myfeature.123`.

Supports raw git repositories as well as GitLab and GitHub builders.

## Print current version of a git repository

```sh
$ go install github.com/linkdata/gitsemver@latest
$ gitsemver $HOME/myreleasedpackage
v1.2.3
```

## Generate a go package file with version information

```go
//go:generate go run github.com/linkdata/gitsemver@latest -gopackage -out version.gen.go
```

Generates a file called `version.gen.go` with contents like

```go
// Code generated by gitsemver at 2025-02-10 07:47:15 UTC DO NOT EDIT.
// branch "mybranch", build 456
package mypackage

const PkgName = "mypackage"
const PkgVersion = "v1.2.3-mybranch.456"
```
