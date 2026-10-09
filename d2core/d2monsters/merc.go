package d2monsters

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// mercTargetBase offsets the target ids of mercenaries so hostile monsters
// can tell them from players (players use 1..n).
const mercTargetBase = 1 << 16

// buffSeconds is how long a merc buff (Might, Blessed Aim, Inner Sight...)
// counts as active in the skill choice. UNVERIFIED: the real duration comes
// from the skill's aura/state data, which is not simulated.
const buffSeconds = 120

// mercExpMultiplier is the multiplier on a kill's experience a merc receives
// (VERIFIED, MERC_AwardExperience 0x57c860: exp += 2 * gain, where gain is the
// merc's own pipeline result, times 86/256 when the merc did not make the
// kill; the caller 0x57c990 passes it in EAX).
const mercExpMultiplier = 2

// mercSharePct256 is the share (in 1/256) of a kill the merc did not make
// itself that it is credited (IMUL 0x56, SAR 8 at 0x57ca2d, VERIFIED).
const mercSharePct256 = 0x56

// MercSave is the persistent part of a mercenary (the d2s header fields).
type MercSave struct {
	Dead       bool
	ID         uint32
	NameID     uint16
	Type       uint16
	Experience uint32
	// Gear is the merc's equipment as stat list items (see d2hero.HeroStateFactory.MercStatItems).
	// It is not part of the save header; SpawnMerc applies it.
	Gear []d2statlist.Item
}

// MercInfo is a snapshot for the UI and logs.
type MercInfo struct {
	Save       MercSave
	Level      int
	HP, MaxHP  int
	Rec        *d2hireling.Record
	Stats      d2hireling.Stats
	ReviveCost int
	// Base is the stats of the hireling table at the current level (no gear) and Gear the
	// effect of the equipment on them; Stats equals the gear-modified numbers. Gear.Resist
	// (fire, cold, lightning, poison) is shown only: monster hits on a merc are not
	// reduced by it yet.
	Base d2hireling.Stats
	Gear d2hireling.Gear
}

// mercUnit is the merc-specific state of a unit.
type mercUnit struct {
	owner  *d2mapentity.Player
	rec    *d2hireling.Record
	save   MercSave
	level  int
	stats  d2hireling.Stats // the table stats with the gear applied
	base   d2hireling.Stats // the table stats of the current level
	items  []d2statlist.Item
	gear   d2hireling.Gear
	ranged bool
	buffs  map[string]int // skill name -> frame the buff ends

	lastOwnerX, lastOwnerY int
	ownerMode              d2monster.Mode
	skill                  string // the skill being cast, for logs
	regen                  mercRegen
}

// SetHirelings gives the director the parsed hireling.txt; without it no merc
// can be spawned.
func (d *Director) SetHirelings(t *d2hireling.Table) { d.hire = t }

// Hirelings returns the table set with SetHirelings (nil if none).
func (d *Director) Hirelings() *d2hireling.Table { return d.hire }

// fallbackClass maps hireling classes whose monstats row has no usable
// animation to one that has (the rogue hireling is a dummy row; RogueScout
// is the rogue archer).
var fallbackClass = map[int]int{271: 270} //nolint:gochecknoglobals // static lookup

// SpawnMerc creates (or replaces) the owner's mercenary next to the owner
// from its saved fields. The level is derived from the experience. A dead
// merc is spawned as a corpse so it can be revived.
func (d *Director) SpawnMerc(owner *d2mapentity.Player, save MercSave) (*d2mapentity.Monster, error) {
	if d.hire == nil {
		return nil, errors.New("no hireling table")
	}

	id := int(save.Type)
	level := d.hire.LevelFromExp(id, save.Experience)

	st, rec := d.hire.StatsFor(id, level)
	if rec == nil {
		return nil, fmt.Errorf("unknown hireling type %d", id)
	}

	stat := d.statByID[rec.Class]
	if stat == nil {
		stat = d.statByID[fallbackClass[rec.Class]]
	}

	if stat == nil {
		return nil, fmt.Errorf("no monstats row for hireling class %d", rec.Class)
	}

	d.DismissMerc(owner)

	ox, oy := playerSubtile(owner)

	p, ok := d2path.NearestFree(d.grid, d2path.MaskMonster, d2path.Point{X: ox + 3, Y: oy}, 10)
	if !ok {
		p = d2path.Point{X: ox, Y: oy}
	}

	m, err := d.spawnMercAt(owner, stat, rec, save, level, st, p.X, p.Y)
	if err != nil && fallbackClass[rec.Class] != 0 && d.statByID[fallbackClass[rec.Class]] != stat {
		m, err = d.spawnMercAt(owner, d.statByID[fallbackClass[rec.Class]], rec, save, level, st, p.X, p.Y)
	}

	return m, err
}

