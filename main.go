package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"syscall"

	gitsemver "github.com/linkdata/gitsemver/internal/gitsemver"
)

var writeFileFn = os.WriteFile
var removeFileFn = os.Remove

func replaceFile(source, target string) (err error) {
	bakname := fmt.Sprintf("%s.gitsemver-%x", target, rand.Uint64())                                       // #nosec G404
	if renameerr := os.Rename(target, bakname); renameerr == nil || errors.Is(renameerr, fs.ErrNotExist) { // #nosec G703
		if err = os.Rename(source, target); err == nil { // #nosec G703
			if renameerr == nil {
				// The replacement already succeeded. Treat backup deletion as best-effort.
				_ = removeFileFn(bakname) // #nosec G703
			}
			return
		}
		if renameerr == nil {
			_ = os.Rename(bakname, target)
		}
	} else {
		err = renameerr
	}
	return
}

func prepareOutput(fileName, content string) (publish func() error, cleanup func(), err error) {
	cleanup = func() {}
	publish = func() error {
		_, err := os.Stdout.WriteString(content)
		return err
	}
	if fileName != "" {
		fileName = filepath.Clean(fileName)
		if fi, statErr := os.Stat(fileName); statErr == nil { // #nosec G703
			if fi.IsDir() {
				err = fmt.Errorf("%q is a directory", fileName)
				return
			}
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			err = statErr
			return
		}
		// File output is always staged: write to temp first, then publish by replace.
		var f *os.File
		if f, err = os.CreateTemp(filepath.Dir(fileName), filepath.Base(fileName)+".gitsemver-*"); err == nil {
			tempFile := f.Name()
			_ = f.Close()
			cleanup = func() {
				_ = os.Remove(tempFile) // #nosec G703
			}
			if err = writeFileFn(tempFile, []byte(content), 0o600); err == nil {
				publish = func() error {
					return replaceFile(tempFile, fileName)
				}
				return
			}
			cleanup()
		}
	}
	return
}

var (
	flagGit       = flag.String("git", "git", "path to Git executable")
	flagOut       = flag.String("out", "", "write to file instead of stdout (relative paths are relative to repo)")
	flagName      = flag.String("name", "", "override the Go PkgName, default is to use last portion of module in go.mod")
	flagDebug     = flag.Bool("debug", false, "write debug info to stderr")
	flagGoPackage = flag.Bool("gopackage", false, "write Go source with PkgName and PkgVersion")
	flagNoFetch   = flag.Bool("nofetch", false, "don't fetch remote tags")
	flagNoNewline = flag.Bool("nonewline", false, "don't print a newline after the output")
	flagIncPatch  = flag.Bool("incpatch", false, "increment the patch level and create a new tag")
	flagBranch    = flag.Bool("branch", false, "print the current branch name")
)

var exitFn func(int) = os.Exit
var testMode bool

func mainfn() int {
	repoDir := os.ExpandEnv(flag.Arg(0))
	if repoDir == "" {
		repoDir = "."
	}

	var debugOut io.Writer
	if *flagDebug {
		debugOut = os.Stderr
	}

	vs, err := gitsemver.New(*flagGit, debugOut)
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
						var clean bool
						if clean, err = vs.Git.CleanStatus(repoDir); err == nil {
							if !clean {
								err = errors.New("cannot use -incpatch with uncommitted changes")
							} else {
								createTag = vi.IncPatch()
								if testMode {
									createTag = ""
								}
							}
						}
					}
					content := vi.Version()
					if *flagBranch {
						content = vi.Branch
					}
					if *flagGoPackage {
						content, err = vi.GoPackage(repoDir, *flagName)
					}
					if err == nil {
						outpath := os.ExpandEnv(*flagOut)
						if outpath != "" && !filepath.IsAbs(outpath) {
							outpath = filepath.Join(repoDir, outpath)
						}
						if !*flagNoNewline {
							content += "\n"
						}
						var publish func() error
						var cleanup func()
						if publish, cleanup, err = prepareOutput(outpath, content); err == nil {
							defer cleanup()
							if err = vs.Git.CreateTag(repoDir, createTag); err == nil {
								if err = vs.Git.PushTag(repoDir, createTag); err == nil {
									if err = publish(); err == nil {
										return 0
									}
									err = errors.Join(err, vs.Git.DeleteRemoteTag(repoDir, createTag))
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
	fmt.Fprintln(os.Stderr, err.Error()) // #nosec G705
	if e := errors.Unwrap(err); e != nil {
		if errno, ok := e.(syscall.Errno); ok {
			retv = int(errno) // #nosec G115
		}
	}
	return retv
}

func main() {
	flag.Parse()
	exitFn(mainfn())
}
