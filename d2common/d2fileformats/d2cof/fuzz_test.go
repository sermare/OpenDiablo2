package d2cof

import "testing"

// FuzzUnmarshal feeds arbitrary bytes to the COF loader; it must never panic.
func FuzzUnmarshal(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 20))
	f.Add(append([]byte{1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, make([]byte, 64)...))
	f.Add(append([]byte{2, 4, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, make([]byte, 400)...))
	f.Fuzz(func(t *testing.T, data []byte) {
		if c, err := Unmarshal(data); err == nil {
			_ = c.Marshal()
		}
	})
}
