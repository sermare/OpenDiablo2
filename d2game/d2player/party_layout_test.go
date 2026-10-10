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
