package d2gamescreen

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2act3"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// Act 3 in the running game: the quest objects (Khalim's chests, Lam Esen's Tome, Gidbinn), the Compelling
// Orb of Travincal, the Council as super uniques with their packs and Khalim's Flail, and Mephisto's red
// portal to the Pandemonium Fortress (docs/PLAYTEST.md, "Act 3 in depth"). The rules are in
// d2common/d2act3; the quest bookkeeping stays in d2common/d2quest, which this file feeds with the same
// events the original raises (object operated, item picked up, monster killed).

// act3State is the Act 3 state of the game screen.
type act3State struct {
	orb         *d2mapentity.Object // the Compelling Orb of the current Travincal, nil elsewhere
	hellgateAt  [2]int              // sub-tile position of the red portal of the lair (valid when hellgateSet)
	hellgateSet bool
	hellgate    *d2mapentity.Object // the red portal while it stands
	portalIn    float64             // frames until the red portal appears, <= 0: not pending
	// flailDropped is set for the whole session once Khalim's Flail lay on the ground (it is not in the
	// inventory yet, so the carry test alone would drop one per Council member)
	flailDropped bool
	// leverPulled is set for the session once the lever of the Lower Kurast Sewers was pulled
	leverPulled bool
}

// questItemInHand says whether the hero carries a quest item (the inventory or the quest system's count).
func (v *Game) questItemInHand(code string) bool {
	if v.gameControls != nil && v.gameControls.ItemCountsByCode()[code] > 0 {
		return true
	}

	r := v.questRT

	return r != nil && r.g.Items[code] > 0
}

// act3Enter runs when the hero arrives in a level.
func (v *Game) act3Enter(level int) {
	v.act3 = act3State{flailDropped: v.act3.flailDropped, leverPulled: v.act3.leverPulled}

	if d2level.ActOfLevel(level) != 3 || v.gameClient == nil || v.gameClient.MapEngine == nil {
		return
	}

	v.logAct3Objects(level)

	switch level {
	case d2act3.LevelTravincal:
		v.placeCompellingOrb()
	case d2act3.LevelDurance3:
		v.prepareHellgate()
	}
}

