package d2asset

import "testing"

func TestRateFromFrameSeconds(t *testing.T) {
	for _, tc := range []struct {
		seconds float64
		want    int
	}{
		{0, 0}, {-1, 0},
		{1.0 / (256 * speedUnit), 256}, // speed 256: one frame per tick
		{1.0 / (128 * speedUnit), 128},
		{1.0 / (100 * speedUnit), 100},
	} {
		if got := rateFromFrameSeconds(tc.seconds); got != tc.want {
			t.Errorf("%v: got %d want %d", tc.seconds, got, tc.want)
		}
	}
}

func TestTickStepper(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rate    int
		seconds float64
		want    int
	}{
		{"one frame per tick", 256, 1, 25},
		{"half rate", 128, 1, 12}, // 25 ticks * 0.5 = 12.5 -> 12 whole frames
		{"half rate 2 s", 128, 2, 25},
		{"speed 100", 100, 10.24, 250}, // 256 ticks * 100 / 256 = 100 per ... 10.24 s = 256 ticks
		{"double rate", 512, 1, 50},
		{"no rate", 0, 1, 0},
	} {
		s := tickStepper{rate: tc.rate}
		got := 0

		// feed in 60 fps slices
		for t0 := 0.0; t0 < tc.seconds-1e-9; t0 += 1.0 / 60 {
			step := 1.0 / 60
			if tc.seconds-t0 < step {
				step = tc.seconds - t0
			}

			got += s.advance(step)
		}

		if tc.name == "speed 100" {
			tc.want = 256 * 100 / 256 // 100 frames
		}

		if got < tc.want-2 || got > tc.want+2 {
			t.Errorf("%s: got %d frames want %d", tc.name, got, tc.want)
		}
	}
}

func TestTickStepperOnlyChangesOnTickBoundaries(t *testing.T) {
	s := tickStepper{rate: 256}

	if f := s.advance(0.039); f != 0 { // less than one 40 ms tick
		t.Errorf("advanced %d frames before a tick", f)
	}

	if f := s.advance(0.002); f != 1 {
		t.Errorf("want 1 frame after the tick completes, got %d", f)
	}
}

func TestTickStepperAccumulatorCarries(t *testing.T) {
	s := tickStepper{rate: 100}
	frames := 0

	for i := 0; i < 256; i++ { // 256 ticks -> exactly 100 frames
		frames += s.advance(1.0 / engineTickRate)
	}

	if frames != 100 {
		t.Errorf("got %d frames want 100", frames)
	}
}
