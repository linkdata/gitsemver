package makeversion

import (
	"strings"
	"testing"
)

func Test_VersionInfo_GoPackage(t *testing.T) {
	const VersionText = "v1.2.3-mybranch.456"
	vi := &VersionInfo{Version: VersionText, Branch: "mybranch", Build: "456"}

	txt, err := vi.GoPackage("FooBar")
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(txt, "package foobar") || !strings.Contains(txt, "const PkgName = \"FooBar\"") {
		t.Error(txt)
	}
	txt, err = vi.GoPackage("123")
	if err == nil {
		t.Error("no error")
	}
	if txt != "" {
		t.Error(txt)
	}
}
