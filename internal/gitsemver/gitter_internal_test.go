package gitsemver

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func Test_hasGitMarker_NotRegular(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkfifo is unix-specific")
	}
	dir := t.TempDir()
	gitPath := filepath.Join(dir, ".git")
	if err := syscall.Mkfifo(gitPath, 0o600); err != nil {
		t.Skipf("mkfifo not supported: %v", err)
	}
	yes, err := hasGitMarker(dir)
	if yes {
		t.Fatal("unexpected yes for non-regular .git")
	}
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory, got %v", err)
	}
}

func Test_hasGitMarker_ReadFileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}
	dir := t.TempDir()
	gitPath := filepath.Join(dir, ".git")
	if err := os.WriteFile(gitPath, []byte("gitdir: /tmp/nowhere"), 0o200); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(gitPath, 0o600) }()
	yes, err := hasGitMarker(dir)
	if yes {
		t.Fatal("unexpected yes when .git cannot be read")
	}
	if err == nil {
		t.Fatal("expected read error")
	}
}

func Test_makeCommitMessage(t *testing.T) {
	if got, want := MakeCommitMessage("v1.2.3"), "tag v1.2.3"; got != want {
		t.Fatalf("makeCommitMessage mismatch: got %q want %q", got, want)
	}
	if got, want := MakeCommitMessage(""), "tag"; got != want {
		t.Fatalf("makeCommitMessage mismatch: got %q want %q", got, want)
	}
}
