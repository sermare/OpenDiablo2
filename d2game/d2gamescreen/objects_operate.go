package d2gamescreen

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"

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

	// variant is the object variant byte (unit +0x78) bit 0 of OBJECT_OperateLootContainer; its source in
	// the original (the creating request) is not decoded, so it is false unless a caller sets it.
	variant   bool
	chestInit *d2object.ChestInit // lock and spawn handler rolled on first use (objects_gaps.go)

	// wells: pulses left (counter in the object data byte +4 of the exe) and the game times at which a spent pulse
	// returns (the refill events of 0x57f410); wellInit tells the counter was set.
	wellInit    bool
	charges     int
	wellRefills []float64
}

// objectState is the world-object state of the game screen.
type objectState struct {
	clock     float64 // game seconds
	overlay   d2object.Overlay
	instances map[string]*objInstance
	shrines   []d2object.Shrine
	lootSeq   uint32
	spawns    []pendingSpawn     // container spawn handlers waiting for their delay
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
	// openChest (ground_items.go) plans the container first: the exploding barrel (OperateFn 7) hurts the units around
	// it, a locked chest needs a key, a trapped one arms its spawn handler (objects_gaps.go)
	v.openChest(ob)
}

// rackBases lists the weapons.txt (rack) or armor.txt (stand) rows for the picker. The records are a map, so the rows
// are sorted by code (the exe walks the file order: UNVERIFIED effect on which row a seed picks).
func (v *Game) rackBases(weapons bool) []d2object.RackBase {
	tbl := v.asset.Records.Item.Armors
	if weapons {
		tbl = v.asset.Records.Item.Weapons
	}

	rows := make([]d2object.RackBase, 0, len(tbl))

	for code, r := range tbl {
		rows = append(rows, d2object.RackBase{Code: code, QLvl: r.Level, Rarity: r.Rarity, Spawnable: r.Spawnable,
			Quest: r.Quest != 0, Expansion: r.Version >= 100})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Code < rows[j].Code })

	return rows
}

// operateRack operates a weapon rack (OperateFn 20) or armor stand (19). VERIFIED (0x582050, 0x581fe0): when the
// object is in mode 0 it creates ONE random base item - a uniform pick among the eligible rows of
// weapons.txt / armor.txt (d2object.PickRackBase), at item level monsterLevel-1, quality chosen by the item
// creation - near the object, goes to mode 2 and becomes unselectable. No treasure class is rolled.
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

	v.objects.lootSeq++
	seed := v.chestSeed() + 1000 + v.objects.lootSeq
	weapons := rec.OperateFn == 20

	// the item level follows from the monster level of the area (the same lookup the chests use)
	_, opts, err := v.itemFactory().ChestSetup(diablo2item.ChestDropOptions{LevelID: v.currentLevel(), Difficulty: v.difficulty(), Expansion: true})
	if err != nil {
		v.Warningf("OBJECT rack %d: %v", rec.Index, err)
		return
	}

	ilvl := d2object.RackItemLevel(opts.ILvl)
	code, ok := d2object.PickRackBase(v.rackBases(weapons), ilvl, false, d2object.NewRoller(seed))

	if !ok {
		v.Warningf("OBJECT rack %d: no eligible base item at item level %d", rec.Index, ilvl)
		return
	}

	item, err := v.itemFactory().Create(diablo2item.CreateParams{Code: code, ILvl: ilvl, Seed: seed, Difficulty: v.difficulty()})
	if err != nil {
		v.Warningf("OBJECT rack %d: creating %q: %v", rec.Index, code, err)
		return
	}

	v.Infof("OBJECT operate id=%d name=%q fn=%d class=rack sound=%s base=%q ilvl=%d seed=%d drops=1",
		rec.Index, ob.Label(), rec.OperateFn, sound, code, ilvl, seed)

	cx, cy := ob.GetPositionF()
	v.spawnLoot(&diablo2item.Loot{Entries: []diablo2item.LootEntry{{Item: item}}}, int(math.Floor(cx)), int(math.Floor(cy)), "rack")
}

