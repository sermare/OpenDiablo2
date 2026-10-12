package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// An antidote given to the merc adds 50 poison resist for 750 frames; a second one extends the time and does
// not add the resist twice (0x5bc990); the state ends on time.
func TestMercAntidoteResist(t *testing.T) {
	d, owner, u := gearDirector(t)
	anti := d2inventory.StatePotion{State: "antidote", Frames: 750,
		Mods: []d2inventory.StateMod{{Stat: "poisonresist", Value: 50}, {Stat: "maxpoisonresist", Value: 10}}}

	base := d.MercReduce(owner, MercPoison, 100)

	d.frame = 100

	if !d.DrinkMercState(owner, anti) {
		t.Fatal("the antidote was refused")
	}

	with := d.MercReduce(owner, MercPoison, 100)
	if with >= base || with != base-50 {
		t.Errorf("poison damage %d with the antidote, %d without: want exactly 50 less", with, base)
	}

	d.frame = 350
	d.DrinkMercState(owner, anti)

	if got := u.merc.potions["antidote"].until; got != 1600 {
		t.Errorf("second antidote until=%d, want 1600", got)
	}

	if again := d.MercReduce(owner, MercPoison, 100); again != with {
		t.Errorf("two antidotes must not stack resist: %d vs %d", again, with)
	}

	d.frame = 1600
	if after := d.MercReduce(owner, MercPoison, 100); after != base {
		t.Errorf("after the state ended: %d, want %d", after, base)
	}

	if d.DrinkMercState(&d2mapentity.Player{}, anti) {
		t.Error("a missing merc takes nothing")
	}
}
