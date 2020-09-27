package ant

// Skip represents an error that can be skipped.
//
// When the engine encounters an error it typically
// aborts and returns the error to the caller.
//
// If skip is implemented by the error, the engine
// will not abort and continue.
//
// This is useful in cases where there are deadlinks
// for example, the HTTP fetcher will return true from
// Skip() if the status code is 404.
type Skip interface {
	// Skip returns true if the error should be skipped.
	Skip() bool
}

// Skip returns true if the error can be skipped.
func skip(err error) bool {
	s, ok := err.(Skip)
	return ok && s.Skip()
}
