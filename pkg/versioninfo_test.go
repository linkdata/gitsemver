package gitsemver_test

import (
	"strings"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/pkg"
)

func Test_VersionInfo_GoPackage(t *testing.T) {
	const VersionText = "v1.2.3-mybranch.456"
	vi := &gitsemver.VersionInfo{Version: VersionText, Branch: "mybranch", Build: "456"}

	txt, err := vi.GoPackage("..", "")
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(txt, "package gitsemver") || !strings.Contains(txt, "const PkgName = \"gitsemver\"") {
		t.Error(txt)
	}
	t.Log(txt)
	txt, err = vi.GoPackage("..", "123")
	if err == nil {
		t.Error("no error")
	}
	if txt != "" {
		t.Error(txt)
	}
}
