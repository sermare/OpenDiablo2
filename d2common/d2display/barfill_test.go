package d2display

import "testing"

func TestParseBarFill(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want BarFill
		ok   bool
	}{
		{"", BarFillNone, true}, {"none", BarFillNone, true}, {" Black ", BarFillBlack, true},
		{"TILE", BarFillTile, true}, {"stretch", BarFillNone, false},
	} {
		got, ok := ParseBarFill(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseBarFill(%q) = %v,%v want %v,%v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestBarFillSpans(t *testing.T) {
	for _, sz := range []Size{{800, 600}, {1512, 982}, {1920, 1080}, {2560, 1440}, {3440, 1440}, {5120, 1440}, {1001, 700}, {640, 480}} {
		for _, tw := range []int{1, 8, 13, 64} {
			spans := BarFillSpans(sz, tw)
			s := Clamp(sz)
			ox, _ := Origin(s, AnchorBottom)

			if ox == 0 {
				if len(spans) != 0 {
					t.Errorf("%v: spans at 800 wide", sz)
				}

				continue
			}

			// together with the bar the spans must cover [-ox, W-ox) exactly once
			cover := make([]int, s.W)

			for _, sp := range spans {
				if sp.W <= 0 || sp.W > tw || sp.SrcOff < 0 || sp.SrcOff+sp.W > tw {
					t.Fatalf("%v tile %d: bad span %+v", sz, tw, sp)
				}

				for x := sp.X; x < sp.X+sp.W; x++ {
					cover[x+ox]++
				}
			}

			for x := 0; x < s.W; x++ {
				want := 1
				if x >= ox && x < ox+BaseW {
					want = 0
				}

				if cover[x] != want {
					t.Fatalf("%v tile %d: column %d covered %d times, want %d", sz, tw, x-ox, cover[x], want)
				}
			}
		}
	}
}

func TestVideoRect(t *testing.T) {
	for _, tc := range []struct {
		name       string
		screen     Size
		vw, vh     int
		x, y, w, h int
	}{
		{"640x480 on 800x600 fills", Size{800, 600}, 640, 480, 0, 0, 800, 600},
		{"640x480 on 1920x1080 pillarbox", Size{1920, 1080}, 640, 480, 240, 0, 1440, 1080},
		{"640x480 on 1512x982", Size{1512, 982}, 640, 480, 101, 0, 1309, 982},
		{"640x480 on 5120x1440", Size{5120, 1440}, 640, 480, 1600, 0, 1920, 1440},
		{"wide video on a tall screen letterbox", Size{800, 1200}, 1600, 800, 0, 400, 800, 400},
		{"empty video", Size{800, 600}, 0, 480, 0, 0, 0, 0},
	} {
		x, y, w, h := VideoRect(tc.screen, tc.vw, tc.vh)
		if x != tc.x || y != tc.y || w != tc.w || h != tc.h {
			t.Errorf("%s: got %d,%d %dx%d want %d,%d %dx%d", tc.name, x, y, w, h, tc.x, tc.y, tc.w, tc.h)
		}

		if w > 0 && (x < 0 || y < 0 || x+w > tc.screen.W || y+h > tc.screen.H) {
			t.Errorf("%s: outside the screen", tc.name)
		}
	}
}
