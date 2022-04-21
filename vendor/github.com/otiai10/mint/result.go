package mint

// Result provide the results of assertion
// for `Dry` option.
type Result struct {
	ok      bool
	message string
}

// OK returns whether result is ok or not.
func (r Result) OK() bool {
	return r.ok
}

// NG is the opposite alias for OK().
func (r Result) NG() bool {
	return !r.ok
}

// Message returns failure message.
func (r Result) Message() string {
	return r.message
}
