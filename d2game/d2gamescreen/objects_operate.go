package d2gamescreen

import (
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2ground"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2objspawn"
)

// World objects beyond doors, waypoints and portals: shrines, wells, weapon
// racks and armor stands, barrels/urns/crates and the other loot containers.
// The rules live in d2common/d2object (the OperateFn table as data); this
// file connects them to the click/walk code (walkToObject -> useObject ->
// operateObject -> operateWorldObject), the hero's stats and the drop code.

// objInstance is the per-object state the original keeps in the object data.
type objInstance struct {
	ob      *d2mapentity.Object
	shrine  *d2object.Shrine // rolled on first use (object data byte +4 in the original)
	rearmAt float64          // game time at which a used shrine / well can be used again, 0 = never
	uses    int
}

// objectState is the world-object state of the game screen.
type objectState struct {
	clock     float64 // game seconds
	overlay   d2object.Overlay
	instances map[string]*objInstance
	shrines   []d2object.Shrine
	lootSeq   uint32
	spawn     *d2objspawn.Tables // adapter view of objects.txt/objgroup.txt, built on first use
	spawnMu   sync.Mutex         // guards the lazy build of spawn
}

// spawnTables returns the d2object tables built from the loaded records.
func (v *Game) spawnTables() *d2objspawn.Tables {
	v.objects.spawnMu.Lock()
	defer v.objects.spawnMu.Unlock()

	if v.objects.spawn == nil {
		t := d2objspawn.FromRecords(v.asset.Records.Object.Details, v.asset.Records.Object.Groups)
		v.objects.spawn = &t
	}

	return v.objects.spawn
}

// instance returns the state of an object, creating it.
func (v *Game) objectInstance(ob *d2mapentity.Object) *objInstance {
	if v.objects.instances == nil {
		v.objects.instances = map[string]*objInstance{}
	}

	in := v.objects.instances[ob.ID()]
	if in == nil {
		in = &objInstance{ob: ob}
		v.objects.instances[ob.ID()] = in
	}

	return in
}

// objectIsOperable reports whether the click code should walk to the object
// and operate it with operateWorldObject.
func objectIsOperable(ob *d2mapentity.Object) bool {
	rec := ob.Record()
	info := d2object.InfoFor(d2object.Def{OperateFn: rec.OperateFn, SubClass: rec.SubClass})

	if !info.Operable() {
		return false
	}

	return rec.Selectable[0]
}

// objectSeed is a stable per-object seed (map seed and position), so a given
// game rolls the same shrine in the same place.
func (v *Game) objectSeed(ob *d2mapentity.Object) uint32 {
	x, y := ob.GetPositionF()
	h := fnv.New32a()
	fmt.Fprintf(h, "%d:%d:%d:%d", v.chestSeed(), ob.Record().Index, int(x*5), int(y*5))

	return h.Sum32()
}

func (v *Game) shrineTable() []d2object.Shrine {
	if v.objects.shrines == nil {
		v.objects.shrines = d2object.DefaultShrines
	}

	return v.objects.shrines
}

// vitals reads the hero's life and mana.
func (v *Game) heroVitals() d2object.Vitals {
	s := v.localPlayer.Stats

	return d2object.Vitals{Life: s.Health, MaxLife: s.MaxHealth, Mana: s.Mana, MaxMana: s.MaxMana}
}

func (v *Game) setHeroVitals(h d2object.Vitals) {
	s := v.localPlayer.Stats
	s.Health, s.Mana = h.Life, h.Mana
}

func (v *Game) vitalsText() string {
	h := v.heroVitals()

	return fmt.Sprintf("life=%d/%d mana=%d/%d", h.Life, h.MaxLife, h.Mana, h.MaxMana)
}

// operateWorldObject runs the OperateFn of an object the hero has reached.
func (v *Game) operateWorldObject(ob *d2mapentity.Object) {
	rec := ob.Record()
	info := d2object.InfoFor(d2object.Def{OperateFn: rec.OperateFn, SubClass: rec.SubClass})

	switch info.Class {
	case d2object.ClassShrine:
		v.operateShrine(ob)
	case d2object.ClassWell:
		v.operateWell(ob)
	case d2object.ClassRack:
		v.operateRack(ob)
	case d2object.ClassLoot, d2object.ClassExplode:
		v.operateContainer(ob, info)
	case d2object.ClassTeleport:
		v.operateTeleport(ob, info)
	default:
		if info.Stub {
			v.Infof("OBJECT stub %q (id %d) fn=%d class=%s: %s (%s)", ob.Label(), rec.Index, rec.OperateFn,
				info.Class, info.Name, info.Note)
		} else {
			v.Infof("OBJECT %q (id %d) fn=%d class=%s has no operate effect", ob.Label(), rec.Index, rec.OperateFn, info.Class)
		}
	}
}

