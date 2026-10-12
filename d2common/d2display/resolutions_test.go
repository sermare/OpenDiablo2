package d2display

import "testing"

func TestResolutionChoices(t *testing.T) {
	tests := []struct {
		name        string
		mon, cur    Size
		first, last Size
		want        Size
	}{
		{"macbook", Size{1512, 982}, Size{1512, 982}, Size{800, 600}, Size{1512, 982}, Size{1280, 720}},
		{"odd current added", Size{1920, 1080}, Size{1000, 700}, Size{800, 600}, Size{1920, 1080}, Size{1000, 700}},
		{"unknown monitor", Size{}, Size{1512, 982}, Size{800, 600}, Size{3440, 1440}, Size{1512, 982}},
	}

	for _, tc := range tests {
		got := ResolutionChoices(tc.mon, tc.cur)
		if got[0] != tc.first || got[len(got)-1] != tc.last {
			t.Errorf("%s: got %v", tc.name, got)
		}

		found := false
		for _, s := range got {
			found = found || s == tc.want
		}

		if !found {
			t.Errorf("%s: %v lacks %v", tc.name, got, tc.want)
		}
	}

	if s, err := ParseRow(Size{1280, 720}.String()); err != nil || s != (Size{1280, 720}) {
		t.Errorf("row round trip: %v %v", s, err)
	}
}
