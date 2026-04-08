package gitsemver

import (
	"os"
	"strings"
)

// Environment allows us to mock the OS environment.
// Returns values with leading and trailing space trimmed.
type Environment interface {
	Getenv(string) string
	LookupEnv(string) (string, bool)
}

// OsEnvironment calls the OS functions.
type OsEnvironment struct{}

func (OsEnvironment) Getenv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func (OsEnvironment) LookupEnv(key string) (v string, ok bool) {
	v, ok = os.LookupEnv(key)
	v = strings.TrimSpace(v)
	return
}
