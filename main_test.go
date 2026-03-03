package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	testMode = true
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %q failed: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b))
}

func TestMainFn(t *testing.T) {
	flag.Parse()
	*flagGoPackage = true
	*flagOut = "test.out"
	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}
	b, err := os.ReadFile("test.out")
	if err == nil {
		defer os.Remove("test.out")
		s := string(b)
		if !strings.Contains(s, "package gitsemver") || !strings.Contains(s, "PkgName = \"gitsemver\"") {
			t.Error(s)
		}
	} else {
		t.Error(err)
	}
}

func TestMainFnBranch(t *testing.T) {
	flag.Parse()
	*flagBranch = true
	*flagOut = "test.out"
	*flagIncPatch = true
	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}
	b, err := os.ReadFile("test.out")
	if err == nil {
		defer os.Remove("test.out")
		s := string(b)
		if !strings.Contains(s, "main") {
			t.Error(s)
		}
	} else {
		t.Error(err)
	}
}

func TestMainError(t *testing.T) {
	exitFn = func(i int) {
		if i == 0 {
			t.Error(i)
		}
		if i == 125 {
			t.Log("didn't get a syscall.Errno")
		}
	}
	flag.Parse()
	*flagOut = "/proc/.nonexistant"
	*flagDebug = true
	main()
}

func TestMainFnIncPatchDoesNotPushOnWriteError(t *testing.T) {
	flag.Parse()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	origGit, origOut, origName := *flagGit, *flagOut, *flagName
	origDebug, origGoPackage := *flagDebug, *flagGoPackage
	origNoFetch, origNoNewline := *flagNoFetch, *flagNoNewline
	origIncPatch, origBranch := *flagIncPatch, *flagBranch
	origTestMode := testMode
	defer func() {
		*flagGit, *flagOut, *flagName = origGit, origOut, origName
		*flagDebug, *flagGoPackage = origDebug, origGoPackage
		*flagNoFetch, *flagNoNewline = origNoFetch, origNoNewline
		*flagIncPatch, *flagBranch = origIncPatch, origBranch
		testMode = origTestMode
	}()

	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	work := filepath.Join(base, "work")

	runGit(t, "", "init", "--bare", "-q", origin)
	runGit(t, "", "clone", "-q", origin, work)
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "a.txt")
	runGit(t, work, "commit", "-q", "-m", "c1")
	runGit(t, work, "tag", "v1.0.0")
	runGit(t, work, "push", "-q", "origin", "HEAD", "--tags")

	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	*flagGit = "git"
	*flagOut = "missing-dir/out.txt"
	*flagName = ""
	*flagDebug = false
	*flagGoPackage = false
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = true
	*flagBranch = false
	testMode = false

	if code := mainfn(); code == 0 {
		t.Fatal("mainfn unexpectedly succeeded")
	}

	localTags := runGit(t, work, "tag", "--list")
	if strings.Contains(localTags, "v1.0.1") {
		t.Fatalf("unexpected local tag v1.0.1: %q", localTags)
	}
	remoteTags := runGit(t, work, "ls-remote", "--tags", "origin")
	if strings.Contains(remoteTags, "refs/tags/v1.0.1") {
		t.Fatalf("unexpected remote tag v1.0.1: %q", remoteTags)
	}
}

func TestMainFnIncPatchDoesNotWriteOutputOnPushError(t *testing.T) {
	flag.Parse()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	origGit, origOut, origName := *flagGit, *flagOut, *flagName
	origDebug, origGoPackage := *flagDebug, *flagGoPackage
	origNoFetch, origNoNewline := *flagNoFetch, *flagNoNewline
	origIncPatch, origBranch := *flagIncPatch, *flagBranch
	origTestMode := testMode
	defer func() {
		*flagGit, *flagOut, *flagName = origGit, origOut, origName
		*flagDebug, *flagGoPackage = origDebug, origGoPackage
		*flagNoFetch, *flagNoNewline = origNoFetch, origNoNewline
		*flagIncPatch, *flagBranch = origIncPatch, origBranch
		testMode = origTestMode
	}()

	work := t.TempDir()
	runGit(t, work, "init", "-q")
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "a.txt")
	runGit(t, work, "commit", "-q", "-m", "c1")
	runGit(t, work, "tag", "v1.0.0")

	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	*flagGit = "git"
	*flagOut = "out.txt"
	*flagName = ""
	*flagDebug = false
	*flagGoPackage = false
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = true
	*flagBranch = false
	testMode = false

	if code := mainfn(); code == 0 {
		t.Fatal("mainfn unexpectedly succeeded")
	}

	if _, err := os.Stat(filepath.Join(work, "out.txt")); err == nil {
		t.Fatal("unexpected output file out.txt")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	localTags := runGit(t, work, "tag", "--list")
	if strings.Contains(localTags, "v1.0.1") {
		t.Fatalf("unexpected local tag v1.0.1: %q", localTags)
	}
}

func TestMainFnIncPatchOverwritesExistingOutputFile(t *testing.T) {
	flag.Parse()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	origGit, origOut, origName := *flagGit, *flagOut, *flagName
	origDebug, origGoPackage := *flagDebug, *flagGoPackage
	origNoFetch, origNoNewline := *flagNoFetch, *flagNoNewline
	origIncPatch, origBranch := *flagIncPatch, *flagBranch
	origTestMode := testMode
	defer func() {
		*flagGit, *flagOut, *flagName = origGit, origOut, origName
		*flagDebug, *flagGoPackage = origDebug, origGoPackage
		*flagNoFetch, *flagNoNewline = origNoFetch, origNoNewline
		*flagIncPatch, *flagBranch = origIncPatch, origBranch
		testMode = origTestMode
	}()

	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	work := filepath.Join(base, "work")

	runGit(t, "", "init", "--bare", "-q", origin)
	runGit(t, "", "clone", "-q", origin, work)
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "a.txt")
	runGit(t, work, "commit", "-q", "-m", "c1")
	runGit(t, work, "tag", "v1.0.0")
	runGit(t, work, "push", "-q", "origin", "HEAD", "--tags")

	if err := os.WriteFile(filepath.Join(work, "out.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	*flagGit = "git"
	*flagOut = "out.txt"
	*flagName = ""
	*flagDebug = false
	*flagGoPackage = false
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = true
	*flagBranch = false
	testMode = false

	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}

	out, err := os.ReadFile(filepath.Join(work, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "old") {
		t.Fatalf("expected out.txt to be replaced, got %q", string(out))
	}
	if !strings.Contains(string(out), "v1.0.1") {
		t.Fatalf("expected out.txt to contain new version, got %q", string(out))
	}
	remoteTags := runGit(t, work, "ls-remote", "--tags", "origin")
	if !strings.Contains(remoteTags, "refs/tags/v1.0.1") {
		t.Fatalf("expected remote tag v1.0.1, got %q", remoteTags)
	}
}
