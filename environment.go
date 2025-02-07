package makeversion

import "os"

// Environment allows us to mock the OS environment
type Environment interface {
	Getenv(string) string
	LookupEnv(string) (string, bool)
}

// OsEnvironment calls the OS functions.
type OsEnvironment struct{}

func (OsEnvironment) Getenv(key string) string {
	return os.Getenv(key)
}

func (OsEnvironment) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
