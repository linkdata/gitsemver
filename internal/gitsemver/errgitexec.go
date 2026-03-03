package gitsemver

import (
	"fmt"
	"strings"
)

type errGitExec struct {
	git    string
	args   []string
	err    error
	stderr string
}

var ErrGitExec = &errGitExec{}

func NewErrGitExec(git string, args []string, err error, stderr string) error {
	return &errGitExec{
		git:    git,
		args:   args,
		err:    err,
		stderr: stderr,
	}
}

func (err *errGitExec) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%q:", strings.Join(append([]string{err.git}, err.args...), " "))
	if err.err != nil {
		fmt.Fprintf(&sb, " %v", err.err)
	}
	if len(err.stderr) > 0 {
		fmt.Fprintf(&sb, " %q", err.stderr)
	}
	return sb.String()
}

func (err *errGitExec) Is(other error) bool {
	return other == ErrGitExec
}

func (err *errGitExec) Unwrap() error {
	return err.err
}
