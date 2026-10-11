package d2monster

import "testing"

func TestDiabRunClient(t *testing.T) {
	p := [7]int{0, 8, 14, 5, 13, 16, 6} // skills.txt row 198
	cases := []struct {
		name      string
		frame     int
		stage     int
		near      bool
		wantStage int
		handled   bool
		run       bool
		seq       int
		total     int
		setSeq    bool
	}{
		{"past stop frame", 14, 1, false, 1, false, false, 0, 0, false},
		{"start", 5, 0, false, 1, true, true, 0, 0, false},
		{"mid run", 9, 1, false, 1, true, false, 0, 0, false},
		{"loop frame", 13, 1, false, 1, true, false, 6, 16 << 8, true},
		{"adjacent target", 9, 1, true, 0, true, false, 14, 8 << 8, true},
		{"restart bit", 9, 2, false, 0, true, false, 14, 8 << 8, true},
	}
	for _, c := range cases {
		e, st := DiabRunClient(p, c.frame, c.stage, c.near)
		if st != c.wantStage || e.Handled != c.handled || e.StartRun != c.run || e.SetSeq != c.setSeq ||
			e.SetTotal != c.setSeq || (c.setSeq && (e.Seq != c.seq || e.Total != c.total)) {
			t.Errorf("%s: %+v stage %d", c.name, e, st)
		}
	}
}

func TestMosquitoClient(t *testing.T) {
	if e, n := MosquitoClient(12, 3, false, true); e.Handled || n != 0 {
		t.Errorf("no target: %+v %d", e, n)
	}

	if e, n := MosquitoClient(12, 3, true, false); e.Handled || n != 3 {
		t.Errorf("out of reach: %+v %d", e, n)
	}

	if e, n := MosquitoClient(12, 1, true, true); e.Handled || n != 1 {
		t.Errorf("last bite: %+v %d", e, n)
	}

	e, n := MosquitoClient(12, 3, true, true)
	if !e.Handled || e.Seq != 12 || !e.TotalLow || e.Total != 0x100 || n != 2 {
		t.Errorf("loop: %+v %d", e, n)
	}
}

func TestQueenDeathClient(t *testing.T) {
	if e, _ := QueenDeathClient(6, 0); e.Seq != 7 || e.Total != 0x3200 {
		t.Errorf("frame 6: %+v", e)
	}

	c := 0

	for i := 0; i < 4; i++ {
		var e FrameEdit

		e, c = QueenDeathClient(0x16, c)
		if e.SetSeq {
			t.Fatalf("pass %d jumped early", i)
		}
	}

	if e, c2 := QueenDeathClient(0x16, c); !e.SetSeq || e.Seq != 0x28 || e.Total != 0x1100 || c2 != c {
		t.Errorf("after loops: %+v", e)
	}

	if e, _ := QueenDeathClient(0x1e, 9); !e.ResetMode || !e.Flag10000 {
		t.Errorf("end: %+v", e)
	}

	if e, _ := QueenDeathClient(3, 0); !e.Handled || e.SetSeq {
		t.Errorf("idle frame: %+v", e)
	}
}

func TestStompShake(t *testing.T) {
	s := StompShake(8, 5, 20, 15) // skills.txt row 301
	if s.Amp != 8 || s.RampMs != 200 || s.HoldMs != 800 || s.DecayMs != 600 {
		t.Fatalf("%+v", s)
	}

	cases := []struct{ ms, want int }{{0, 0}, {100, 4}, {200, 8}, {900, 8}, {1000, 8}, {1300, 4}, {1600, 0}, {1601, 0}}
	for _, c := range cases {
		if got := s.Magnitude(c.ms); got != c.want {
			t.Errorf("t=%d: %d, want %d", c.ms, got, c.want)
		}
	}

	if (Shake{Amp: 5, RampMs: 10, DecayMs: 10}).Active(5) {
		t.Error("zero hold disables the shake")
	}
}