func (d *Director) spawnMercAt(owner *d2mapentity.Player, stat *d2records.MonStatRecord, rec *d2hireling.Record,
	save MercSave, level int, st d2hireling.Stats, x, y int) (*d2mapentity.Monster, error) {
	prof := profileFromRecord(stat, d.opt.Difficulty)
	d.nextID++

	b := d2monster.NewBrain(d.nextID, rec.Class, d.opt.Difficulty, prof, d.opt.Seed)
	b.X, b.Y = x, y

	if def, ok := d2monster.Lookup("Hireable"); ok {
		b.SetAI(def)
	}

	m, err := d.engine.NewMonster(x, y, stat, 0, b)
	if err != nil {
		return nil, err
	}

	if m.StatEx.SizeX/2 > 1 {
		b.Size = m.StatEx.SizeX / 2
	}

	m.Vitals = d2mapentity.MonsterVitals{
		Level: level, HP: st.MaxHP, MaxHP: st.MaxHP, Defense: st.Defense, Difficulty: d.opt.Difficulty,
		A1: MonsterAttackFrom(st.AR, st.DmgMin, st.DmgMax),
	}

	b.Wake = d.frame
	m.SetSelectable(false)
	m.Blocker = func(x, y int) bool { return d.fp.BlockedFor(b.ID, x, y) }
	d.fp.Move(b.ID, x, y, d2path.FlagMonster)

	if key := d.MercName(rec, save); d.asset.TranslateString(key) != "" && d.asset.TranslateString(key) != key {
		m.SetLabel(d.asset.TranslateString(key))
	}

	mu := &mercUnit{
		owner: owner, rec: rec, save: save, level: level, stats: st, base: st, items: save.Gear,
		ranged: rec.Class == 271 || rec.Class == 359, buffs: map[string]int{},
		ownerMode: d2monster.ModeNeutral,
	}
	mu.lastOwnerX, mu.lastOwnerY = playerSubtile(owner)

	u := &unit{m: m, b: b, merc: mu}
	d.units[b.ID] = u
	d.byEntity[m.ID()] = u
	d.mercs[owner] = u
	d.engine.AddEntity(m)

	if len(mu.items) > 0 {
		d.applyMercGear(u, true)
	}

	d.Counters.MercSpawns++
	d.emit("merc", "MERC spawn name=%s id=%08x type=%d class=%d level=%d hp=%d/%d defense=%d dmg=%d-%d exp=%d dead=%v pos=(%d,%d)",
		d.MercName(rec, save), save.ID, save.Type, rec.Class, level, st.MaxHP, st.MaxHP, st.Defense, st.DmgMin, st.DmgMax,
		save.Experience, save.Dead, x, y)

	if save.Dead {
		m.Vitals.HP = 0
		m.Die()
		d.emit("merc", "MERC corpse name=%s (dead in the save)", d.MercName(rec, save))
	}

	return m, nil
}

// MercName is the string.tbl key of the merc's name (translated by the UI).
func (d *Director) MercName(rec *d2hireling.Record, s MercSave) string {
	return d2hireling.NameKey(rec, int(s.NameID))
}

// DismissMerc removes the owner's current mercenary, if any.
func (d *Director) DismissMerc(owner *d2mapentity.Player) {
	u := d.mercs[owner]
	if u == nil {
		return
	}

	d.engine.RemoveEntity(u.m)
	d.forget(u)
	delete(d.mercs, owner)
	d.emit("merc", "MERC dismissed id=%08x", u.merc.save.ID)
}

// Merc returns a snapshot of the owner's mercenary.
func (d *Director) Merc(owner *d2mapentity.Player) (MercInfo, bool) {
	u := d.mercs[owner]
	if u == nil {
		return MercInfo{}, false
	}

	mu := u.merc

	return MercInfo{
		Save: mu.currentSave(u), Level: mu.level, HP: u.m.Vitals.HP, MaxHP: u.m.Vitals.MaxHP, Rec: mu.rec,
		Stats: mu.stats, ReviveCost: d2hireling.ReviveCost(mu.level), Base: mu.base, Gear: mu.gear,
	}, true
}

