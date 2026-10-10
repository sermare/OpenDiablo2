package d2ds1

import "testing"

// FuzzUnmarshal feeds arbitrary bytes to the DS1 loader; it must never panic.
func FuzzUnmarshal(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{18, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Add(append([]byte{18, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}, make([]byte, 200)...))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Unmarshal(data)
	})
}
