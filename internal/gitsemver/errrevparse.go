package gitsemver

import "fmt"

type errUnexpectedRevParseOutput struct {
	expectedTags int
	gotLines     int
}

// ErrUnexpectedRevParseOutput classifies errors where rev-parse output line
// count does not match the requested tag/tree pairs.
var ErrUnexpectedRevParseOutput = &errUnexpectedRevParseOutput{}

func NewErrUnexpectedRevParseOutput(expectedTags, gotLines int) error {
	return &errUnexpectedRevParseOutput{
		expectedTags: expectedTags,
		gotLines:     gotLines,
	}
}

func (err *errUnexpectedRevParseOutput) Error() string {
	return fmt.Sprintf(
		"unexpected rev-parse output for %d tags: got %d lines",
		err.expectedTags, err.gotLines,
	)
}

func (err *errUnexpectedRevParseOutput) Is(other error) bool {
	return other == ErrUnexpectedRevParseOutput
}

