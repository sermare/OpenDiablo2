package d2gamescreen

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// commandSpawnRank spawns a champion pack, a unique pack or a super unique
// (by its SuperUniques.txt key) beside the hero and logs its type flags and
// modifiers (MONSTER rank ...), for the drop tests.
func (v *Game) commandSpawnRank(args []string) error {
	d := v.monsterDirector()
	if d == nil {
		return errors.New("no monsters yet")
	}

	centre := d2path.Point{X: int(v.localPlayer.Position.X()) + monsterTestRing, Y: int(v.localPlayer.Position.Y())}

	var (
		res *d2monsters.PackResult
		err error
	)

	switch args[0] {
	case "super":
		res, err = d.SpawnSuperUnique(args[1], centre)
	case "champion", "unique":
		stat := d.FindStat(args[1])
		if stat == nil {
			return fmt.Errorf("no monster %q", args[1])
		}

		if args[0] == "champion" {
			res, err = d.SpawnChampionGroup(stat, centre)
		} else {
			res, err = d.SpawnUniqueGroup(stat, centre)
		}
	default:
		return fmt.Errorf("spawnrank: unknown kind %q", args[0])
	}

	if err != nil {
		return err
	}

	v.rankLeader = res.Leader

	for _, m := range res.Monsters {
		v.Infof("MONSTER rank kind=%s name=%s type_flags=%#x super_unique=%q hcidx=%d mods=%v", args[0], m.Label(), m.TypeFlags,
			m.SuperUnique, m.SuperUniqueIdx, m.Modifiers)
	}

	return nil
}

// commandKillLeader kills the leader of the last spawnrank pack as the hero.
func (v *Game) commandKillLeader(_ []string) error {
	d := v.monsterDirector()
	if d == nil || v.rankLeader == nil {
		return errors.New("no pack leader")
	}

	v.Infof("KILLLEADER %s", v.rankLeader.Label())
	d.Damage(v.rankLeader, 1<<20, v.localPlayer)

	return nil
}
