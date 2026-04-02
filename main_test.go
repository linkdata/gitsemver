package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/internal/gitsemver"
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

func runGitHead(t *testing.T, repo string) string {
	t.Helper()
	dg, err := gitsemver.NewDefaultGitter("git", nil)
	if err != nil {
		t.Fatal(err)
	}
	head, err := dg.GetHead(repo, false)
	if err != nil {
		t.Fatal(err)
	}
	return head
}

func TestReplaceFile_TargetMissing(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(source, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(source, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Fatalf("unexpected target contents %q", string(got))
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("expected source to be renamed away, got err=%v", err)
	}
}

func TestReplaceFile_RestoreTargetOnSourceRenameError(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "missing-source.txt")
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(source, target); err == nil {
		t.Fatal("expected replaceFile to fail")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old\n" {
		t.Fatalf("target should be restored, got %q", string(got))
	}
}

func TestReplaceFile_RenameTargetError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission test is unix-specific")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(source, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()
	if err := replaceFile(source, target); err == nil {
		t.Fatal("expected replaceFile to fail")
	}
}

func TestReplaceFile_IgnoresBackupRemoveErrorAfterSuccessfulReplace(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(source, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	origRemoveFileFn := removeFileFn
	removeFileFn = func(string) error {
		return errors.New("forced remove error")
	}
	defer func() { removeFileFn = origRemoveFileFn }()

	if err := replaceFile(source, target); err != nil {
		t.Fatalf("replaceFile should ignore backup delete errors, got: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Fatalf("unexpected target contents %q", string(got))
	}
}

func TestPrepareOutput_Stdout(t *testing.T) {
	publish, cleanup, err := prepareOutput("", "hello")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	if err := publish(); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("unexpected stdout %q", string(b))
	}
}

func TestPrepareOutput_RejectsDirectory(t *testing.T) {
	_, _, err := prepareOutput(t.TempDir(), "x")
	if err == nil {
		t.Fatal("expected directory target error")
	}
}

func TestPrepareOutput_StatError(t *testing.T) {
	longName := strings.Repeat("a", 300)
	target := filepath.Join(t.TempDir(), longName, "out.txt")
	_, _, err := prepareOutput(target, "x")
	if err == nil {
		t.Fatal("expected stat error for invalid path")
	}
}

func TestPrepareOutput_CreateTempError(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing", "out.txt")
	_, _, err := prepareOutput(target, "x")
	if err == nil {
		t.Fatal("expected create temp error")
	}
}

func TestPrepareOutput_WriteFileError(t *testing.T) {
	origWriteFileFn := writeFileFn
	writeFileFn = func(string, []byte, os.FileMode) error {
		return errors.New("forced write error")
	}
	defer func() { writeFileFn = origWriteFileFn }()
	target := filepath.Join(t.TempDir(), "out.txt")
	_, _, err := prepareOutput(target, "x")
	if err == nil {
		t.Fatal("expected write error")
	}
}

func TestExitCodeForError_Default(t *testing.T) {
	if got := exitCodeForError(errors.New("boom")); got != 125 {
		t.Fatalf("expected default exit code 125, got %d", got)
	}
}

func TestExitCodeForError_FindsErrnoInJoinedError(t *testing.T) {
	joined := errors.Join(
		errors.New("publish failed"),
		&os.PathError{Op: "open", Path: "/tmp/x", Err: syscall.ENOENT},
	)
	if got := exitCodeForError(joined); got != int(syscall.ENOENT) {
		t.Fatalf("expected exit code %d, got %d", int(syscall.ENOENT), got)
	}
}

func TestMainFn(t *testing.T) {
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
	runGit(t, work, "branch", "-M", "main")
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "go.mod"), []byte("module example.com/gitsemvertest\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "go.mod", "a.txt")
	runGit(t, work, "commit", "-q", "-m", "c1")
	runGit(t, work, "tag", "v1.0.0")
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	*flagGit = "git"
	*flagOut = "test.out"
	*flagName = ""
	*flagDebug = false
	*flagGoPackage = true
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = false
	*flagBranch = false
	testMode = true

	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}
	b, err := os.ReadFile(filepath.Join(work, "test.out"))
	if err == nil {
		s := string(b)
		if !strings.Contains(s, "package gitsemvertest") || !strings.Contains(s, "PkgName = \"gitsemvertest\"") {
			t.Error(s)
		}
	} else {
		t.Error(err)
	}
}

