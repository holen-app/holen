package main

type SkipError struct {
	Message string
}

func (e *SkipError) Error() string {
	return e.Message
}
