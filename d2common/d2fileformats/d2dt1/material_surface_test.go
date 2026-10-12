package d2dt1

import "testing"

func TestFootstepSurface(t *testing.T) {
	for _, c := range []struct {
		flags uint16
		want  int
	}{
		{0x20, 1}, {0x08, 2}, {0x10, 3}, {0x40, 4}, {0x400, 5}, {0x80, 6}, {0x01, 0}, {0x28, 1}, {0x480, 6}, {0, 0},
	} {
		if got := NewMaterialFlags(c.flags).FootstepSurface(); got != c.want {
			t.Errorf("%#x: %d want %d", c.flags, got, c.want)
		}
	}
}