// operateWell is the well of OperateFn 22 (0x5837b0, VERIFIED). A well holds Parm2*2 pulses (2 for every well,
// init 0x550ba0). A pulse heals Parm1/256 of the maximum life, mana and stamina of the hero and of his
// pets and cures; only a pulse that changed something is spent. Every spent pulse schedules a refill event
// Parm0+1 frames later that returns one pulse (0x57f410). The object mode is 0 full, 1 half, 2 empty.
// UNVERIFIED: the hero's poison and freeze states and the minions' life are not modelled (only the
// mercenary is healed).
func (v *Game) operateWell(ob *d2mapentity.Object) {
	rec := ob.Record()
	in := v.objectInstance(ob)
	parm2 := rec.Parm[2]

	if !in.wellInit {
		in.wellInit, in.charges = true, d2object.WellCharges(parm2)
	}

	if in.charges <= 0 {
		v.Infof("OBJECT operate id=%d name=%q fn=%d class=well empty (next pulse in %.1fs)", rec.Index, ob.Label(),
			rec.OperateFn, in.rearmAt-v.objects.clock)
		return
	}

	before := v.vitalsText()
	st := v.localPlayer.Stats

	wv, changed := d2object.WellPulse(d2object.WellVitals{Vitals: v.heroVitals(), Stamina: int(st.Stamina), MaxStamina: st.MaxStamina},
		rec.Parm[1], rec.Parm[3])

	mercHealed := false

	if md := v.monsterDirector(); md != nil {
		if mi, ok := md.Merc(v.localPlayer); ok && mi.HP > 0 && mi.HP < mi.MaxHP {
			md.SetMercHP(v.localPlayer, mi.MaxHP)

			mercHealed = true
		}
	}

	if !changed && !mercHealed {
		v.Infof("OBJECT operate id=%d name=%q fn=%d class=well nothing to heal: the pulse is kept (%d left)", rec.Index,
			ob.Label(), rec.OperateFn, in.charges)
		return
	}

	v.setHeroVitals(wv.Vitals)
	st.Stamina = float64(wv.Stamina)
	sound := d2object.SoundFor(rec.OperateFn, rec.Name)
	v.playSoundAt(sound, ob.GetPosition(), "object")

	in.charges--
	in.uses++

	mode, _ := d2object.WellMode(in.charges, parm2)
	if in.charges == 0 {
		if _, err := ob.Open(); err != nil {
			v.Warningf("OBJECT %q: %v", ob.Label(), err)
		}
	}

	refill := float64(d2object.WellRefillFrames(rec.Parm[0])) / d2object.FramesPerSecond
	in.wellRefills = append(in.wellRefills, v.objects.clock+refill)
	in.rearmAt = v.objects.clock + refill

	v.Infof("OBJECT operate id=%d name=%q fn=%d class=well sound=%s before[%s] after[%s] pulses_left=%d mode=%d merc_healed=%v refill=%.1fs",
		rec.Index, ob.Label(), rec.OperateFn, sound, before, v.vitalsText(), in.charges, mode, mercHealed, refill)
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

// worldShrine runs the magic shrines. Portal is implemented (it opens a town
// portal, see portalShrine); the others are documented stubs.
func (v *Game) worldShrine(ob *d2mapentity.Object, s d2object.Shrine) string {
	w := d2object.WorldFor(s)

	if w == d2object.WorldPortal {
		return v.portalShrine()
	}

	switch w {
	case d2object.WorldStorm:
		return v.stormShrine(s)
	case d2object.WorldExploding:
		return v.potionShrine(ob, s, d2object.ExplodingShrine)
	case d2object.WorldPoison:
		return v.potionShrine(ob, s, d2object.PoisonShrine)
	case d2object.WorldWarping:
		return v.warpingShrine(ob)
	case d2object.WorldGemUpgrade:
		if v.gameControls != nil {
			return v.gemShrine(ob)
		}
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

	v.runSpawns()

	for id, in := range v.objects.instances {
		if in.wellInit {
			v.advanceWell(id, in)
			continue
		}

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

// advanceWell returns the pulses of a well whose refill events are due (0x57f410: one pulse per event, up to
// Parm2*2) and closes the object again when it had run empty.
func (v *Game) advanceWell(id string, in *objInstance) {
	if len(in.wellRefills) == 0 {
		return
	}

	rec := in.ob.Record()
	keep := in.wellRefills[:0]

	for _, at := range in.wellRefills {
		if v.objects.clock < at {
			keep = append(keep, at)
			continue
		}

		if in.charges >= d2object.WellCharges(rec.Parm[2]) || v.gameClient.MapEngine.Entities()[id] == nil {
			continue
		}

		wasEmpty := in.charges == 0
		in.charges++
		mode, _ := d2object.WellMode(in.charges, rec.Parm[2])

		if wasEmpty {
			if _, err := in.ob.Close(); err != nil {
				v.Warningf("OBJECT rearm %q: %v", in.ob.Label(), err)
			}
		}

		v.Infof("OBJECT rearmed id=%d name=%q pulses=%d mode=%d", rec.Index, in.ob.Label(), in.charges, mode)
	}

	in.wellRefills = keep
	if len(keep) == 0 {
		in.rearmAt = 0
	}
}

// portalShrine is OBJECT_OperateTeleportToNearActStart (0x580950, VERIFIED
// against the exe): the shrine does not move the hero. It opens a town portal
// pair, one end beside the hero and one in the town of the act, through the
// routine the Town Portal skill uses; unlike the skill it records no owner and
// leaves the hero's own portals alone (this engine's pair registry has one
// pair per owner, so the shrine's pair replaces the hero's: a known
// approximation). In a town room the exe does nothing.
func (v *Game) portalShrine() string {
	level := v.currentLevel()
	dest := d2level.PortalShrineDest(level)

	if !d2level.PortalShrineOpens(level) {
		return fmt.Sprintf("world=portal level=%d: no portal opens here", level)
	}

	if err := v.openTownPortal(); err != nil {
		return fmt.Sprintf("world=portal dest=%d refused: %v", dest, err)
	}

	return fmt.Sprintf("world=portal dest=%d (%s) portal opened", dest, v.levelName(dest))
}
