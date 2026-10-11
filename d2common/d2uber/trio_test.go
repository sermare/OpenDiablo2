package d2uber

import "testing"

func TestTrio(t *testing.T) {
	var tr Trio

	for _, c := range []int{TrioClassC, 5, TrioClassA, TrioClassA} {
		if _, ok := tr.Kill(c, 3); ok {
			t.Fatalf("class %#x must not complete the trio", c)
		}
	}

	d, ok := tr.Kill(TrioClassB, 3)
	if !ok || d.Charm != "cm2" || !d.MaxCharges || d.StandardNum != 3 {
		t.Fatalf("%+v %v", d, ok)
	}

	// the flags stay set: another kill of a trio class drops again (as the exe)
	if _, ok := tr.Kill(TrioClassA, 0); !ok {
		t.Fatal("flags persist")
	}
}

func TestTrioPlayerCounter(t *testing.T) {
	tests := []struct{ players, want int }{{-2, 0}, {0, 0}, {1, 1}, {8, 8}}
	for _, tc := range tests {
		tr := Trio{a: true, b: true}
		d, _ := tr.Kill(TrioClassC, tc.players)

		if d.StandardNum != tc.want {
			t.Errorf("players %d: %d", tc.players, d.StandardNum)
		}
	}
}