// SetMercItems sets the merc's equipment (stat list items, see MercStatItems of the hero
// factory) and applies it with d2hireling.ApplyGear: defense, life, attack rating, damage
// and the gear's resists on MercInfo. The items are remembered: a level-up and a revive
// apply them again, so the gear is not lost. Calling it with no items restores the table
// values, so a merc without gear behaves exactly as before. Life is kept in proportion.
func (d *Director) SetMercItems(owner *d2mapentity.Player, items []d2statlist.Item) bool {
	u := d.mercs[owner]
	if u == nil {
		return false
	}

	u.merc.items = append([]d2statlist.Item(nil), items...)
	d.applyMercGear(u, false)

	d.emit("merc", "MERC gear name=%s items=%d defense=%d hp=%d dmg=%d-%d ar=%d resist=%v", u.m.Label(), len(items),
		u.merc.stats.Defense, u.merc.stats.MaxHP, u.merc.stats.DmgMin, u.merc.stats.DmgMax, u.merc.stats.AR, u.merc.gear.Resist)

	return true
}

// applyMercGear computes the gear effect on the table stats of the current level and
// writes it to the unit. full sets the life to the new maximum (level-up, revive, spawn),
// otherwise it keeps the proportion.
func (d *Director) applyMercGear(u *unit, full bool) {
	mu := u.merc
	g := d2hireling.ApplyGear(mu.base, mu.items)

	d.setMercGear(u, g, full)
}

// SetMercGear applies a computed gear effect to the owner's merc (the low-level form of
// SetMercItems; the effect is not remembered across a level-up).
func (d *Director) SetMercGear(owner *d2mapentity.Player, g d2hireling.Gear) bool {
	u := d.mercs[owner]
	if u == nil {
		return false
	}

	d.setMercGear(u, g, false)

	return true
}

func (d *Director) setMercGear(u *unit, g d2hireling.Gear, full bool) {
	mu := u.merc
	frac := float64(1)

	if u.m.Vitals.MaxHP > 0 {
		frac = float64(u.m.Vitals.HP) / float64(u.m.Vitals.MaxHP)
	}

	mu.gear = g
	mu.stats.Str, mu.stats.Dex, mu.stats.MaxHP, mu.stats.Defense, mu.stats.AR = g.Str, g.Dex, g.MaxHP, g.Defense, g.AR
	mu.stats.DmgMin, mu.stats.DmgMax = g.DmgMin, g.DmgMax

	u.m.Vitals.MaxHP, u.m.Vitals.Defense = g.MaxHP, g.Defense
	u.m.Vitals.A1 = MonsterAttackFrom(g.AR, g.DmgMin, g.DmgMax)

	switch {
	case full:
		u.m.Vitals.HP = g.MaxHP
	case u.m.Alive():
		u.m.Vitals.HP = int(frac * float64(g.MaxHP))
	}
}

func (mu *mercUnit) currentSave(u *unit) MercSave {
	s := mu.save
	s.Dead = !u.m.Alive()

	return s
}

// ReviveMerc brings a dead merc back with full hit points next to the owner
// (MERC_ReviveUnit). The caller charges ReviveCost.
func (d *Director) ReviveMerc(owner *d2mapentity.Player) error {
	u := d.mercs[owner]
	if u == nil {
		return errors.New("no mercenary")
	}

	if u.m.Alive() {
		return errors.New("mercenary is alive")
	}

	u.m.Revive()
	d.applyMercGear(u, true) // MERC_ReviveUnit re-applies the effects of the equipped items (FUN_005752a0)
	u.merc.save.Dead = false
	u.b.Wake = d.frame
	d.teleportNextToOwner(u)

	d.Counters.MercRevives++
	d.emit("merc", "MERC revive name=%s id=%08x level=%d hp=%d/%d cost=%d", d.MercName(u.merc.rec, u.merc.save),
		u.merc.save.ID, u.merc.level, u.m.Vitals.HP, u.m.Vitals.MaxHP, d2hireling.ReviveCost(u.merc.level))

	return nil
}

