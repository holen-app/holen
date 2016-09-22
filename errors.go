package main

// SkipError can be returned from a Strategy to indicate that it is unable to
// run the requested utility or version, but no actual error occurred.  For
// instance, the Docker strategy could return this when Docker is not
// installed.
type SkipError struct {
	Message string
}

func (e *SkipError) Error() string {
	return e.Message
}
