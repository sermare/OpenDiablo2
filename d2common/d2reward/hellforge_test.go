package d2reward

import (
	"reflect"
	"testing"
)

func TestHellforgeTables(t *testing.T) {
	tests := []struct {
		stage int
		first string
		last  string
		nil   bool
	}{
		{0, "", "", true},
		{1, "gsv", "sku", false},
		{2, "gzv", "skl", false},
		{3, "gzv", "skl", false},
		{4, "gpv", "skz", false},
		{5, "", "", true},
	}

	for _, tc := range tests {
		got := HellforgeTable(tc.stage)
		if tc.nil {
			if got != nil {
				t.Errorf("stage %d: %v", tc.stage, got)
			}

			continue
		}

		if len(got) != 7 || got[0] != tc.first || got[6] != tc.last {
			t.Errorf("stage %d: %v", tc.stage, got)
		}
	}
}

func TestHellforgeRunes(t *testing.T) {
	tests := []struct {
		diff        int
		first, last string
	}{{0, "r01", "r11"}, {1, "r12", "r22"}, {2, "r15", "r25"}, {7, "r01", "r11"}}
	for _, tc := range tests {
		r := HellforgeRunes(tc.diff)
		if len(r) != 11 || r[0] != tc.first || r[10] != tc.last {
			t.Errorf("difficulty %d: %v", tc.diff, r)
		}
	}
}

func TestShowerFullRun(t *testing.T) {
	s := NewShower(2, true, 1)
	step := func() uint32 { return 9 } // 9 % 7 = 2: the third entry

	var dropped []string

	drop := func(c string) bool { dropped = append(dropped, c); return true }
	roll := func(n int32) uint32 { return uint32(n) - 1 } // last rune

	wantStages := []string{"gpb", "glb", "glb", "gsb"}
	for i, w := range wantStages {
		dropped = nil
		res := s.Pulse(step, roll, drop)

		if len(dropped) < 2 || !reflect.DeepEqual(dropped[:2], []string{w, w}) {
			t.Fatalf("pulse %d dropped %v want two %s", i, dropped, w)
		}

		if i < 3 && (res.Next != HellforgeDelayFrames || res.Rune != "") {
			t.Fatalf("pulse %d: %+v", i, res)
		}
	}

	if s.Active || s.Stage != 0 {
		t.Fatalf("shower must be over: %+v", s)
	}

	if len(dropped) != 3 || dropped[2] != "r22" {
		t.Fatalf("the last pulse also drops the nightmare rune: %v", dropped)
	}

	if res := s.Pulse(step, roll, drop); len(res.Dropped) != 0 {
		t.Fatal("a finished shower stays quiet")
	}
}

func TestShowerNoRuneInClassic(t *testing.T) {
	s := NewShower(1, false, 0)
	s.Stage = 1
	n := 0
	res := s.Pulse(func() uint32 { return 0 }, func(int32) uint32 { return 0 }, func(string) bool { n++; return true })

	if n != 1 || res.Rune != "" || s.Active {
		t.Fatalf("%d %+v %+v", n, res, s)
	}
}

func TestShowerStopsWhenNothingDrops(t *testing.T) {
	s := NewShower(3, true, 0)
	res := s.Pulse(func() uint32 { return 0 }, func(int32) uint32 { return 0 }, func(string) bool { return false })

	if len(res.Dropped) != 0 || res.Next != 0 || s.Stage != HellforgeStartStage || !s.Active {
		t.Fatalf("%+v %+v", res, s)
	}
}

func TestShowerBadStage(t *testing.T) {
	s := &Shower{Active: true, Stage: 9, Count: 2}
	n := 0
	s.Pulse(func() uint32 { return 0 }, nil, func(string) bool { n++; return true })

	if n != 0 || s.Stage != 9 {
		t.Fatal("unknown stage leaves everything alone")
	}
}