func (d *Director) teleportNextToOwner(u *unit) bool {
	var owner *d2mapentity.Player
	if u.merc != nil {
		owner = u.merc.owner
	} else {
		owner = u.ally.owner
	}

	ox, oy := playerSubtile(owner)

	p, ok := d2path.NearestFree(d.grid, d2path.MaskMonster, d2path.Point{X: ox + 2, Y: oy}, 10)
	if !ok {
		return false
	}

	u.m.TeleportTo(p.X, p.Y)
	u.mv = nil
	u.b.X, u.b.Y = p.X, p.Y

	return true
}

// ---- d2monster.MercWorld ----

func (d *Director) ownerTargetID(mu *mercUnit) uint32 { return d.playerTargetID(mu.owner) }

func (d *Director) playerTargetID(owner *d2mapentity.Player) uint32 {
	for id, p := range d.targets {
		if p == owner {
			return id
		}
	}

	return 0
}

// Owner implements d2monster.MercWorld. The owner's mode is derived from its
// movement since the last frame (walking, or running when the hero runs).
func (d *Director) Owner(b *d2monster.Brain) (d2monster.OwnerInfo, bool) {
	u := d.unitOf(b)
	if u != nil && u.merc == nil && u.ally != nil && u.ally.owner != nil { // a summoned pet
		x, y := playerSubtile(u.ally.owner)

		return d2monster.OwnerInfo{
			Target: d2monster.Target{ID: d.playerTargetID(u.ally.owner), X: x, Y: y, Size: 1, IsPlayer: true},
			Mode:   d2monster.ModeNeutral,
		}, true
	}

	if u == nil || u.merc == nil || u.merc.owner == nil {
		return d2monster.OwnerInfo{}, false
	}

	mu := u.merc
	x, y := playerSubtile(mu.owner)

	return d2monster.OwnerInfo{
		Target: d2monster.Target{ID: d.ownerTargetID(mu), X: x, Y: y, Size: 1, IsPlayer: true},
		Mode:   mu.ownerMode,
	}, true
}

// Teleport implements d2monster.MercWorld.
func (d *Director) Teleport(b *d2monster.Brain) bool {
	u := d.unitOf(b)
	if u != nil && u.merc == nil && u.ally != nil && u.ally.owner != nil {
		return d.teleportNextToOwner(u)
	}

	if u == nil || u.merc == nil {
		return false
	}

	ok := d.teleportNextToOwner(u)
	if ok {
		d.Counters.MercTeleports++
		d.emit("merc", "MERC follow teleport name=%s id=%08x to=(%d,%d)", u.m.Label(), u.merc.save.ID, u.b.X, u.b.Y)
	}

	return ok
}

// MercLevel implements d2monster.MercWorld.
func (d *Director) MercLevel(b *d2monster.Brain) int {
	if u := d.unitOf(b); u != nil && u.merc != nil {
		return u.merc.level
	}

	return 1
}

// IsRanged implements d2monster.MercWorld.
func (d *Director) IsRanged(b *d2monster.Brain) bool {
	u := d.unitOf(b)

	return u != nil && d.isRanged(u)
}

func (d *Director) isRanged(u *unit) bool {
	if u.merc != nil {
		return u.merc.ranged
	}

	return attackIsRanged(u.m.Stat, d2monster.ModeAttack1)
}

// tableMode converts a hireling.txt ModeN number (1 NU, 4 A1, 5 A2, 7 SC,
// 14 SQ) to a unit mode; instant is true for the neutral (buff) mode.
func tableMode(m int) (mode d2monster.Mode, instant bool) {
	switch m {
	case 1:
		return d2monster.ModeNeutral, true
	case 4:
		return d2monster.ModeAttack1, false
	case 5:
		return d2monster.ModeAttack2, false
	case 8, 9, 10, 11:
		return d2monster.Mode(m), false
	}

	return d2monster.ModeCast, false // 7 (SC) and 14 (SQ)
}

