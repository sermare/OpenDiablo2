package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Rescue on Mount Arreat (A5Q2): the barbarian prisoners stand in cages (objects.txt 473, "cagedwussie1", a DS1
// object of the Arreat levels). The prison door the quest wants killed is a monster of class 434 (monstats
// "prisondoor", class 0x1b2 in the exe's quest handler, which spawns it by class when the quest is on; monpreset.txt has
// no prisondoor row, so no DS1 monster marker can place it). The generated level therefore had the cages but no door,
// and the kill step could not be played. The door is put on each cage when the level is built.
// UNVERIFIED: the exact spot of the door relative to the cage (the cage's own sub-tile is used).
const (
	objCagedWussie   = 473
	prisonDoorMonKey = "prisondoor"
)

// placePrisonDoors puts one prison door monster on every barbarian cage of the level, once per level build, unless the
// quest was already handed in.
func (v *Game) placePrisonDoors() {
	if v.prisonDoors == v.levels.changes+1 || v.monsters == nil || v.gameClient == nil {
		return
	}

	v.prisonDoors = v.levels.changes + 1

	if r := v.quests(); r != nil {
		if q := r.g.Quest(d2quest.QuestRescue); q != nil && r.g.Rec.Get(q.Slot, d2quest.FlagRewardGranted) {
			return
		}
	}

	stat := v.monsters.FindStat(prisonDoorMonKey)
	if stat == nil {
		return
	}

	placed := 0

	for _, e := range v.gameClient.MapEngine.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok || ob.Record().Index != objCagedWussie {
			continue
		}

		x, y := ob.GetPositionF()
		sx, sy := int(x*subtiles), int(y*subtiles)

		if _, err := v.monsters.SpawnNear(stat, sx, sy, 2); err != nil {
			v.Warningf("PRISONDOOR could not place a door at the cage (%.1f,%.1f): %v", x, y, err)
			continue
		}

		placed++
	}

	if placed > 0 {
		v.Infof("PRISONDOOR placed %d prison door(s) on the barbarian cages of level %d", placed, v.currentLevel())
	}
}
