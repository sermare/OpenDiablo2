package d2tbl

import "testing"

// FuzzLoadTextDictionary feeds arbitrary bytes to the TBL loader; it must never panic.
func FuzzLoadTextDictionary(f *testing.F) {
	f.Add([]byte{})
	f.Add(exampleData().Marshal())
	f.Add(make([]byte, 21))
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		if td, err := LoadTextDictionary(data); err == nil {
			_ = td.Marshal()
		}
	})
}