// ChooseAndCast implements d2monster.MercWorld (MERC_ChooseAndQueueSkill):
// the weighted choice comes from d2hireling; skills.txt effects are not
// simulated, so attack and cast modes deal the merc's own damage, buffs only
// set a timed state used by the next choice.
func (d *Director) ChooseAndCast(b *d2monster.Brain, t d2monster.Target, dist int) d2monster.CastResult {
	u := d.unitOf(b)
	if u == nil || u.merc == nil {
		return d2monster.CastFailed
	}

	mu := u.merc

	env := d2hireling.SkillEnv{
		ReqLevel: func(name string) int {
			if rec := d.asset.Records.GetSkillByName(name); rec != nil {
				return rec.Reqlevel
			}

			return 0
		},
		StateActive: func(name string) bool { return mu.buffs[name] > d.frame },
	}

	ch := mu.rec.ChooseSkill(mu.level, env, b.Roll)

	mode, instant := d2monster.ModeAttack1, false
	if !ch.Default {
		mode, instant = tableMode(ch.Skill.Mode)
	}

	if instant {
		mu.buffs[ch.Skill.Name] = d.frame + buffSeconds*25
		d.Counters.MercSkills++
		d.emit("merc", "MERC skill name=%s id=%08x skill=%q level=%d mode=NU buff", u.m.Label(), mu.save.ID, ch.Skill.Name, ch.Level)

		return d2monster.CastInstant
	}

	if !d.InRange(b, t, dist) {
		tt := t

		if d.moveTo(u, d2monster.Point{X: t.X, Y: t.Y}, &tt, meleeInRange-1, false) {
			return d2monster.CastBusy
		}

		return d2monster.CastFailed
	}

	mu.skill = ch.Skill.Name
	if ch.Default {
		mu.skill = ""
	} else {
		d.Counters.MercSkills++
		d.emit("merc", "MERC skill name=%s id=%08x skill=%q level=%d mode=%s", u.m.Label(), mu.save.ID, ch.Skill.Name, ch.Level, mode)
	}

	if d.Attack(b, mode, t) {
		return d2monster.CastBusy
	}

	return d2monster.CastFailed
}

// ---- per-frame upkeep ----

// stepMerc updates the owner-derived state before the AI runs.
func (d *Director) stepMerc(u *unit) {
	mu := u.merc
	x, y := playerSubtile(mu.owner)

	switch {
	case x == mu.lastOwnerX && y == mu.lastOwnerY:
		mu.ownerMode = d2monster.ModeNeutral
	case mu.owner.IsRunning():
		mu.ownerMode = d2monster.ModeRun
	default:
		mu.ownerMode = d2monster.ModeWalk
	}

	mu.lastOwnerX, mu.lastOwnerY = x, y

	if u.m.Alive() {
		mu.stepRegen(&u.m.Vitals)
	}
}

// nearestEnemy is the merc's attack target: the nearest living hostile
// monster (UNVERIFIED filter; the exe uses FUN_005dc370/FUN_005dbe00).
func (d *Director) nearestEnemy(b *d2monster.Brain) (d2monster.Target, int, bool) {
	var (
		best     d2monster.Target
		bestDist int
		found    bool
	)

	for _, o := range d.sortedUnits() {
		if o.friendly() || o.b.Allied || !o.m.Alive() || o.b == b {
			continue
		}

		x, y := o.m.SubtilePos()
		dist := d2monster.EdgeDistance(b.X-x, b.Y-y, b.Size)

		// a mercenary names its enemies by plain unit id, a converted monster by
		// the unit-target offset (the plain ids below mercTargetBase are players)
		id := o.b.ID
		if me := d.unitOf(b); me == nil || (me.merc == nil && me.ally == nil) {
			id += unitTargetBase
		}

		if !found || dist < bestDist {
			best, bestDist, found = d2monster.Target{ID: id, X: x, Y: y, Size: 1}, dist, true
		}
	}

	return best, bestDist, found
}

