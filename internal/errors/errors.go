package errors

// FormattedError is a sentinel error type indicating the error has already
// been formatted and written to stderr by cmd.WriteError. This prevents
// double-printing of errors in cli.Run.
type FormattedError struct {
	Err error
}

func (e FormattedError) Error() string {
	return e.Err.Error()
}

func (e FormattedError) Unwrap() error {
	return e.Err
}

// NewFormattedError wraps an error to indicate it has already been formatted
// and written to stderr. This prevents double-printing in cli.Run.
func NewFormattedError(err error) error {
	return FormattedError{Err: err}
}

// IsFormattedError returns true if the error is a FormattedError sentinel.
func IsFormattedError(err error) bool {
	_, ok := err.(FormattedError)
	return ok
}
