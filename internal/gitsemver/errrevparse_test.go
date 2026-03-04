package gitsemver

import (
	"errors"
	"testing"
)

func Test_errUnexpectedRevParseOutput_Error(t *testing.T) {
	err := &errUnexpectedRevParseOutput{
		expectedTags: 3,
		gotLines:     5,
	}
	if got, want := err.Error(), "unexpected rev-parse output for 3 tags: got 5 lines"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrUnexpectedRevParseOutput) {
		t.Fatal("not ErrUnexpectedRevParseOutput")
	}
}

func Test_NewErrUnexpectedRevParseOutput(t *testing.T) {
	err := NewErrUnexpectedRevParseOutput(2, 1)
	if !errors.Is(err, ErrUnexpectedRevParseOutput) {
		t.Fatal("not ErrUnexpectedRevParseOutput")
	}
	if got, want := err.Error(), "unexpected rev-parse output for 2 tags: got 1 lines"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

