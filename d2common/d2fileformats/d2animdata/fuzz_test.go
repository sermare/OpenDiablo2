package d2animdata

import "testing"

// FuzzLoad feeds arbitrary bytes to the animdata loader; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 4))
	f.Add(make([]byte, 300))
	f.Fuzz(func(t *testing.T, data []byte) {
		if ad, err := Load(data); err == nil {
			_ = ad.Marshal()
		}
	})
}
