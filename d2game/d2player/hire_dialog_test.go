package d2player

import (
	"image/color"
	"testing"
)

func fakeTr(k string) string {
	return map[string]string{
		"ItemDesc1s": "Your Gold: %d     Hire which Mercenary?", "ItemDesc1t": "There are no Mercenaries left to hire.",
		"HP": "Life", "AC": "Def", "Level": "Lvl", "Cost": "Cost", "strhirespecial1": "Fire Arrow",
	}[k]
}

func TestHireDialogText(t *testing.T) {
	if got, want := HireDialogTitle(fakeTr, 1234), "Your Gold: 1234     Hire which Mercenary?"; got != want {
		t.Errorf("title %q, want %q", got, want)
	}

	if got, want := HireDialogLine(fakeTr, "Aliza", 3, 61, 9, 1500), "Aliza - Lvl:  3  Life: 61  Def:  9  Cost: 1500"; got != want {
		t.Errorf("line %q, want %q", got, want)
	}

	if got := HireDialogDesc(fakeTr, "farw"); got != "Fire Arrow" {
		t.Errorf("desc %q", got)
	}

	if got := HireDialogDesc(fakeTr, ""); got != "" {
		t.Errorf("empty desc %q", got)
	}

	if HireDialogEmpty(fakeTr) == "" {
		t.Error("no empty text")
	}
}

func TestReviveLine(t *testing.T) {
	if got := ReviveLine("Resurrect %s: %d", "Aliza", 450); got != "Resurrect Aliza: 450" {
		t.Errorf("got %q", got)
	}

	if got := ReviveLine("hireresurrect2", "Aliza", 450); got != "Resurrect Aliza: 450" {
		t.Errorf("fallback %q", got)
	}
}

func TestMercPortraitBar(t *testing.T) {
	for _, tc := range []struct {
		hp, max, fill int
		c             color.RGBA
		drawn         bool
	}{
		{0, 100, 0, color.RGBA{}, false},
		{100, 100, 46, mercBarGreen, true},
		{250, 100, 46, mercBarGreen, true}, // clamped
		{50, 100, 23, mercBarGreen, true},
		{49, 100, 22, mercBarYellow, true},
		{25, 100, 11, mercBarYellow, true},
		{24, 100, 11, mercBarRed, true},
		{1, 1000, 0, color.RGBA{}, false}, // under one percent: 0 percent, no bar
	} {
		fill, c, ok := MercPortraitBar(tc.hp, tc.max)
		if fill != tc.fill || c != tc.c || ok != tc.drawn {
			t.Errorf("bar(%d/%d) = %d %v %v, want %d %v %v", tc.hp, tc.max, fill, c, ok, tc.fill, tc.c, tc.drawn)
		}
	}
}

func TestMercPortraitHitAndIcon(t *testing.T) {
	if !MercPortraitHit(15, 14) || !MercPortraitHit(60, 59) || MercPortraitHit(61, 30) || MercPortraitHit(30, 60) || MercPortraitHit(14, 30) {
		t.Error("portrait hit rectangle")
	}

	for class, want := range map[int]string{271: "rogueicon", 338: "act2hireableicon", 359: "act3hireableicon", 561: "barbhirable_icon", 560: "", 1: ""} {
		if got := MercPortraitIcon(class); got != want {
			t.Errorf("icon(%d) = %q, want %q", class, got, want)
		}
	}

	tip := MercPortraitTooltip("Right-click to Open Inventory (%s)", "Drop Potion on Portrait to Heal", "O")
	if len(tip) != 2 || tip[0] != "Right-click to Open Inventory (O)" {
		t.Errorf("tooltip %v", tip)
	}
}