// logAct3Objects lists the objects of the freshly built level (the playthrough scenarios read this line).
func (v *Game) logAct3Objects(level int) {
	counts := map[string]int{}

	for _, e := range v.gameClient.MapEngine.Entities() {
		if ob, ok := e.(*d2mapentity.Object); ok {
			rec := ob.Record()
			counts[rec.Name+"#"+strconv.Itoa(rec.Index)]++
		}
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var sb strings.Builder

	for _, k := range keys {
		sb.WriteString(" " + k + "x" + strconv.Itoa(counts[k]))
	}

	v.Infof("ACT3 level %d objects:%s", level, sb.String())
}

// newObjectAt creates an object of objects.txt at a sub-tile position and adds it to the map.
func (v *Game) newObjectAt(id, subX, subY int) *d2mapentity.Object {
	rec := v.asset.Records.Object.Details[id]
	if rec == nil {
		v.Warningf("ACT3 no objects.txt row %d", id)

		return nil
	}

	ob, err := v.gameClient.MapEngine.NewObject(subX, subY, rec, d2resource.PaletteUnits)
	if err != nil {
		v.Warningf("ACT3 object %d at (%d,%d): %v", id, subX, subY, err)

		return nil
	}

	v.gameClient.MapEngine.AddEntity(ob)

	return ob
}

// objectsOf lists the objects of the map with an objects.txt id.
func (v *Game) objectsOf(id int) []*d2mapentity.Object {
	var out []*d2mapentity.Object

	for _, e := range v.gameClient.MapEngine.Entities() {
		if ob, ok := e.(*d2mapentity.Object); ok && ob.Record().Index == id {
			out = append(out, ob)
		}
	}

	return out
}

// placeCompellingOrb creates the Compelling Orb at the stairs to the Durance of Hate. No DS1 carries the
// orb (it has no row in the object lookup), so the game makes it: it stands on the dummy object 386 "stairs
// of the Compelling Orb" of the Travincal preset. The orb is gone once Khalim's Will smashed it.
func (v *Game) placeCompellingOrb() {
	if r := v.quests(); r != nil && d2level.OrbSmashed(v.heroQuests()) {
		v.Infof("ACT3 the Compelling Orb is already smashed")

		return
	}

	stairs := v.objectsOf(d2act3.ObjDuranceStairs)
	if len(stairs) == 0 {
		v.Infof("ACT3 Travincal has no stairs object (386): no Compelling Orb placed")

		return
	}

	x, y := stairs[0].GetPositionF()

	v.act3.orb = v.newObjectAt(d2act3.ObjCompellingOrb, int(math.Floor(x*subtilesInTile)), int(math.Floor(y*subtilesInTile)))
	if v.act3.orb != nil {
		v.Infof("ACT3 Compelling Orb placed at the Durance stairs (%.1f,%.1f)", x, y)
	}
}

// prepareHellgate hides the red portal of Mephisto's lair until Mephisto is dead (the DS1 carries the
// portal object 342, which the original shows only after the kill) and points it to the Fortress.
func (v *Game) prepareHellgate() {
	gates := v.objectsOf(d2act3.ObjHellgate)
	if len(gates) == 0 {
		v.Infof("ACT3 Durance of Hate 3 has no hellgate object (342)")

		return
	}

	gate := gates[0]
	x, y := gate.GetPositionF()
	v.act3.hellgateAt = [2]int{int(math.Floor(x * subtilesInTile)), int(math.Floor(y * subtilesInTile))}
	v.act3.hellgateSet = true

	for _, g := range gates {
		v.gameClient.MapEngine.RemoveEntity(g)
	}

	if d2act3.HellgateOpen(d2level.MephistoPortalOpen(v.heroQuests())) {
		v.openHellgate("Mephisto was killed in an earlier visit")
	}
}

// openHellgate stands the red portal up.
func (v *Game) openHellgate(why string) {
	if v.act3.hellgate != nil || !v.act3.hellgateSet {
		return
	}

	ob := v.newObjectAt(d2act3.ObjHellgate, v.act3.hellgateAt[0], v.act3.hellgateAt[1])
	if ob == nil {
		return
	}

	ob.PortalDest = d2act3.FortressStart
	ob.PortalOwner = v.gameClient.PlayerID
	v.act3.hellgate = ob
	v.act3.portalIn = 0

	v.Infof("ACT3 the red portal to the Pandemonium Fortress opens at (%d,%d) (%s)", v.act3.hellgateAt[0], v.act3.hellgateAt[1], why)
}

// advanceAct3 runs the timer of the red portal.
func (v *Game) advanceAct3(elapsed float64) {
	if v.act3.portalIn <= 0 {
		return
	}

	v.act3.portalIn -= elapsed * questFrameRate
	if v.act3.portalIn <= 0 {
		v.openHellgate("the delay after Mephisto's death is over")
	}
}

// act3Killed reacts to a monster death: Khalim's Flail from the Council, Mephisto's red portal.
func (v *Game) act3Killed(ev d2monsters.KillEvent) {
	if v.gameClient == nil || d2level.ActOfLevel(v.currentLevel()) != 3 {
		return
	}

	super := ""
	if ev.Monster != nil {
		super = ev.Monster.SuperUnique
	}

	if code, ok := d2act3.FlailDrop(super, v.questItemInHand); ok && !v.act3.flailDropped {
		v.act3.flailDropped = true
		x, y := 0, 0

		if ev.Monster != nil {
			x, y = ev.Monster.SubtilePos()
		}

		v.Infof("ACT3 %s drops Khalim's Flail (%s) at (%d,%d)", super, code, x, y)
		v.dropQuestItemAt(code, x, y)
	}

	if ev.Class == 242 && v.currentLevel() == d2act3.LevelDurance3 && v.act3.hellgateSet { // Mephisto
		v.act3.portalIn = d2act3.HellgateDelayFrames
		v.Infof("ACT3 Mephisto is dead: the red portal opens in %d frames", d2act3.HellgateDelayFrames)
	}
}

// dropQuestItemAt lays a quest item on the ground at a sub-tile (the monster that carried it, the chest it was
// in); without a position or director it falls at the hero's feet.
func (v *Game) dropQuestItemAt(code string, subX, subY int) {
	if v.monsters == nil || (subX == 0 && subY == 0) || v.asset.Records.Item.All[code] == nil {
		v.spawnQuestItem(code)

		return
	}

	if err := v.monsters.DropItemCode(code, subX/subtilesInTile, subY/subtilesInTile); err != nil {
		v.Infof("ACT3 could not drop %s at (%d,%d): %v", code, subX, subY, err)
		v.spawnQuestItem(code)
	}
}

// act3WarpAllowed applies the Act 3 rules of the stairs the quest system does not know (the sewer lever).
func (v *Game) act3WarpAllowed(from, to int) error {
	return d2act3.CheckSewerStairs(from, to, v.act3.leverPulled)
}

// warpRefused says whether the rules refuse the stair from one level to another (no log line).
func (v *Game) warpRefused(from, to int) bool {
	if r := v.questRT; r != nil && d2level.CheckActThreeWarp(from, to, r.g.Rec) != nil {
		return true
	}

	return v.act3WarpAllowed(from, to) != nil
}

// act3Operate carries out the operation of an Act 3 quest object or the red portal. It reports whether the
// object was one of them.
func (v *Game) act3Operate(ob *d2mapentity.Object) bool {
	id := ob.Record().Index

	if id == d2act3.ObjHellgate {
		v.operateHellgate(ob)

		return true
	}

	q, ok := d2act3.Object(id)
	if !ok {
		return false
	}

	if id == d2act3.ObjCompellingOrb {
		v.smashOrb(ob)

		return true
	}

	opened, err := ob.Open()
	if err != nil {
		v.Warningf("ACT3 object %q: %v", ob.Label(), err)
	}

	if !opened {
		v.Infof("ACT3 %s is already open", q.Name)

		return true
	}

	v.Infof("ACT3 %s operated (id %d): gives %q", q.Name, id, q.Item)
	v.questObjectOperated(ob)

	if id == d2act3.ObjSewerLever {
		v.act3.leverPulled = true
		v.Infof("ACT3 the lever is pulled: the stairs to the Lower Kurast Sewers 2 work")
	}

	if q.Item != "" {
		x, y := ob.GetPositionF()
		v.dropQuestItemAt(q.Item, int(math.Floor(x*subtilesInTile)), int(math.Floor(y*subtilesInTile))+2)
	}

	return true
}

// smashOrb is the hero using the Compelling Orb: it breaks with Khalim's Will in the inventory and the
// stairs to the Durance of Hate open (the quest system sets the bits that d2level.CheckActThreeWarp reads).
func (v *Game) smashOrb(ob *d2mapentity.Object) {
	if !d2act3.CanSmashOrb(v.questItemInHand) {
		v.Infof("ACT3 the Compelling Orb does not break: the hero has no Khalim's Will")

		return
	}

	v.Infof("ACT3 the Compelling Orb is smashed with Khalim's Will")
	v.questObjectOperated(ob)
	v.gameClient.MapEngine.RemoveEntity(ob)
	v.act3.orb = nil
}

// operateHellgate is the hero using the red portal: the act change to the Pandemonium Fortress (the rule
// of d2level.CheckActTravel wants Mephisto dead).
func (v *Game) operateHellgate(ob *d2mapentity.Object) {
	v.Infof("ACT3 the hero enters the red portal")

	_ = v.travelToAct(4, "portal")
}
