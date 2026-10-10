package d2player

import "testing"

func TestModePanelGeometry(t *testing.T) {
	cases := []struct {
		m                Mode
		left, right, top int
	}{
		{Mode800, 80, 400, 60},
		{Mode640, 0, 320, 0},
	}

	for _, c := range cases {
		if c.m.LeftPanelX() != c.left || c.m.RightPanelX() != c.right || c.m.PanelTop() != c.top {
			t.Errorf("%v: left %d right %d top %d", c.m, c.m.LeftPanelX(), c.m.RightPanelX(), c.m.PanelTop())
		}
	}

	if p := Mode800.PanelPictures(Mode800.RightPanelX()); p[0].X != 400 || p[0].Y != 60 || p[3].Y != 316 {
		t.Errorf("800 pictures %v", p)
	}
}

func TestHUDRects(t *testing.T) {
	want := map[string]UIRect{
		"left_skill.hit":  {"hud", "left_skill.hit", 117, 552, 49, 49},
		"mana.hit":        {"hud", "mana.hit", 689, 525, 81, 61},
		"experience.hit":  {"hud", "experience.hit", 254, 557, 124, 10},
		"stat_button.hit": {"hud", "stat_button.hit", 207, 559, 33, 33},
		"run_button":      {"hud", "run_button", 255, 570, 16, 20},
	}

	got := map[string]UIRect{}
	for _, r := range Mode800.HUDRects() {
		got[r.Name] = r
	}

	for n, w := range want {
		if got[n] != w {
			t.Errorf("%s: got %v want %v", n, got[n], w)
		}
	}

	for _, r := range Mode640.HUDRects() {
		if r.Name == "mana.hit" && (r.X != 529 || r.Y != 405) {
			t.Errorf("640 mana.hit %v", r)
		}
	}
}

func TestMiniPanelRects(t *testing.T) {
	rs := Mode800.MiniPanelRects(true, 0)
	if rs[0].X != 323 || rs[1].X != 326 || rs[7].X != 452 || rs[1].Y != 530 {
		t.Errorf("single player strip %v", rs)
	}

	if rs := Mode800.MiniPanelRects(false, -1); rs[1].X != 198 {
		t.Errorf("multiplayer strip with a panel on the right: %v", rs[1])
	}
}
