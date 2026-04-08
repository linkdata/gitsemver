package gitsemver_test

import (
	"os"
	"strings"
	"testing"

	gitsemver "github.com/linkdata/gitsemver/internal/gitsemver"
)

func Test_OsEnvironment_Getenv(t *testing.T) {
	const VarName = "MKENV_TEST3141592654"
	env := gitsemver.OsEnvironment{}
	_, expectOk := os.LookupEnv(VarName)
	_, actualOk := env.LookupEnv(VarName)
	if expectOk != actualOk {
		t.Error(actualOk)
	}
	expect := os.Getenv(VarName)
	actual := env.Getenv(VarName)
	if strings.TrimSpace(expect) != actual {
		t.Error(actual)
	}
}

func Test_OsEnvironment_Getenv_TrimsSpace(t *testing.T) {
	const varName = "MKENV_TEST_TRIM_SPACE"
	t.Setenv(varName, " \t trimmed value \n")
	env := gitsemver.OsEnvironment{}
	if got := env.Getenv(varName); got != "trimmed value" {
		t.Fatalf("expected trimmed value, got %q", got)
	}
}
