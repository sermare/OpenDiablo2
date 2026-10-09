package d2object

import "testing"

func TestPadPartner(t *testing.T) {
	pads := []Pos{{10, 10}, {20, 10}, {12, 11}, {200, 200}}

	for _, c := range []struct {
		name string
		self Pos
		want Pos
		ok   bool
	}{
		{"nearest other pad, not itself", Pos{10, 10}, Pos{12, 11}, true},
		{"the far pad is out of reach", Pos{200, 200}, Pos{}, false},
		{"second pad", Pos{20, 10}, Pos{12, 11}, true},
	} {
		got, ok := PadPartner(c.self, pads)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got %v %v, want %v %v", c.name, got, ok, c.want, c.ok)
		}
	}

	if !Lookup(FnTeleportPad).Operable() || !Lookup(FnArcanePortal).Operable() || !Lookup(FnDurielPortal).Operable() {
		t.Error("the Act 2 teleport objects must be operable")
	}

	if Lookup(47).Operable() {
		t.Error("stairs (fn 47) are not operated by click")
	}
}
