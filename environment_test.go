package makeversion

import (
	"os"
	"testing"
)

func Test_OsEnvironment_Getenv(t *testing.T) {
	const VarName = "MKENV_TEST3141592654"
	env := OsEnvironment{}
	_, expectOk := os.LookupEnv(VarName)
	_, actualOk := env.LookupEnv(VarName)
	if expectOk != actualOk {
		t.Error(actualOk)
	}
	expect := os.Getenv(VarName)
	actual := env.Getenv(VarName)
	if expect != actual {
		t.Error(actual)
	}
}
