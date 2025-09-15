package gitsemver

import (
	"strconv"
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
	var b []byte
	if err.git != "" {
		b = append(b, err.git...)
		for _, arg := range err.args {
			b = append(b, ' ')
			b = strconv.AppendQuote(b, arg)
		}
		b = append(b, ": "...)
	}
	if len(err.stderr) > 0 {
		b = append(b, err.stderr...)
	} else if err.err != nil {
		b = append(b, err.err.Error()...)
	}
	return string(b)
}

func (err *errGitExec) Is(other error) bool {
	return other == ErrGitExec
}

func (err *errGitExec) Unwrap() error {
	return err.err
}
