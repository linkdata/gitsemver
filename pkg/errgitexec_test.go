package gitsemver

import (
	"errors"
	"os"
	"testing"
)

func Test_errGitExec_Error(t *testing.T) {
	tests := []struct {
		name   string
		git    string
		args   []string
		err    error
		stderr string
		want   string
	}{
		{
			name:   "nil",
			git:    "",
			args:   nil,
			err:    nil,
			stderr: "",
			want:   "",
		},
		{
			name:   "os.ErrNotExist",
			git:    "_nope_",
			args:   nil,
			err:    os.ErrNotExist,
			stderr: "",
			want:   "_nope_: " + os.ErrNotExist.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &errGitExec{
				git:    tt.git,
				args:   tt.args,
				err:    tt.err,
				stderr: tt.stderr,
			}
			if got := err.Error(); got != tt.want {
				t.Errorf("errGitExec.Error() = \n got %q\nwant %q\n", got, tt.want)
			}
			if !errors.Is(err, ErrGitExec) {
				t.Error("not ErrGitExec")
			}
			if uw := errors.Unwrap(err); uw != tt.err {
				t.Errorf("unwrap = \n got %#v\nwant %#v\n", uw, tt.err)
			}
		})
	}
}
