package d2lightmap

import "testing"

func TestApproxDistance(t *testing.T) {
	for _, tc := range []struct{ dx, dy, want int }{
		{0, 0, 0}, {1024, 0, 0x3d7}, {0, -1024, 0x3d7}, {1024, 1024, (1024*0x3d7 + 1024*0x197) >> 10},
		{-8, 3, (8*0x3d7 + 3*0x197) >> 10},
	} {
		if got := ApproxDistance(tc.dx, tc.dy); got != tc.want {
			t.Errorf("ApproxDistance(%d,%d)=%d want %d", tc.dx, tc.dy, got, tc.want)
		}
	}
}

func TestRebuildBaseOnly(t *testing.T) {
	m := New()
	base := Cell{Intensity: 100, R: 1, G: 2, B: 3}
	m.Rebuild(500, 300, base, nil)

	if m.OriginX != 476 || m.OriginY != 276 {
		t.Fatalf("origin %d,%d", m.OriginX, m.OriginY)
	}

	if got := m.Sample(500.5, 300.2); got != base {
		t.Errorf("sample %v", got)
	}

	// outside the map clamps to the border cell
	if got := m.Sample(-1000, 99999); got != base {
		t.Errorf("clamped sample %v", got)
	}
}

func TestRadialFalloff(t *testing.T) {
	m := New()
	s := NewSource(10, 200, 255, 255, 255)
	s.SetPosition(100, 100)
	m.Rebuild(100, 100, Cell{}, []*Source{s})

	centre := m.Sample(100, 100)
	near := m.Sample(103, 100)
	far := m.Sample(108, 100)
	out := m.Sample(112, 100)

	if centre.Intensity < 190 || centre.Intensity > 200 {
		t.Errorf("centre %d, want about the source intensity", centre.Intensity)
	}

	if !(centre.Intensity > near.Intensity && near.Intensity > far.Intensity && far.Intensity > 0) {
		t.Errorf("not monotonic: %d %d %d", centre.Intensity, near.Intensity, far.Intensity)
	}

	if out.Intensity != 0 {
		t.Errorf("outside radius lit: %d", out.Intensity)
	}

	if centre.R < 250 || centre.B < 250 {
		t.Errorf("colour %v", centre)
	}
}

func TestColourBlendIsIntensityWeighted(t *testing.T) {
	m := New()
	// dark blue ambient + bright white light: the lit cell must lean to white
	base := Cell{Intensity: 40, R: 0, G: 0, B: 255}
	s := NewSource(8, 200, 255, 255, 255)
	s.SetPosition(50, 50)
	m.Rebuild(50, 50, base, []*Source{s})

	c := m.Sample(50, 50)
	if c.R < 150 || c.Intensity < 230 {
		t.Errorf("centre %v did not blend toward white", c)
	}

	edge := m.Sample(50+7.5, 50)
	if edge.R >= c.R {
		t.Errorf("edge should be bluer than the centre: %v vs %v", edge, c)
	}
}

func TestIntensitySaturates(t *testing.T) {
	m := New()
	a := NewSource(8, 200, 255, 255, 255)
	b := NewSource(8, 200, 255, 255, 255)
	a.SetPosition(10, 10)
	b.SetPosition(10, 10)
	m.Rebuild(10, 10, Cell{Intensity: 100, R: 255, G: 255, B: 255}, []*Source{a, b})

	if got := m.Sample(10, 10).Intensity; got != 255 {
		t.Errorf("intensity %d, want 255", got)
	}
}

func TestSourceRamp(t *testing.T) {
	s := NewSource(2, 255, 255, 255, 255)
	s.SetTargetRadius(5)

	steps := 0
	for s.Radius != s.Target {
		s.Step()

		steps++
		if steps > 100 {
			t.Fatal("never reached target")
		}
	}

	if steps != (40-16)/8 {
		t.Errorf("steps %d", steps)
	}

	s.SetTargetRadius(100)
	if s.Target != MaxRadius {
		t.Errorf("target not capped: %d", s.Target)
	}
}
