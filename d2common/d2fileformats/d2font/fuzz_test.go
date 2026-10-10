package d2font

import "testing"

// FuzzLoad feeds arbitrary bytes to the font loader; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("Woo!\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"))
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		if ft, err := Load(data); err == nil {
			_ = ft.Marshal()
		}
	})
}
