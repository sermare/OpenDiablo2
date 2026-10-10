package d2automap

import "testing"

func TestClassify(t *testing.T) {
	both := Options{ShowParty: true, ShowNames: true}
	none := Options{}

	tests := []struct {
		name string
		u    Unit
		o    Options
		want Marker
	}{
		{"self", Unit{Kind: UnitPlayer, Self: true}, none, Marker{Cross: true, Color: ColorSelf}},
		{"self never labelled", Unit{Kind: UnitPlayer, Self: true}, both, Marker{Cross: true, Color: ColorSelf}},
		{"hostile player", Unit{Kind: UnitPlayer}, none, Marker{Cross: true, Color: ColorOther}},
		{"hostile player named", Unit{Kind: UnitPlayer}, both, Marker{Cross: true, Color: ColorOther, Label: LabelName}},
		{"names need party option", Unit{Kind: UnitPlayer}, Options{ShowNames: true}, Marker{Cross: true, Color: ColorOther}},
		{"party hidden when option off", Unit{Kind: UnitPlayer, Party: true}, none, Marker{Color: ColorParty}},
		{"party shown", Unit{Kind: UnitPlayer, Party: true}, Options{ShowParty: true}, Marker{Cross: true, Color: ColorParty}},
		{"party named", Unit{Kind: UnitPlayer, Party: true}, both, Marker{Cross: true, Color: ColorParty, Label: LabelName}},
		{"dead other player", Unit{Kind: UnitPlayer, Dead: true}, both, Marker{}},
		{"town npc", Unit{Kind: UnitMonster, FriendlyNPC: true}, none, Marker{Cross: true, Color: ColorNPC}},
		{"town npc name", Unit{Kind: UnitMonster, FriendlyNPC: true}, Options{ShowNames: true},
			Marker{Cross: true, Color: ColorNPC, Label: LabelName}},
		{"hostile monster", Unit{Kind: UnitMonster}, both, Marker{}},
		{"minion", Unit{Kind: UnitMonster, Minion: true}, none, Marker{Cross: true, Color: ColorMinion}},
		{"dead npc", Unit{Kind: UnitMonster, FriendlyNPC: true, Dead: true}, none, Marker{}},
		{"town portal", Unit{Kind: UnitObject, ObjectID: ObjectTownPortal}, none, Marker{Cross: true, Color: ColorPortal}},
		{"permanent portal", Unit{Kind: UnitObject, ObjectID: ObjectPermanentPortal, Level: 40}, none,
			Marker{Cross: true, Color: ColorPortal}},
		{"permanent portal hidden in 0x75", Unit{Kind: UnitObject, ObjectID: ObjectPermanentPortal, Level: 0x75}, none, Marker{}},
		{"stash cross", Unit{Kind: UnitObject, ObjectID: ObjectStash}, none, Marker{Cross: true, Color: ColorBlack}},
		{"stash text", Unit{Kind: UnitObject, ObjectID: ObjectStash}, Options{ShowNames: true},
			Marker{Color: ColorBlack, Label: LabelStash}},
		{"other object", Unit{Kind: UnitObject, ObjectID: 5}, both, Marker{}},
	}

	for _, tc := range tests {
		if got := Classify(tc.u, tc.o); got != tc.want {
			t.Errorf("%s: got %+v want %+v", tc.name, got, tc.want)
		}
	}
}

func TestCrossOffsetAndRoster(t *testing.T) {
	if dx, dy := CrossOffset(SizeMini); dx != -1 || dy != 5 {
		t.Errorf("mini cross offset %d,%d", dx, dy)
	}

	if dx, dy := CrossOffset(SizeFull); dx != 0 || dy != 0 {
		t.Errorf("full cross offset %d,%d", dx, dy)
	}

	if !RosterMarker(Options{ShowParty: true}, true) || RosterMarker(Options{ShowParty: true}, false) || RosterMarker(Options{}, true) {
		t.Error("roster marker rules")
	}
}

func TestCellTransparency(t *testing.T) {
	const w, h = 800, 600

	// hero screen centre (the engine's window centre) fades the most, far cells are normal
	tests := []struct {
		name         string
		fade         bool
		size         Size
		x, y, shiftX int
		want         Transparency
	}{
		{"fade off", false, SizeFull, 400, 300, 0, TransNormal},
		{"mini flat", true, SizeMini, 10, 10, 0, TransLight},
		{"centre", true, SizeFull, 400, 290, 0, TransVeryLight},
		{"middle ring", true, SizeFull, 400 + 40, 290 + 40, 0, TransLight},
		{"outer ring", true, SizeFull, 400 + 60, 290 + 70, 0, TransMedium},
		{"outside the box", true, SizeFull, 400 + 0x8d, 290, 0, TransNormal},
		{"far above", true, SizeFull, 400, 300 - 0x97, 0, TransNormal},
		{"panel shifts the box", true, SizeFull, 400 - w/4, 290, -w / 4, TransVeryLight},
		{"off the box without the shift", true, SizeFull, 400 - w/4, 290, 0, TransNormal},
	}

	for _, tc := range tests {
		if got := CellTransparency(tc.fade, tc.size, tc.x, tc.y, w, h, tc.shiftX); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}

	if TransNormal.Alpha() != 1 || TransVeryLight.Alpha() >= TransLight.Alpha() || TransLight.Alpha() >= TransMedium.Alpha() {
		t.Error("alpha must grow from very light to normal")
	}
}

func TestStorePersistsPerLevel(t *testing.T) {
	s := NewStore()

	a := s.Level(1)
	if !a.Add(LayerFloor, 8, 8, 100) {
		t.Fatal("add")
	}

	if s.Level(2).Count() != 0 {
		t.Error("another level must start empty")
	}

	// leaving and returning to level 1 gives the same model with its cells
	if s.Level(1) != a || s.Level(1).Count() != 1 {
		t.Error("revisited level lost its cells")
	}

	if !s.Has(2) || s.Has(3) || s.Levels() != 2 {
		t.Errorf("Has/Levels: %v %v %d", s.Has(2), s.Has(3), s.Levels())
	}

	s.Forget(1)

	if s.Level(1).Count() != 0 {
		t.Error("forgotten level must be empty")
	}
}

func TestMiniBoxOffsets(t *testing.T) {
	// 800x600 box at the right: d264 = 533, d260 = 0x4e (see docs/automap.md)
	d210, d214 := MiniBoxOffsets(800, 600, false)
	if d210 != 800/3-533-0x10 || d214 != 600/3-0x4e-0x10 {
		t.Errorf("right offsets %d,%d", d210, d214)
	}

	l210, l214 := MiniBoxOffsets(800, 600, true)
	if l210 != 800/3-0x10 || l214 != 600/3-0x10 {
		t.Errorf("left offsets %d,%d", l210, l214)
	}

	// stale offsets (centre option off) move the hero out of the centre of the box
	fresh := ComputeLayout(SizeMini, 800, 600, 40, 20, PanelNone, true)
	stale := ComputeLayoutOffsets(SizeMini, 800, 600, 40, 20, PanelNone, true, d210, d214)

	if fresh.OriginX == stale.OriginX && fresh.OriginY == stale.OriginY {
		t.Error("stale offsets must change the origin")
	}
}
