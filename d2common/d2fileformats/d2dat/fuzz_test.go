package d2dat

import "testing"

// FuzzLoad feeds arbitrary bytes to the palette loader; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 767))
	f.Add(make([]byte, 768))
	f.Add(make([]byte, 769))
	f.Fuzz(func(t *testing.T, data []byte) {
		if p, err := Load(data); err == nil && p != nil {
			_ = p.NumColors()
		}
	})
}
