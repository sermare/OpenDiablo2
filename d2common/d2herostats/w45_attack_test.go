package d2herostats

import "testing"

// TestAttackFramesExe pins AttackFrames to the exe rule (PATH_UpdateUnitVelocityAndAnimRate 0x624150): percent
// 100 - wsm + skill + floor(120*ias/(120+ias)) clamped to 15..175, rate floor(256*percent/100), game frames
// ceil(256*animFrames/rate) and at least 2.
func TestAttackFramesExe(t *testing.T) {
	cases := []struct {
		name                 string
		frames, ias, wsm, sk int
		want                 int
	}{
		{"plain", 16, 0, 0, 0, 16},
		{"60 ias gives 40 percent: rate 358", 16, 60, 0, 0, 12},
		{"slow weapon 80 percent: rate 204", 16, 0, 20, 0, 21},
		{"fast weapon 130 percent: rate 332", 16, 0, -30, 0, 13},
		{"upper clamp 175 percent: rate 448", 16, 1000, 0, 0, 10},
		{"skill term", 16, 0, 0, 30, 13},
		{"lower clamp 15 percent: rate 38", 16, 0, 200, 0, 108},
		{"never below 2", 1, 0, 0, 0, 2},
	}

	for _, c := range cases {
		if got := AttackFrames(c.frames, c.ias, c.wsm, c.sk); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}
