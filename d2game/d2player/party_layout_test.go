package d2player

import "testing"

func TestPartyLayout(t *testing.T) {
	inv, rel, b1, listen, see := Mode800.PartyRowX()

	for _, c := range []struct {
		name      string
		got, want int
	}{
		{"invite", inv, 270}, {"relation", rel, 94}, {"bit01", b1, 325}, {"listen", listen, 346}, {"see", see, 367},
		{"go invite", inviteAcceptButtonX, 270}, {"go listen", listeningSwitcherX, 346}, {"go see", seeingSwitcherX, 367},
		{"go relation", relationshipSwitcherX, 94},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %d want %d", c.name, c.got, c.want)
		}
	}

	if r := Mode800.PartyCloseRect(); r.X != 358 || r.Y != 449 || r.W != 32 || r.H != 32 {
		t.Errorf("close rect %v", r)
	}

	if partyPanelCloseButtonX != 358 || partyPanelCloseButtonY != 449 {
		t.Errorf("panel close button not on the layout rect")
	}

	if r := Mode640.PartyCloseRect(); r.X != 0x116 || r.Y != 0x1a5-32 {
		t.Errorf("640 close rect %v", r)
	}
}

func TestPartyRowGeometry(t *testing.T) {
	// row bottom: 0x5a + 0x26*i panel relative (UI_RebuildPartyList EDI), screen = + 60 at 800x600, widgets 20 high
	exp := map[string]UIRect{
		"invite":     {"party", "invite", 270, 130, 0x35, 20},
		"hostile":    {"party", "hostile", 94, 137, 0x14, 20},
		"bit01":      {"party", "bit01", 325, 130, 0x14, 20},
		"listen":     {"party", "listen", 346, 130, 0x14, 20},
		"mute":       {"party", "mute", 367, 130, 0x14, 20},
		"name_hover": {"party", "name_hover", 116, 150 + 0xd - 0x1a, 0x8c - 0x24, 0x1a},
	}

	for _, g := range Mode800.PartyRowRects(0) {
		if g != exp[g.Name] {
			t.Errorf("row 0 %s: got %v want %v", g.Name, g, exp[g.Name])
		}
	}

	if got := PartyRowBottom(7); got != 0x5a+0x26*7 {
		t.Errorf("row 7 bottom %d", got)
	}

	if r := Mode800.PartyRowRects(3); r[0].Y != 130+3*38 || r[1].Y != 137+3*38 {
		t.Errorf("row 3: %v", r)
	}

	if r := Mode640.PartyRowRects(0); r[0].X != 0xbe || r[0].Y != 0x5a-20 {
		t.Errorf("640 row 0: %v", r[0])
	}

	if h := Mode800.PartyHeaderHover(); h.X != 80+0xe3 || h.Y != 60+0x18 || h.W != 0x118-0xe3 || h.H != 0xb {
		t.Errorf("header hover %v", h)
	}

	// the Go widgets stand on the original's tops
	if baseInviteAcceptButtonY != 130 || baseListeningSwitcherY != 130 || baseSeeingSwitcherY != 130 || baseRelationshipSwitcherY != 137 {
		t.Errorf("go widget tops: %d %d %d %d", baseInviteAcceptButtonY, baseListeningSwitcherY, baseSeeingSwitcherY, baseRelationshipSwitcherY)
	}
}
