package d2gamescreen

import (
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
)

// spawnRoomObjects creates the random room objects the real-map generator planned (d2mapgen
// real_objects.go, the port of OBJRGN_SpawnRoomObjects) and logs a per-level census:
//
//	ROOMOBJ-SPAWN level N: made=M of P planned name=count ...
func (v *Game) spawnRoomObjects(level int) {
	plan := v.gameClient.MapEngine.RoomObjects()
	if len(plan) == 0 {
		return
	}

	counts := map[string]int{}
	made := 0

	for _, po := range plan {
		rec := v.asset.Records.Object.Details[po.ID]
		if rec == nil {
			continue
		}

		ob, err := v.gameClient.MapEngine.NewObject(po.X, po.Y, rec, d2resource.PaletteUnits)
		if err != nil {
			v.Warningf("ROOMOBJ object %d (%s) at (%d,%d): %v", po.ID, rec.Name, po.X, po.Y, err)
			continue
		}

		v.gameClient.MapEngine.AddEntity(ob)

		counts[strings.ReplaceAll(rec.Name, " ", "_")]++
		made++
	}

	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}

	sort.Strings(names)

	var sb strings.Builder
	for _, n := range names {
		sb.WriteString(" " + n + "=" + strconv.Itoa(counts[n]))
	}

	v.Infof("ROOMOBJ-SPAWN level %d: made=%d of %d planned%s", level, made, len(plan), sb.String())
}