func TestMainFnBranch(t *testing.T) {
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
	runGit(t, work, "branch", "-M", "main")
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
	*flagOut = "test.out"
	*flagName = ""
	*flagDebug = false
	*flagGoPackage = false
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = true
	*flagBranch = true
	testMode = true

	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}
	b, err := os.ReadFile(filepath.Join(work, "test.out"))
	if err == nil {
		s := string(b)
		if !strings.Contains(s, "main") {
			t.Error(s)
		}
	} else {
		t.Error(err)
	}
}

func TestMainFn_IgnoresUntrackedFilesForVersion(t *testing.T) {
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
	runGit(t, work, "branch", "-M", "main")
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "a.txt")
	runGit(t, work, "commit", "-q", "-m", "c1")
	runGit(t, work, "tag", "v1.0.0")
	if err := os.WriteFile(filepath.Join(work, "tmp.generated"), []byte("generated\n"), 0o644); err != nil {
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
	*flagIncPatch = false
	*flagBranch = false
	testMode = false

	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}
	b, err := os.ReadFile(filepath.Join(work, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "v1.0.0\n" {
		t.Fatalf("expected release version despite untracked files, got %q", got)
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

	preHead := runGitHead(t, work)
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
	afterHead := runGitHead(t, work)
	if afterHead != preHead {
		t.Fatalf("expected HEAD to roll back to %q, got %q", preHead, afterHead)
	}
	if status := runGit(t, work, "status", "--porcelain"); status != "" {
		t.Fatalf("expected clean status after rollback, got %q", status)
	}
}

func TestMainFnIncPatchRefusesDirtyTree(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\ndirty\n"), 0o644); err != nil {
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

	if code := mainfn(); code == 0 {
		t.Fatal("mainfn unexpectedly succeeded on dirty tree")
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
	remoteTags := runGit(t, work, "ls-remote", "--tags", "origin")
	if strings.Contains(remoteTags, "refs/tags/v1.0.1") {
		t.Fatalf("unexpected remote tag v1.0.1: %q", remoteTags)
	}
}

func TestMainFnIncPatchRefusesUntrackedFile(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(work, "untracked.txt"), []byte("new\n"), 0o644); err != nil {
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

	if code := mainfn(); code == 0 {
		t.Fatal("mainfn unexpectedly succeeded with untracked file")
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
	remoteTags := runGit(t, work, "ls-remote", "--tags", "origin")
	if strings.Contains(remoteTags, "refs/tags/v1.0.1") {
		t.Fatalf("unexpected remote tag v1.0.1: %q", remoteTags)
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
	runGit(t, work, "add", "out.txt")
	runGit(t, work, "commit", "-q", "-m", "add out")

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

func TestMainFnIncPatchRollsBackRemoteTagOnPublishError(t *testing.T) {
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
	*flagOut = ""
	*flagName = ""
	*flagDebug = false
	*flagGoPackage = false
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = true
	*flagBranch = false
	testMode = false

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	os.Stdout = w
	if code := mainfn(); code == 0 {
		t.Fatal("mainfn unexpectedly succeeded")
	}
	os.Stdout = origStdout
	_ = w.Close()

	localTags := runGit(t, work, "tag", "--list")
	if strings.Contains(localTags, "v1.0.1") {
		t.Fatalf("unexpected local tag v1.0.1: %q", localTags)
	}
	remoteTags := runGit(t, work, "ls-remote", "--tags", "origin")
	if strings.Contains(remoteTags, "refs/tags/v1.0.1") {
		t.Fatalf("unexpected remote tag v1.0.1: %q", remoteTags)
	}
}

func TestMainFnNoFetchDoesNotRunAnyFetch(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "commit", "-qam", "c2")

	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	*flagGit = "git"
	*flagOut = "out.txt"
	*flagName = ""
	*flagDebug = true
	*flagGoPackage = false
	*flagNoFetch = true
	*flagNoNewline = false
	*flagIncPatch = false
	*flagBranch = false
	testMode = true

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	code := mainfn()
	os.Stderr = origStderr
	_ = w.Close()
	logBytes, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if code != 0 {
		t.Fatalf("mainfn failed with code %d, debug log:\n%s", code, string(logBytes))
	}
	logText := string(logBytes)
	if strings.Contains(logText, " fetch --") {
		t.Fatalf("unexpected git fetch while -nofetch is set:\n%s", logText)
	}
}

func TestMainFnIncPatchInTestModeDoesNotCreateTag(t *testing.T) {
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
	testMode = true

	if code := mainfn(); code != 0 {
		t.Fatalf("mainfn failed with code %d", code)
	}

	localTags := runGit(t, work, "tag", "--list")
	if strings.Contains(localTags, "v1.0.1") {
		t.Fatalf("unexpected local tag v1.0.1 in test mode: %q", localTags)
	}
}