// operateContainer opens a chest/casket/barrel/urn/crate: the opening
// animation, the sound and the treasure class drop of openChest.
func (v *Game) operateContainer(ob *d2mapentity.Object, info d2object.Info) {
	rec := ob.Record()
	if ob.IsOpened() {
		v.Infof("OBJECT operate id=%d name=%q fn=%d class=%s already opened", rec.Index, ob.Label(), rec.OperateFn, info.Class)
		return
	}

	v.Infof("OBJECT operate id=%d name=%q fn=%d class=%s sound=%s", rec.Index, ob.Label(), rec.OperateFn, info.Class,
		d2object.SoundFor(rec.OperateFn, rec.Name))
	v.openChest(ob)

	if info.Class == d2object.ClassExplode {
		// the explosion damage is objects.txt Damage; applying it to the hero and to monsters
		// near the barrel is not implemented (needs a damage API for the hero) - UNVERIFIED radius.
		v.Infof("OBJECT explosion id=%d damage=%d (not applied: TODO)", rec.Index, rec.Damage)
	}
}

// equipTreasureClass picks the "Act N Equip X" class for weapon racks and
// armor stands (the treasure class the original uses is not in the notes: the
// Equip classes are the equipment-only classes of the act - UNVERIFIED).
func (v *Game) equipTreasureClass() string {
	best := ""
	act := v.localPlayer.Act

	if act < 1 {
		act = 1
	}

	ilvl := v.areaLevel()

	for _, k := range d2ground.ChestKinds {
		name := fmt.Sprintf("Act %d Equip %s", act, k)

		lvl, ok := v.itemFactory().TreasureClassLevel(name)
		if !ok {
			continue
		}

		if best == "" || lvl <= ilvl {
			best = name
		}
	}

	return best
}

// operateRack operates a weapon rack or armor stand. VERIFIED (0x582050): when
// the object is in mode 0 it drops its items, goes to mode 2 and becomes
// unselectable. The treasure class is UNVERIFIED.
func (v *Game) operateRack(ob *d2mapentity.Object) {
	rec := ob.Record()

	opened, err := ob.Open()
	if err != nil {
		v.Warningf("OBJECT %q: %v", ob.Label(), err)
	}

	if !opened {
		v.Infof("OBJECT operate id=%d name=%q already used", rec.Index, ob.Label())
		return
	}

	sound := d2object.SoundFor(rec.OperateFn, rec.Name)
	v.playSoundAt(sound, ob.GetPosition(), "object")

	tc := v.equipTreasureClass()
	ilvl := v.areaLevel()
	v.objects.lootSeq++
	seed := v.chestSeed() + 1000 + v.objects.lootSeq

	loot, err := v.itemFactory().DropLoot(tc, diablo2item.DropOptions{Seed: seed, ILvl: ilvl, Players: 1, Difficulty: v.difficulty(), MagicFind: v.heroMagicFind(), GoldFind: v.heroGoldFind()}, 0)
	if err != nil {
		v.Warningf("OBJECT rack %d: tc %q: %v", rec.Index, tc, err)
		return
	}

	v.Infof("OBJECT operate id=%d name=%q fn=%d class=rack sound=%s tc=%q ilvl=%d seed=%d drops=%d",
		rec.Index, ob.Label(), rec.OperateFn, sound, tc, ilvl, seed, len(loot.Entries))

	cx, cy := ob.GetPositionF()
	v.spawnLoot(loot, int(math.Floor(cx)), int(math.Floor(cy)), "rack")
}

// operateWell heals Parm1/256 of the maximum life, mana and stamina (VERIFIED, see d2object.WellPulse), plays the well
// sound, empties the well for WellRefillSeconds and then refills it.
func (v *Game) operateWell(ob *d2mapentity.Object) {
	rec := ob.Record()
	in := v.objectInstance(ob)

	opened, err := ob.Open()
	if err != nil {
		v.Warningf("OBJECT %q: %v", ob.Label(), err)
	}

	if !opened {
		v.Infof("OBJECT operate id=%d name=%q fn=%d class=well empty (refills in %.1fs)", rec.Index, ob.Label(),
			rec.OperateFn, in.rearmAt-v.objects.clock)
		return
	}

	before := v.vitalsText()
	st := v.localPlayer.Stats

	wv, _ := d2object.WellPulse(d2object.WellVitals{Vitals: v.heroVitals(), Stamina: int(st.Stamina), MaxStamina: st.MaxStamina},
		rec.Parm[1], rec.Parm[3])
	v.setHeroVitals(wv.Vitals)
	st.Stamina = float64(wv.Stamina)
	sound := d2object.SoundFor(rec.OperateFn, rec.Name)
	v.playSoundAt(sound, ob.GetPosition(), "object")

	refill := d2object.WellRefillSeconds(rec.Parm[0])
	in.rearmAt = v.objects.clock + refill
	in.uses++

	v.Infof("OBJECT operate id=%d name=%q fn=%d class=well sound=%s before[%s] after[%s] refill=%.1fs",
		rec.Index, ob.Label(), rec.OperateFn, sound, before, v.vitalsText(), refill)
}