// mercStrike resolves a merc attack at its animation's hit frame.
func (d *Director) mercStrike(u *unit, mode d2monster.Mode) {
	mu := u.merc

	tu := d.units[u.attackTarget]
	if tu == nil || tu.friendly() || tu.b.Allied || !tu.m.Alive() {
		return
	}

	tx, ty := tu.m.SubtilePos()
	sx, sy := u.m.SubtilePos()
	dist := d2monster.EdgeDistance(sx-tx, sy-ty, u.b.Size)

	reach := meleeInRange
	if mu.ranged {
		reach = rangedInRange
	}

	d.Counters.MercAttacks++

	if dist > reach+heroReach/2 {
		d.emit("merc", "MERC attack name=%s id=%08x mode=%s hit=false reason=target_moved_away dist=%d", u.m.Label(), mu.save.ID, mode, dist)

		return
	}

	in := d2combat.ToHitInput{
		AttackRating:  d2combat.MonsterAttackRating(mu.stats.AR, 0, mu.stats.Dex),
		Defense:       tu.m.Vitals.Defense,
		AttackerLevel: mu.level,
		DefenderLevel: tu.m.Vitals.Level,
	}

	hit, chance, roll := d2combat.RollToHit(u.b.Seed, in)
	if !hit {
		d.emit("merc", "MERC attack name=%s id=%08x target=%s mode=%s hit=false chance=%d roll=%d", u.m.Label(), mu.save.ID,
			tu.m.Label(), mode, chance, roll)

		return
	}

	dmg := mu.stats.DmgMin + int(u.b.Seed.Roll(int32(mu.stats.DmgMax-mu.stats.DmgMin+1)))
	if dmg < 1 {
		dmg = 1
	}

	d.Counters.MercHits++
	d.emit("merc", "MERC attack name=%s id=%08x target=%s mode=%s hit=true chance=%d roll=%d dmg=%d", u.m.Label(), mu.save.ID,
		tu.m.Label(), mode, chance, roll, dmg)

	// VERIFIED (0x579d70): hireling hits on a boss record are scaled by the
	// difficulty's HireableBossDamagePercent.
	if tu.m.Stat != nil && tu.m.Stat.IsSpecialBoss {
		dmg = d2hireling.BossDamage(dmg, d2difficulty.Default.Row(d2difficulty.Level(d.opt.Difficulty)).HireableBossDamagePercent)
	}

	d.killer = u
	d.damage(tu, mu.owner, dmg)
	d.killer = nil
}

// strikeMerc resolves a monster's attack on a merc.
func (d *Director) strikeMerc(u *unit, tu *unit, atk d2mapentity.MonsterAttack, mode d2monster.Mode) {
	tx, ty := tu.m.SubtilePos()
	sx, sy := u.m.SubtilePos()
	dist := d2monster.EdgeDistance(sx-tx, sy-ty, u.b.Size)

	reach := meleeInRange
	if u.m.Stat.IsRanged {
		reach = rangedInRange
	}

	d.Counters.Attacks++

	if dist > reach+heroReach/2 {
		d.emit("attack", "MONSTER attack name=%s id=%d mode=%s hit=false reason=target_moved_away dist=%d", u.m.Label(), u.b.ID, mode, dist)

		return
	}

	hit, chance, roll, dmg := d.rollMercHit(u, tu, atk)

	if hit {
		d.Counters.AttackHits++
	}

	d.emit("attack", "MONSTER attack name=%s id=%d mode=%s target=merc hit=%v chance=%d roll=%d dmg=%d merc_hp=%d/%d def=%d",
		u.m.Label(), u.b.ID, mode, hit, chance, roll, dmg, maxInt(tu.m.Vitals.HP-dmg, 0), tu.m.Vitals.MaxHP, tu.m.Vitals.Defense)

	if hit {
		d.damageMerc(tu, dmg, u.m.Label())
	}
}

// rollMercHit rolls a monster attack against a merc: the to-hit against the merc's total
// defense (table plus gear, kept in its vitals), then the attack's damage after the merc's
// gear (flat reduction, physical resist; both are no-ops without gear).
func (d *Director) rollMercHit(u, tu *unit, atk d2mapentity.MonsterAttack) (hit bool, chance, roll, dmg int) {
	in := d2combat.ToHitInput{
		AttackRating:  d2combat.MonsterAttackRating(atk.ToHit, 0, 0),
		Defense:       tu.m.Vitals.Defense,
		AttackerLevel: u.m.Vitals.Level,
		DefenderLevel: tu.merc.level,
	}

	hit, chance, roll = d2combat.RollToHit(u.b.Seed, in)
	if !hit {
		return hit, chance, roll, 0
	}

	dmg = atk.Min + int(u.b.Seed.Roll(int32(atk.Max-atk.Min+1)))
	if dmg < 1 {
		dmg = 1
	}

	return hit, chance, roll, d.takenByMerc(tu, dmg)
}

