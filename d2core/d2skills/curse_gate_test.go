package d2skills

import "testing"

func TestCurseDoFunc(t *testing.T) {
	for n, want := range map[int]bool{30: true, 59: true, 61: true, 0: false, 65: false, 56: false} {
		if curseDoFunc(n) != want {
			t.Errorf("do %d: %v", n, !want)
		}
	}
}
