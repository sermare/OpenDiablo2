package d2dc6

import "testing"

// FuzzLoad feeds arbitrary bytes to the DC6 loader and decoder; it must never panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte{})
	f.Add(getExampleDC6().Marshal())
	f.Add(make([]byte, 24))
	f.Add(make([]byte, 200))
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := Load(data)
		if err != nil {
			return
		}

		for i := 0; i < len(d.Frames) && i < 64; i++ {
			_ = d.DecodeFrame(i)
		}
	})
}