// operateShrine rolls the shrine type on first use (the original stores it in
// the object data), applies its effect and starts the reset timer.
func (v *Game) operateShrine(ob *d2mapentity.Object) {
	rec := ob.Record()
	in := v.objectInstance(ob)

	if ob.IsOpened() {
		v.Infof("OBJECT operate id=%d name=%q fn=%d class=shrine already used (resets in %.1fs)", rec.Index, ob.Label(),
			rec.OperateFn, in.rearmAt-v.objects.clock)
		return
	}

	if in.shrine == nil {
		r := d2object.NewRoller(v.objectSeed(ob))

		var (
			s  d2object.Shrine
			ok bool
		)

		if rec.Parm[0] == 0 {
			s, ok = d2object.RollShrineUniform(v.shrineTable(), r, v.areaLevel()) // VERIFIED path (0x54d840)
		} else {
			s, ok = d2object.RollShrine(v.shrineTable(), r, v.areaLevel(), rec.Parm[0], false)
		}
		if !ok {
			v.Warningf("OBJECT shrine %d: no shrine type fits area level %d", rec.Index, v.areaLevel())
			return
		}

		in.shrine = &s
	}

	v.useShrine(ob, in, *in.shrine)
}

// useShrine applies one shrine type to the hero (also used by the autotest to
// force a type).
func (v *Game) useShrine(ob *d2mapentity.Object, in *objInstance, s d2object.Shrine) {
	rec := ob.Record()

	if _, err := ob.Open(); err != nil {
		v.Warningf("OBJECT %q: %v", ob.Label(), err)
	}

	in.uses++
	v.playSoundAt(s.Sound, ob.GetPosition(), "object")

	effect := ""
	before := v.vitalsText()

	if after, ok := d2object.ApplyInstant(s, v.heroVitals()); ok {
		v.setHeroVitals(after)

		effect = fmt.Sprintf("instant before[%s] after[%s]", before, v.vitalsText())
	} else if b, ok := d2object.NewShrineBuff(s, v.objects.clock); ok {
		v.objects.overlay.Buffs.Add(b)
		v.objects.overlay.Apply(v.localPlayer.Stats.Totals, v.objects.clock)

		effect = fmt.Sprintf("buff state=%d stats=[%s] duration=%.0fs expires_at=%.1f", b.State, b.Stats.String(),
			s.DurationSeconds(), b.Expires)
	} else {
		effect = v.worldShrine(ob, s)
	}

	if s.ResetFrames() > 0 {
		in.rearmAt = v.objects.clock + s.ResetSeconds()
	}

	reset := "never"
	if in.rearmAt > 0 {
		reset = fmt.Sprintf("%.1fs", in.rearmAt-v.objects.clock)
	}

	v.Infof("OBJECT operate id=%d name=%q fn=%d class=shrine shrine=%d %q sound=%s %s reset=%s",
		rec.Index, ob.Label(), rec.OperateFn, s.Code, s.Name, s.Sound, effect, reset)
}

// townLevelOf is the town of an act (1-based).
var townLevelOf = map[int]int{1: 1, 2: 40, 3: 75, 4: 103, 5: 109}

// worldShrine runs the magic shrines. Portal is implemented (a town portal
// next to the hero); the others are documented stubs.
func (v *Game) worldShrine(ob *d2mapentity.Object, s d2object.Shrine) string {
	w := d2object.WorldFor(s)

	if w == d2object.WorldPortal {
		act := v.localPlayer.Act
		if act < 1 || act > 5 {
			act = 1
		}

		town := townLevelOf[act]
		if err := v.commandSpawnPortal([]string{strconv.Itoa(town)}); err != nil {
			return fmt.Sprintf("world=portal failed: %v", err)
		}

		return fmt.Sprintf("world=portal dest=%d", town)
	}

	return fmt.Sprintf("world=%s STUB (not implemented: %s)", w, s.Effect)
}

// advanceObjects runs the clock of the object effects: shrine buffs on the
// hero's totals, their expiry, the unlimited stamina, and the re-arming of
// used shrines and wells.
func (v *Game) advanceObjects(elapsed float64) {
	v.objects.clock += elapsed

	if v.localPlayer == nil || v.localPlayer.Stats == nil {
		return
	}

	st := v.localPlayer.Stats

	for _, b := range v.objects.overlay.Apply(st.Totals, v.objects.clock) {
		v.Infof("OBJECT effect expired %s state=%d", b.Source, b.State)
	}

	if v.objects.overlay.UnlimitedStamina() && st.MaxStamina > 0 {
		st.Stamina = float64(st.MaxStamina)
	}

	for id, in := range v.objects.instances {
		if in.rearmAt <= 0 || v.objects.clock < in.rearmAt {
			continue
		}

		in.rearmAt = 0

		if v.gameClient.MapEngine.Entities()[id] == nil {
			continue
		}

		if closed, err := in.ob.Close(); err != nil {
			v.Warningf("OBJECT rearm %q: %v", in.ob.Label(), err)
		} else if closed {
			v.Infof("OBJECT rearmed id=%d name=%q", in.ob.Record().Index, in.ob.Label())
		}
	}
}

// experienceBonusPct is the shrine experience bonus, used by the monster director.
func (v *Game) experienceBonusPct() int { return v.objects.overlay.ExperiencePct() }
