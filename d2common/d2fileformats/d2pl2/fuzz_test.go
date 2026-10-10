package d2pl2

import "testing"

// FuzzLoad feeds arbitrary bytes to the PL2 loader; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 1024))
	f.Add(make([]byte, 1<<16))
	f.Fuzz(func(t *testing.T, data []byte) {
		if p, err := Load(data); err == nil {
			_ = p.Marshal()
		}
	})
}
