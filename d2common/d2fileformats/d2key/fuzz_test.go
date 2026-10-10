package d2key

import "testing"

// FuzzParse feeds arbitrary bytes to the key binding parser; it must never panic.
func FuzzParse(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 8))
	f.Add(make([]byte, 400))
	f.Fuzz(func(t *testing.T, data []byte) {
		if k, err := Parse(data); err == nil {
			_ = k.Marshal()
			k.Lookup(1)
		}
	})
}
