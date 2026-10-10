package d2txt

import "testing"

// FuzzLoadDataDictionary feeds arbitrary text to the TXT table parser; it must never panic.
func FuzzLoadDataDictionary(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("a\tb\tc\n1\t2\t3\n"))
	f.Add([]byte("a\tb\r\n1\r\n\r\n\t\t\t\n"))
	f.Add([]byte("\xff\xfe\x00\n\n\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		d := LoadDataDictionary(data)
		if d == nil {
			return
		}

		for i := 0; d.Next() && i < 10000; i++ {
			_ = d.String("a")
			_ = d.Number("b")
			_ = d.List("c")
			_ = d.Bool("a")
		}
	})
}
