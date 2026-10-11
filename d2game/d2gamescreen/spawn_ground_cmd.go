package d2gamescreen

import (
	"fmt"
	"math"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// spawnGroundDefaultRing is the index of the free cell (nearest first) that spawnground uses unless told otherwise:
// far enough from the cells the spawnchest command takes (its odd indexes 1, 3, 5) for both to be in one scenario.
const spawnGroundDefaultRing = 6

// commandSpawnGround is the test console command "spawnground gold <amount> [cell]" or "spawnground <item code> [cell]":
// one gold pile or normal item on a free ground cell next to the hero (the n-th nearest, default 6), for the scripted
// click scenarios (click:left@item picks it up). Unlike spawnitem this is the ground entity of the drop code, so
// the click, the walk and the pickup are the real ones.
func (v *Game) commandSpawnGround(args []string) error {
	if v.localPlayer == nil || len(args) == 0 {
		return fmt.Errorf("spawnground gold <amount> [cell] | <item code> [cell]")
	}

	ring := spawnGroundDefaultRing
	rest := args[1:]

	var amount int

	if args[0] == "gold" {
		if len(rest) == 0 {
			return fmt.Errorf("spawnground gold <amount> [cell]")
		}

		n, err := strconv.Atoi(rest[0])
		if err != nil || n <= 0 {
			return fmt.Errorf("bad gold amount %q", rest[0])
		}

		amount, rest = n, rest[1:]
	}

	if len(rest) > 0 {
		n, err := strconv.Atoi(rest[0])
		if err != nil || n < 0 {
			return fmt.Errorf("bad cell index %q", rest[0])
		}

		ring = n
	}

	px, py := v.localPlayer.GetPositionF()

	cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), ring+1, true)
	if ring >= len(cells) {
		return fmt.Errorf("only %d free cells next to the hero", len(cells))
	}

	c := cells[ring]

	if amount > 0 {
		ent, err := v.spawnGoldPile(amount, c)
		if err != nil {
			return err
		}

		v.Infof("AUTOGROUND spawn source=command name=%q quality=gold pos=(%d,%d)", plainLabel(ent.Label()), c.X, c.Y)

		return nil
	}

	it, err := v.itemFactory().ItemFromCode(args[0], d2drop.QualityNormal, v.areaLevel(), 4242)
	if err != nil {
		return err
	}

	it.Identify()

	ent, err := v.spawnGroundItem(it, c)
	if err != nil {
		return err
	}

	v.Infof("AUTOGROUND spawn source=command name=%q quality=%s pos=(%d,%d)", plainLabel(ent.Label()), it.QualityName(), c.X, c.Y)

	return nil
}
