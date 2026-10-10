package d2datautils

import "fmt"

// RecoverError is deferred by loaders of untrusted binary files whose low-level readers (BitMuncher,
// slices indexed from file offsets) panic on truncated or corrupt input. It turns such a panic
// into an error stored in *err, so a bad file fails to load instead of crashing the engine:
//
//	func Load(data []byte) (f *File, err error) {
//		defer d2datautils.RecoverError("dcc", &err)
//
// Loaders must still bound every allocation derived from the file (see CheckCount); recovering does
// not help against running out of memory.
func RecoverError(what string, err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%s: corrupt or truncated data: %v", what, r)
	}
}

// CheckCount returns an error when a count read from a file cannot possibly fit in the input:
// every element needs at least minBytes and only remaining bytes are left. It guards
// "make([]T, n)" against hostile headers.
func CheckCount(what string, count int64, minBytes, remaining uint64) error {
	if count < 0 {
		return fmt.Errorf("%s: negative count %d", what, count)
	}

	if minBytes == 0 {
		minBytes = 1
	}

	if uint64(count) > remaining/minBytes {
		return fmt.Errorf("%s: count %d exceeds the %d bytes of input", what, count, remaining)
	}

	return nil
}
