// +build freebsd

package mint

// Exit ...
func (testee *Testee) Exit(expectedCode int) Result {
	panic("Exit method can NOT be used on FreeBSD, for now.")
	return Result{ok: false}
}
