package d2hero

import (
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestExportDeathStatus(t *testing.T) {
	data, err := d2s.NewCharacter("Mortal", d2s.Barbarian,
		d2s.NewCharacterFlags{Hardcore: true, Created: time.Unix(1700000000, 0)}, d2s.DefaultAppearance(d2s.Barbarian))
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(data, nil)
	if err != nil {
		t.Fatal(err)
	}

	state := &HeroState{Death: &DeathState{Died: true, Deaths: 1}}
	exportDeath(c, state)

	out, err := d2s.Write(c, nil)
	if err != nil {
		t.Fatal(err)
	}

	h, err := d2s.ParseHeader(out)
	if err != nil {
		t.Fatal(err)
	}

	if !h.IsDead() || !h.IsHardcore() || !h.IsNewCharacter() {
		t.Fatalf("status %#x", h.Status)
	}

	state.Death.Died = false
	exportDeath(c, state)

	if c.Header.Status&d2s.StatusDied != 0 {
		t.Fatal("the died flag must clear after a softcore respawn")
	}

	exportDeath(c, &HeroState{}) // no death record: untouched
}
