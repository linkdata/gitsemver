package main

import (
	"flag"
	"os"
	"strings"
	"testing"
)

func TestMainFn(t *testing.T) {
	flag.Parse()
	*flagGoPackage = true
	*flagOut = "test.out"
	mainfn()
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
	*flagIncPatch = true
	main()
}
