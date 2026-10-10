package d2autoscript

import "testing"

// FuzzParse feeds arbitrary script text to the parser and runs whatever parses against a fake
// host for a bounded number of ticks: no panics, and every Advance returns.
func FuzzParse(f *testing.F) {
	f.Add("wait:2;move:10,20;cast:Fire Bolt@3,4;panel:inventory;say:fps;expect:log=hi;exit")
	f.Add("move:npc=Akara;shot:/tmp/a.png")
	f.Add("cast:Teleport;wait:NaN;wait:Inf;wait:-0")
	f.Add("")
	f.Add(";;;;")
	f.Add("move:1e999,-1e999;cast:x@NaN,NaN")

	f.Fuzz(func(t *testing.T, spec string) {
		steps, err := Parse(spec)
		if err != nil {
			return
		}

		h := &fakeHost{}
		r := NewRunner(steps, h)

		for i := 0; i < 300 && !r.Done(); i++ {
			r.Advance(0.5)
		}

		_ = r.Failed()
	})
}
