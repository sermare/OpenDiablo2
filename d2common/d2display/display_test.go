package d2display

import "testing"

var targets = []Size{
	{800, 600}, {1512, 982}, {1920, 1080}, {2560, 1440}, {3440, 1440}, {5120, 1440}, {1980, 1800}, {640, 480},
}

func TestOriginGolden(t *testing.T) {
	want := map[Size][4]int{ // bottom x,y then centre x,y
		{800, 600}:   {0, 0, 0, 0},
		{1512, 982}:  {356, 382, 356, 191},
		{1920, 1080}: {560, 480, 560, 240},
		{2560, 1440}: {880, 840, 880, 420},
		{3440, 1440}: {1320, 840, 1320, 420},
		{5120, 1440}: {2160, 840, 2160, 420},
		{1980, 1800}: {590, 1200, 590, 600},
		{640, 480}:   {0, 0, 0, 0}, // clamped to 800x600
	}

	for _, s := range targets {
		bx, by := Origin(s, AnchorBottom)
		cx, cy := Origin(s, AnchorCenter)
		got := [4]int{bx, by, cx, cy}

		if got != want[s] {
			t.Errorf("%v: got %v want %v", s, got, want[s])
		}
	}
}

func TestLogical(t *testing.T) {
	cases := []struct {
		w, h, scale int
		want        Size
	}{
		{800, 600, 1, Size{800, 600}},
		{1600, 1200, 2, Size{800, 600}},
		{3840, 2160, 2, Size{1920, 1080}},
		{640, 480, 1, Size{800, 600}},
		{1512, 982, 1, Size{1512, 982}},
		{5120, 1440, 0, Size{5120, 1440}},
	}

	for _, c := range cases {
		if got := Logical(c.w, c.h, c.scale); got != c.want {
			t.Errorf("%v: got %v", c, got)
		}
	}
}

func TestResolve(t *testing.T) {
	s, _ := Resolve("", Size{}, Size{})
	if s != (Size{1512, 982}) {
		t.Errorf("default: %v", s)
	}

	s, _ = Resolve("", Size{}, Size{1440, 900})
	if s != (Size{1440, 900}) {
		t.Errorf("screen clamp: %v", s)
	}

	s, _ = Resolve("3440x1440", Size{1000, 700}, Size{1440, 900})
	if s != (Size{3440, 1440}) {
		t.Errorf("override: %v", s)
	}

	s, _ = Resolve("", Size{1920, 1080}, Size{})
	if s != (Size{1920, 1080}) {
		t.Errorf("config: %v", s)
	}

	if _, err := Resolve("bogus", Size{}, Size{}); err == nil {
		t.Error("bogus override must report an error")
	}
}

func TestColumnRoundTrip(t *testing.T) {
	Set(Size{1920, 1080})
	SetAnchor(AnchorBottom)

	defer func() { Set(Base); SetAnchor(AnchorBottom) }()

	x, y := ToColumn(960, 1050)
	if x != 400 || y != 570 {
		t.Errorf("column %d,%d", x, y)
	}

	if sx, sy := ToScreen(x, y); sx != 960 || sy != 1050 {
		t.Errorf("round trip %d,%d", sx, sy)
	}
}