func (d *Director) damageMerc(tu *unit, dmg int, by string) {
	if !tu.m.Alive() {
		return
	}

	tu.m.Vitals.HP -= dmg

	if tu.m.Vitals.HP > 0 {
		tu.m.StopMoving()
		tu.m.DropHitEvents()
		tu.mv = nil

		if tu.m.SetMode(d2monster.ModeGetHit) {
			tu.b.Wake = 1 << 30
		} else {
			tu.b.WakeNow(d.frame)
		}

		return
	}

	tu.m.Vitals.HP = 0
	tu.m.Die()
	d.fp.Remove(tu.b.ID)
	tu.mv = nil
	tu.merc.save.Dead = true
	d.Counters.MercDeaths++
	d.emit("merc", "MERC death name=%s id=%08x level=%d by=%s revive_cost=%d", tu.m.Label(), tu.merc.save.ID, tu.merc.level, by,
		d2hireling.ReviveCost(tu.merc.level))
}

// mercGetsKillXP is the gate of MERC_AwardExperience (0x0057c860, hirelings.md,
// VERIFIED): only a merc below its owner's level and below the table maximum
// gains experience from a kill.
func mercGetsKillXP(mercLevel, ownerLevel int) bool {
	return mercLevel < ownerLevel && mercLevel < d2hireling.MaxLevel
}

// creditMerc is MERC_AwardExperience: a merc that helped kill a monster and
// is below its owner's level gains experience and levels up.
func (d *Director) creditMerc(mu *mercUnit, u *unit, xp int) {
	if !mercGetsKillXP(mu.level, mu.owner.Stats.Level) {
		return
	}

	mu.save.Experience += uint32(xp * mercExpMultiplier)

	nl := d.hire.LevelAfterGain(int(mu.save.Type), mu.level, mu.save.Experience)
	d.emit("merc", "MERC exp name=%s id=%08x gained=%d exp=%d", u.m.Label(), mu.save.ID, xp*mercExpMultiplier, mu.save.Experience)

	if nl > mu.level {
		st, rec := d.hire.StatsFor(int(mu.save.Type), nl)
		if rec == nil {
			return
		}

		mu.level, mu.stats, mu.base, mu.rec = nl, st, st, rec
		u.m.Vitals.Level = nl
		d.applyMercGear(u, true) // the gear stays on: table stats of the new level plus the items
		st = mu.stats
		d.Counters.MercLevelUps++
		d.emit("merc", "MERC levelup name=%s id=%08x level=%d hp=%d defense=%d dmg=%d-%d", u.m.Label(), mu.save.ID, nl, st.MaxHP,
			st.Defense, st.DmgMin, st.DmgMax)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// Difficulty is the difficulty the director runs at.
func (d *Director) Difficulty() d2monster.Difficulty { return d.opt.Difficulty }

// MercPosition is the merc's subtile position (0,0 without a merc).
func (d *Director) MercPosition(owner *d2mapentity.Player) (x, y int) {
	if u := d.mercs[owner]; u != nil {
		return u.m.SubtilePos()
	}

	return 0, 0
}

// KillMerc kills the owner's merc at once (scenario and tests).
func (d *Director) KillMerc(owner *d2mapentity.Player, by string) {
	if u := d.mercs[owner]; u != nil {
		d.damageMerc(u, u.m.Vitals.HP, by)
	}
}

// FarPoint finds a point about dist subtiles from the owner that a path leads
// to, trying eight directions (scenario helper). The result is in tiles, as
// the game's move command takes.
func (d *Director) FarPoint(owner *d2mapentity.Player, dist int) (x, y float64, ok bool) {
	ox, oy := playerSubtile(owner)
	dirs := [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {-1, 1}, {-1, -1}, {1, -1}}

	for _, dd := range dirs {
		goal := d2path.Point{X: ox + dd[0]*dist, Y: oy + dd[1]*dist}

		p, found := d2path.NearestFree(d.grid, d2path.MaskMonster, goal, 3)
		if !found {
			continue
		}

		route, okp := d2path.FindPath(d.grid, d2path.MaskMonster, d2path.Point{X: ox, Y: oy}, p)
		if okp && len(route.Nodes) > 0 {
			return float64(p.X) / subtilesPerTile, float64(p.Y) / subtilesPerTile, true
		}
	}

	return 0, 0, false
}

// GrantMercExp adds experience to the owner's merc without the owner-level
// limit (scenario helper).
func (d *Director) GrantMercExp(owner *d2mapentity.Player, exp uint32) {
	u := d.mercs[owner]
	if u == nil {
		return
	}

	u.merc.save.Experience += exp
	d.emit("merc", "MERC exp granted name=%s id=%08x exp=%d", u.m.Label(), u.merc.save.ID, u.merc.save.Experience)
}
