package d2dcc

import "testing"

// FuzzLoad feeds arbitrary bytes to the DCC loader; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x74, 6, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0})
	f.Add(append([]byte{0x74, 6, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0}, make([]byte, 100)...))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Load(data)
	})
}
