package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// creditOwnerMerc gives the owner's merc XP for a kill. When that XP levels the
// merc up, the level and the gear must land on the MERC's unit, never on the
// dead monster that was killed.
func TestCreditOwnerMercLevelsTheMercNotTheVictim(t *testing.T) {
	d, owner, mercUnit := gearDirector(t)

	victim := &unit{m: &d2mapentity.Monster{}, b: &d2monster.Brain{}}
	victim.m.Vitals = d2mapentity.MonsterVitals{Level: 3, MaxHP: 10, HP: 0}

	if mercUnit.m.Vitals.Level != 3 {
		t.Fatalf("setup: merc level %d", mercUnit.m.Vitals.Level)
	}

	d.creditOwnerMerc(owner, victim, 40000)

	if mercUnit.merc.level <= 3 {
		t.Fatalf("setup: the XP should have levelled the merc up, level %d", mercUnit.merc.level)
	}

	if victim.m.Vitals.Level != 3 {
		t.Errorf("the dead monster's level changed to %d: the level-up was applied to the victim", victim.m.Vitals.Level)
	}

	if mercUnit.m.Vitals.Level != mercUnit.merc.level {
		t.Errorf("merc unit level %d, merc state level %d: the merc unit was not updated",
			mercUnit.m.Vitals.Level, mercUnit.merc.level)
	}
}
