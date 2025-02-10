package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"syscall"

	gitsemver "github.com/linkdata/gitsemver/pkg"
)

func writeOutput(fileName, content string) (err error) {
	f := os.Stdout
	if len(fileName) > 0 {
		fileName = path.Clean(fileName)
		if f, err = os.Create(fileName); err != nil /* #nosec G304 */ {
			return
		}
		defer f.Close()
	}
	_, err = f.WriteString(content)
	return
}

var (
	flagGit       = flag.String("git", "git", "path to Git executable")
	flagOut       = flag.String("out", "", "write to file instead of stdout (relative paths are relative to repo)")
	flagName      = flag.String("name", "", "override the Go PkgName, default is to use last portion of module in go.mod")
	flagGoPackage = flag.Bool("gopackage", false, "write Go source with PkgName and PkgVersion")
	flagNoFetch   = flag.Bool("nofetch", false, "don't fetch remote tags")
	flagNoNewline = flag.Bool("nonewline", false, "don't print a newline after the output")
	flagIncPatch  = flag.Bool("incpatch", false, "increment the patch level and create a new tag")
)

func mainfn() int {
	repoDir := os.ExpandEnv(flag.Arg(0))
	if repoDir == "" {
		repoDir = "."
	}

	vs, err := gitsemver.New(*flagGit)
	if err == nil {
		var createTag string
		if repoDir, err = vs.Git.CheckGitRepo(repoDir); err == nil {
			if !*flagNoFetch {
				err = vs.Git.FetchTags(repoDir)
			}
			if err == nil {
				var vi gitsemver.VersionInfo
				if vi, err = vs.GetVersion(repoDir); err == nil {
					if *flagIncPatch {
						createTag = vi.IncPatch()
					}
					content := vi.Version()
					if *flagGoPackage {
						content, err = vi.GoPackage(repoDir, *flagName)
					}
					if err == nil {
						outpath := os.ExpandEnv(*flagOut)
						if outpath != "" && !path.IsAbs(outpath) {
							outpath = path.Join(repoDir, outpath)
						}
						if !*flagNoNewline {
							content += "\n"
						}
						if err = writeOutput(outpath, content); err == nil {
							if err = vs.Git.CreateTag(repoDir, createTag); err == nil {
								if err = vs.Git.PushTag(repoDir, createTag); err == nil {
									return 0
								}
							}
						}
					}
				}
			}
		}
		_ = vs.Git.DeleteTag(repoDir, createTag)
	}

	retv := 125
	fmt.Fprintln(os.Stderr, err.Error())
	if e := errors.Unwrap(err); e != nil {
		if errno, ok := e.(syscall.Errno); ok {
			retv = int(errno)
		}
	}
	return retv
}

var exitFn func(int) = os.Exit

func main() {
	flag.Parse()
	exitFn(mainfn())
}
