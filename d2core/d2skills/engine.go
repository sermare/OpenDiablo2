package d2skills

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const (
	logPrefix       = "Skills"
	frameSeconds    = 1.0 / 25.0
	maxCatchUp      = 0.25
	subtilesPerTile = 5
	pickRadius      = 6 // subtiles around the cursor in which a melee skill finds its target
)

// Options configure an Engine.
type Options struct {
	Seed uint32
	// IgnoreTown lets skills be cast in town (scenarios).
	IgnoreTown bool
	// InfiniteAmmo makes arrow skills find ammunition (the quiver is not
	// modelled).
	InfiniteAmmo bool
}

// Counters tally what happened, for scenario summaries.
type Counters struct {
	Casts, Refused, Missiles, Hits, Misses, Walls, Expired, Kills, Melee int
	// AreaHits are targets reached by area effects and splash, DotDamage the
	// poison and burn damage dealt.
	AreaHits, DotDamage int
	ManaSpent           int // 8.8
	Damage              int // whole points dealt after resists
}

type timer struct {
	frame int
	fn    func()
}

// Engine is the in-engine skill runner.
type Engine struct {
	*d2util.Logger

	asset     *d2asset.AssetManager
	mapEngine *d2mapengine.MapEngine
	monsters  *d2monsters.Director
	opt       Options

	pipe *d2skill.Pipeline
	sim  *d2missile.Sim

	frame int
	acc   float64

	heroes  map[string]*heroUnit
	targets map[string]*monsterTarget
	visuals map[uint32]*d2mapentity.Missile
	fx      map[*d2mapentity.Missile]int // explosion entities and the frame they vanish at
	sets    map[string]*d2state.Set      // unit id -> states and DoT streams
	defs    d2state.Defs                 // states.txt rules shared by the sets
	timers  []timer

	auras  map[string]*auraRun // hero id -> the aura it keeps on
	storms []*stormRun
	traps  []*trapRun

	fakeCaster trapCaster                        // tests only
	pets       map[string][]*d2mapentity.Monster // hero id -> summons by pet type (see summon.go)
	watches    []*watch
	dots       map[string]dotTotal

	// Counters are updated as events happen.
	Counters Counters
	// OnEvent receives every log line as a structured event (kind cast, mana,
	// missile, hit, damage, state...).
	OnEvent func(kind, line string)
}

// New creates an engine for a map. monsters supplies targets and the grid.
func New(asset *d2asset.AssetManager, mapEngine *d2mapengine.MapEngine, monsters *d2monsters.Director,
	l d2util.LogLevel, opt Options) *Engine {
	e := &Engine{
		Logger: d2util.NewLogger(), asset: asset, mapEngine: mapEngine, monsters: monsters, opt: opt,
		heroes: map[string]*heroUnit{}, targets: map[string]*monsterTarget{},
		visuals: map[uint32]*d2mapentity.Missile{}, fx: map[*d2mapentity.Missile]int{},
		sets: map[string]*d2state.Set{}, auras: map[string]*auraRun{}, pets: map[string][]*d2mapentity.Monster{}, dots: map[string]dotTotal{},
	}

	e.Logger.SetLevel(l)
	e.Logger.SetPrefix(logPrefix)

	w := &world{e: e}
	e.sim = d2missile.NewSim(w, asset.Records.MissileTable())
	e.sim.OnEvent = e.onSim
	e.pipe = &d2skill.Pipeline{
		Skills: asset.Records.SkillTable(), Missiles: asset.Records.MissileTable(), Sim: e.sim,
		Grid: monsters.Grid(), Frame: func() int { return e.frame },
		Opt: d2skill.Options{IgnoreTown: opt.IgnoreTown, StaticFieldMinPct: staticFieldMin(asset, monsters)},
	}
	e.pipe.ApplyState = e.applyMissileState
	e.pipe.Near = e.near
	e.pipe.After = e.after
	e.pipe.Walkable = func(x, y int) bool {
		return e.monsters.Grid().Flags(x, y)&(d2path.FlagWalk|d2path.FlagWall) == 0
	}
	monsters.HeroDefense = e.heroDefense
	monsters.HeroAvoid = e.heroAvoid

	return e
}

// staticFieldMin reads DifficultyLevels StaticFieldMin for normal difficulty
// (the scenario difficulty is not tracked here); 0 when the table has no row.
func staticFieldMin(asset *d2asset.AssetManager, _ *d2monsters.Director) int {
	if rec := asset.Records.DifficultyLevels[d2enum.DifficultyNormal]; rec != nil {
		return rec.StaticFieldMin
	}

	return 0
}

func (e *Engine) emit(kind, format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	e.Info(line)

	if e.OnEvent != nil {
		e.OnEvent(kind, line)
	}
}

// Frame returns the simulation frame.
func (e *Engine) Frame() int { return e.frame }

// Pipeline exposes the pure cast pipeline (tests).
func (e *Engine) Pipeline() *d2skill.Pipeline { return e.pipe }

// SkillID resolves a skill name (case-insensitive) or numeric id; -1 if unknown.
func (e *Engine) SkillID(ref string) int {
	if sk := e.pipe.Skills.ByName(strings.TrimSpace(ref)); sk != nil {
		return sk.ID
	}

	var id int
	if _, err := fmt.Sscanf(ref, "%d", &id); err == nil && e.pipe.Skills.ByID(id) != nil {
		return id
	}

	return -1
}

// Supported reports whether the pipeline handles a skill (otherwise the caller
// should fall back to the old client-only cast).
func (e *Engine) Supported(skillID int) bool {
	return d2skill.Implemented(e.pipe.Skills.ByID(skillID))
}

// HasState reports whether a unit currently has a state.
func (e *Engine) HasState(unitID, state string) bool {
	return e.setOf(unitID).Active(e.frame, state)
}

// StateStats is the sum of the stat mods of the unit's active states, by
// ItemStatCost name: the aurastat1..6 lists of its buffs and auras (each
// statlist adds to the unit's stats, verified at 0x5c4c60). Passive skills
// are not included.
func (e *Engine) StateStats(unitID string) map[string]int {
	st := e.sets[unitID]
	if st == nil {
		return nil
	}

	return st.StatMods(e.frame)
}

// PassiveTotals returns the stats a hero's true passives (passivestate set,
// verified at 0x648130) add to its totals, ItemStatCost name -> value. Stats
// keyed to a weapon type (passiveitype, the masteries) are left out: the exe
// keys them by item type and they are not part of the plain totals.
func (e *Engine) PassiveTotals(unitID string) map[string]int {
	h := e.heroes[unitID]
	if h == nil || h.inPassive {
		return nil
	}

	out := map[string]int{}

	h.inPassive = true
	for id, s := range h.p.Skills {
		if s == nil || s.SkillPoints < 1 {
			continue
		}

		for _, m := range e.pipe.TruePassiveStats(h, id) {
			if m.Param == "" && m.Value != 0 {
				out[m.Stat] += m.Value
			}
		}
	}
	h.inPassive = false

	return out
}

// stateDefs converts the states.txt records into the rules the state sets use
// (cached; nil when the records have no states table).
func (e *Engine) stateDefs() d2state.Defs {
	if e.defs != nil || e.asset == nil || len(e.asset.Records.States) == 0 {
		return e.defs
	}

	e.defs = d2state.Defs{}

	for name, r := range e.asset.Records.States {
		e.defs[name] = d2state.Def{
			ID: r.ID, Name: name, Group: r.Group, Curse: r.Curse, RemHit: r.RemHit, Aura: r.Aura,
			PlrStayDeath: r.PlrStayDeath, MonStayDeath: r.MonStayDeath, BossStayDeath: r.BossStayDeath,
			Shatter: r.Shatter, Blue: r.Blue, ColorPri: r.ColorPri, ColorShift: r.ColorShift,
			Overlay1: r.Overlay1, Stat: r.Stat,
		}
	}

	return e.defs
}

// setOf returns the state set of a unit, creating it.
func (e *Engine) setOf(id string) *d2state.Set {
	st := e.sets[id]
	if st == nil {
		st = d2state.New()
		st.SetDefs(e.stateDefs())
		e.sets[id] = st
	}

	return st
}

func (e *Engine) after(frames int, fn func()) {
	e.timers = append(e.timers, timer{frame: e.frame + frames, fn: fn})
}

// ---- casting ----

// Cast starts a skill for the hero aimed at a point in tile coordinates (what
// OnPlayerCast receives). It returns false when the skill is not handled by
// the pipeline or was refused; a refused cast is logged.
func (e *Engine) Cast(p *d2mapentity.Player, skillID int, tileX, tileY float64) bool {
	return e.CastAt(p, skillID, int(math.Floor(tileX*subtilesPerTile)), int(math.Floor(tileY*subtilesPerTile)))
}

// CastAt is Cast with a subtile aim point.
func (e *Engine) CastAt(p *d2mapentity.Player, skillID, sx, sy int) bool {
	sk := e.pipe.Skills.ByID(skillID)
	if !d2skill.Implemented(sk) {
		return false
	}

	u := e.hero(p)
	tg := e.targetAt(sx, sy)
	st := e.pipe.Start(u, skillID, tg)

	e.Counters.Casts++
	e.emit("cast", "CAST start skill=%q id=%d level=%d aim=(%d,%d) ok=%v reason=%s mana_paid=%.2f mana=%s",
		sk.Name, skillID, st.Level, sx, sy, st.OK, reasonOf(st.Reason), fixed(st.ManaPaid), u.manaString())

	if !st.OK {
		e.Counters.Refused++
		return false
	}

	e.Counters.ManaSpent += st.ManaPaid
	p.SetDirection(p.Position.DirectionTo(*d2vector.NewVector(float64(sx), float64(sy))))

	e.castOverlay(p, e.asset.Records.Skill.Details[skillID])

	run := func() { e.runDo(p, u, sk, tg) }

	rec := e.asset.Records.Skill.Details[skillID]
	if rec == nil || rec.Anim == d2enum.PlayerAnimationModeNone {
		run()
		return true
	}

	if rec.Anim == d2enum.PlayerAnimationModeAttack1 {
		p.StartAttack(run)
	} else {
		p.StartCasting(rec.Anim, run)
	}

	return true
}

func reasonOf(r string) string {
	if r == "" {
		return "-"
	}

	return r
}

func fixed(v int) float64 { return float64(v) / 256 }

// targetAt builds the aim: the point, and the nearest living monster around it.
func (e *Engine) targetAt(sx, sy int) d2skill.Target {
	tg := d2skill.Target{X: sx, Y: sy}

	cbest := pickRadius + 1

	for _, m := range e.monsters.Corpses() {
		mx, my := m.SubtilePos()
		if d := chebyshev(mx-sx, my-sy); d < cbest {
			cbest = d
			tg.Corpse, tg.CX, tg.CY, tg.CorpseID, tg.CorpseHP, tg.CorpseKey = true, mx, my, m.ID(), m.Vitals.MaxHP, m.Stat.Key
		}
	}

	best := pickRadius + 1

	for _, m := range e.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if d := chebyshev(mx-sx, my-sy); d < best {
			best = d
			tg.Unit, tg.UX, tg.UY = e.target(m), mx, my
		}
	}

	return tg
}

func chebyshev(dx, dy int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dx > dy {
		return dx
	}

	return dy
}

// runDo is the action frame of the cast animation.
func (e *Engine) runDo(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, tg d2skill.Target) {
	// the target may have moved while the hero played the animation
	if tg.Unit != nil {
		if mt, ok := tg.Unit.(*monsterTarget); ok {
			tg.UX, tg.UY = mt.m.SubtilePos()
		}
	}

	res := e.pipe.Do(u, sk.ID, tg)

	e.emit("cast", "CAST do skill=%q level=%d ok=%v reason=%s missiles=%d melee=%v effects=%d mana_paid=%.2f cooldown=%d mana=%s",
		sk.Name, res.Level, res.OK, reasonOf(res.Reason), len(res.Missiles), res.Melee != nil, len(res.Effects),
		fixed(res.ManaPaid), res.Cooldown, u.manaString())

	if !res.OK {
		e.Counters.Refused++
		return
	}

	e.Counters.ManaSpent += res.ManaPaid

	if res.ManaPaid > 0 {
		e.emit("mana", "MANA paid skill=%q cost=%.2f left=%s", sk.Name, fixed(res.ManaPaid), u.manaString())
	}

	for _, m := range res.Melees {
		e.meleeResult(p, sk, m)
	}

	if len(res.Melees) == 0 && res.Melee != nil {
		e.meleeResult(p, sk, res.Melee)
	}

	for i := range res.Effects {
		e.effect(p, u, sk, &res.Effects[i])
	}
}

func (e *Engine) castOverlay(p *d2mapentity.Player, rec *d2records.SkillRecord) {
	if rec == nil || rec.Castoverlay == "" {
		return
	}

	ov := e.asset.Records.Layout.Overlays[rec.Castoverlay]
	if ov == nil {
		return
	}

	ent, err := e.mapEngine.NewCastOverlay(int(p.Position.X()), int(p.Position.Y()), ov)
	if err != nil {
		return
	}

	ent.SetOnDoneFunc(func() { e.mapEngine.RemoveEntity(ent) })
	e.mapEngine.AddEntity(ent)
}

// ---- results ----

func (e *Engine) meleeResult(p *d2mapentity.Player, sk *d2skill.Skill, r *d2skill.MeleeResult) {
	e.Counters.Melee++

	mt, _ := r.Target.(*monsterTarget)
	name := "?"

	if mt != nil && !mt.m.Alive() { // an earlier strike of the same cast killed it
		return
	}

	if mt != nil {
		name = mt.m.Label()
	}

	if !r.Hit {
		e.Counters.Misses++
		e.emit("hit", "SKILL melee skill=%q target=%s hit=false chance=%d roll=%d", sk.Name, name, r.Chance, r.Roll)

		return
	}

	e.emit("hit", "SKILL melee skill=%q target=%s hit=true chance=%d roll=%d %s", sk.Name, name, r.Chance, r.Roll,
		describe(&r.Damage))

	if mt != nil {
		e.Counters.Hits++
		e.hurt(mt.m, p, &r.Damage, sk.Name)
		e.itemEvents(mt.m, p, true) // crushing blow, open wounds: after the base damage
	}
}

// staticField is SRVDO_StaticField's callback (U): every monster in the
// radius loses Pct percent of its current life, never going below FloorPct of
// its maximum life, and at least MinDamage when it can afford it.
func (e *Engine) staticField(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	hx, hy := u.Pos()
	n := 0

	for _, m := range e.monsters.Monsters() {
		mx, my := m.SubtilePos()
		if !m.Alive() || chebyshev(mx-hx, my-hy) > ef.Radius {
			continue
		}

		v := &m.Vitals
		dmg := v.HP * ef.Pct / 100

		if dmg < ef.MinDamage {
			dmg = ef.MinDamage
		}

		if floor := v.MaxHP * ef.FloorPct / 100; v.HP-dmg < floor {
			dmg = v.HP - floor
		}

		if dmg < 0 {
			dmg = 0
		}

		res := e.resistFrom(m, p, "ltng")
		dmg, _ = d2combat.ReduceComponent(dmg, 0, res, false, false, 0, 0)
		n++

		e.emit("damage", "SKILL static_field target=%s pct=%d floor=%d%% resist=%d dmg=%d hp=%d/%d", m.Label(), ef.Pct,
			ef.FloorPct, res, dmg, v.HP, v.MaxHP)

		if dmg > 0 {
			e.Counters.Damage += dmg
			e.monsters.Damage(m, dmg, p)

			if !m.Alive() {
				e.Counters.Kills++
				e.setOf(m.ID()).Death("monster")
			}
		}
	}

	e.emit("state", "STATE area skill=%q radius=%d affected=%d", sk.Name, ef.Radius, n)
}

func (e *Engine) applyMissileState(owner d2skill.Unit, t d2missile.Target, state string, frames int) {
	name := t.ID()

	if mt, ok := t.(*monsterTarget); ok {
		name = mt.m.Label()
		e.applyMonsterState(mt.m, d2state.Instance{Name: state, Until: e.frame + frames, Source: owner.ID()})
	} else {
		e.setOf(t.ID()).Apply(e.frame, d2state.Instance{Name: state, Until: e.frame + frames, Source: owner.ID()})
	}

	e.emit("state", "STATE apply by=%s unit=%s state=%s frames=%d", owner.ID(), name, state, frames)
}

// ---- damage ----

// resistOf returns the monster's resist percent for a damage type.
func (e *Engine) resist(m *d2mapentity.Monster, kind string) int {
	return e.resistFrom(m, nil, kind)
}

// pierceOf is the attacker's pierce percent against a damage kind: the
// passive pierce stats 333..336. VERIFIED (0x579b10 descriptor table at
// 0x72ff38): each damage type names exactly one pierce stat, 333 fire, 334
// lightning, 335 cold, 336 poison, and the function reads nothing else. The
// item stats 305..308 are NOT added (they are never consulted there), so they
// have no effect on resists here.
func pierceOf(src *d2mapentity.Player, kind string) (pierce int, has bool) {
	ids := map[string]int{"fire": 333, "ltng": 334, "cold": 335, "pois": 336}

	id, ok := ids[kind]
	if !ok {
		return 0, false // physical and magic have no pierce stat
	}

	if src == nil || src.Stats == nil || src.Stats.Totals == nil || src.Stats.Totals.Stats == nil {
		return 0, true
	}

	l := src.Stats.Totals.Stats

	return int(l.Get(id)), true
}

// resistFrom is resist with the attacker's pierce (src may be nil).
func (e *Engine) resistFrom(m *d2mapentity.Monster, src *d2mapentity.Player, kind string) int {
	s := m.Stat
	diff := int(m.Vitals.Difficulty)
	pick := func(n, nm, h int) int { return [3]int{n, nm, h}[diff] }

	var res int

	phys := false
	magic := false

	switch kind {
	case "phys":
		phys = true
		res = pick(s.ResistancePhysicalNormal, s.ResistancePhysicalNightmare, s.ResistancePhysicalHell)
	case "mag":
		magic = true
		res = pick(s.ResistanceMagicNormal, s.ResistanceMagicNightmare, s.ResistanceMagicHell)
	case "fire":
		res = pick(s.ResistanceFireNormal, s.ResistanceFireNightmare, s.ResistanceFireHell)
	case "ltng":
		res = pick(s.ResistanceLightningNormal, s.ResistanceLightningNightmare, s.ResistanceLightningHell)
	case "cold":
		res = pick(s.ResistanceColdNormal, s.ResistanceColdNightmare, s.ResistanceColdHell)
	case "pois":
		res = pick(s.ResistancePoisonNormal, s.ResistancePoisonNightmare, s.ResistancePoisonHell)
	}

	// no difficulty penalty: it applies to player defenders (inferred)
	_ = magic

	// curses and auras on the monster (Amplify Damage is damageresist -100)
	res += e.setOf(m.ID()).ResistDelta(e.frame, kind)

	pierce, hasPierce := pierceOf(src, kind)

	// VERIFIED (0x579b10 ctx[5]): a non-mercenary monster defender sets the one
	// ignore flag: no cap (a monstats resist of 100 is an immunity), no
	// difficulty penalty, and pierce cannot lower a resist of 100 or more.
	return d2combat.EffectiveResist(d2combat.ResistInput{
		Resist: res, IsPhysical: phys, NoDifficultyPenalty: true, Ignore: true, Pierce: pierce, HasPierce: hasPierce,
		// VERIFIED 0x579b10: attacker state 0x2f (sanctuary) vs an undead (lUndead/hUndead, helper 0x63f9e0) defender zeroes positive physical resist
		ZeroPhysical: phys && e.physNullified(m, src),
	})
}

// hurt applies a rolled damage struct to a monster: resists per type, then
// whole hit points through the monster director.
func (e *Engine) hurt(m *d2mapentity.Monster, src *d2mapentity.Player, d *d2combat.Damage, what string) {
	parts := []struct {
		kind string
		v    int32
	}{{"phys", d.Physical}, {"fire", d.Fire}, {"ltng", d.Lightning}, {"mag", d.Magic}, {"cold", d.Cold}}

	var total int

	for _, p := range parts {
		if p.v > 0 {
			// per type: flat reduction, percent resist, absorb (0x579c90). Monsters
			// have no stat 34/35 or absorb stats; components are not floored, the
			// Total is (applied below).
			out, _ := d2combat.ReduceComponent(int(p.v), 0, e.resistFrom(m, src, p.kind), false, false, 0, 0)
			total += out
		}
	}

	whole := 0
	if d2combat.ApplicableTotal(int32(total)) {
		whole = (total + 128) >> 8
	}

	if total > 0 && whole < 1 {
		whole = 1
	}

	// poison and burn are damage over time: the Damage struct holds the
	// per-frame 8.8 value, spread over the length
	set := e.setOf(m.ID())
	cannotCold := e.resistFrom(m, src, "cold") >= d2combat.ImmuneResist
	h := d2state.Hit{
		ColdLen: int(d.ColdLen), FreezeLen: int(d.FreezeLen), StunLen: int(d.StunLen), Source: e.sourceID(src),
		Poison: d2combat.ApplyResist(int(d.Poison), e.resistFrom(m, src, "pois")), PoisonLen: int(d.PoisonLen),
		Burn: d2combat.ApplyResist(int(d.Burn), e.resistFrom(m, src, "fire")), BurnLen: int(d.BurnLen),
		CannotChill: cannotCold, CannotFreeze: cannotCold,
	}
	h.ColdEffect, h.HasColdEffect = coldEffect(m), true
	h.ChillDiv, h.FreezeDiv = e.coldDivisors(m)

	// a hit ends the states flagged remhit (states.txt)
	set.Hit(e.frame)

	hp0 := m.Vitals.HP
	e.Counters.Damage += whole
	e.emit("damage", "DAMAGE skill=%q target=%s raw=%.2f after_resist=%.2f dmg=%d hp=%d->%d/%d", what, m.Label(),
		fixed(int(d.Physical+d.Fire+d.Lightning+d.Magic+d.Cold)), fixed(total), whole, hp0, maxInt(hp0-whole, 0), m.Vitals.MaxHP)

	if whole > 0 {
		e.monsters.Damage(m, whole, src)
		e.afterHit(m, src, whole)
	}

	if !m.Alive() {
		set.Death("monster")
		e.Counters.Kills++
		e.emit("damage", "KILL skill=%q target=%s", what, m.Label())

		return
	}

	if applied := set.ApplyHit(e.frame, h); len(applied) > 0 {
		e.emit("state", "STATE hit skill=%q unit=%s applied=%v stun=%df freeze=%df chill=%df poison=%.2f/f x%df burn=%.2f/f x%df",
			what, m.Label(), applied, h.StunLen, h.FreezeLen, h.ColdLen, fixed(h.Poison), h.PoisonLen, fixed(h.Burn), h.BurnLen)
		e.syncMonster(m)
	}
}

// coldDivisors are the DifficultyLevels MonsterColdDivisor and
// MonsterFreezeDivisor (record +0x18 and +0x14, verified) of the monster's
// difficulty; 0 when the table is not loaded.
func (e *Engine) coldDivisors(m *d2mapentity.Monster) (chill, freeze int) {
	if e.asset == nil {
		return 0, 0
	}

	return divisorsFor(e.asset.Records.DifficultyLevels, int(m.Vitals.Difficulty))
}

func divisorsFor(recs d2records.DifficultyLevels, diff int) (chill, freeze int) {
	if rec := recs[d2enum.DifficultyType(diff)]; rec != nil {
		return rec.MonsterColdDivisor, rec.MonsterFreezeDivisor
	}

	return 0, 0
}

// HeroDied clears the states of a dying hero: the statlists without
// plrstaydeath end, and so do the DoT streams (verified, 0x57d310).
//
// An aura the hero keeps on stops pulsing when its state went with the
// states; otherwise the next pulse would give the dead hero the state back
// (an aura with plrstaydeath keeps running). What the original does with the
// hero's summons, storms, traps and missiles at his death is UNVERIFIED and
// left alone.
func (e *Engine) HeroDied(id string) {
	e.setOf(id).Death("player")

	if a := e.auras[id]; a != nil && !e.setOf(id).Active(e.frame, a.ef.State) {
		delete(e.auras, id)
	}
}

// AreaChanged tells the engine the hero moved to another area, whose units
// belong to md (nil in tests). Everything tied to units or missiles of the
// old area is dropped: missiles in flight, storms, traps and totem pulses,
// pending timers, summon bookkeeping, monster targets and the states of
// monsters (their ids may be reused by the new area). The heroes keep their
// states, auras, mana and cooldowns. Whether summons follow the hero through
// a portal or stairs is UNVERIFIED (mercenaries do, see Director.SpawnMerc);
// the new Director starts without them.
func (e *Engine) AreaChanged(md *d2monsters.Director) {
	if e.sim != nil {
		e.sim.Clear()
	}

	for id, ent := range e.visuals {
		if ent != nil && e.mapEngine != nil {
			e.mapEngine.RemoveEntity(ent)
		}

		delete(e.visuals, id)
	}

	for ent := range e.fx {
		if e.mapEngine != nil {
			e.mapEngine.RemoveEntity(ent)
		}

		delete(e.fx, ent)
	}

	e.storms, e.traps, e.watches, e.timers = nil, nil, nil, nil
	e.pets = map[string][]*d2mapentity.Monster{}
	e.targets = map[string]*monsterTarget{}
	e.dots = map[string]dotTotal{}

	for id := range e.sets {
		if e.heroes[id] == nil {
			delete(e.sets, id)
		}
	}

	if md != nil {
		e.monsters = md
		e.pipe.Grid = md.Grid()
		md.HeroDefense = e.heroDefense
	}
}

// Monsters returns the Director the engine is bound to.
func (e *Engine) Monsters() *d2monsters.Director { return e.monsters }

// coldEffect is the monstats ColdEffect of a monster for its difficulty
// (negative = slow percent, 0 = cannot be chilled).
func coldEffect(m *d2mapentity.Monster) int {
	s := m.Stat

	return [3]int{s.ColdSensitivityNormal, s.ColdSensitivityNightmare, s.ColdSensitivityHell}[int(m.Vitals.Difficulty)]
}

// afterHit runs the effects of curses on a monster that was hurt by the hero:
// Iron Maiden (the hero takes a share of the damage), Life Tap (the hero
// heals a share).
func (e *Engine) afterHit(m *d2mapentity.Monster, src *d2mapentity.Player, dmg int) {
	if src == nil {
		return
	}

	set := e.setOf(m.ID())

	if pct := set.ReflectPct(e.frame); pct > 0 {
		back := dmg * pct / 100
		src.Stats.Health -= back

		if src.Stats.Health < 0 {
			src.Stats.Health = 0
		}

		e.emit("damage", "DAMAGE iron_maiden hero=%s dmg=%d hero_hp=%d/%d", src.Name(), back, src.Stats.Health, src.Stats.MaxHealth)
	}

	if pct := set.LifeTapPct(e.frame); pct > 0 {
		heal := dmg * pct / 100
		src.Stats.Health = minInt(src.Stats.Health+heal, src.Stats.MaxHealth)
		e.emit("damage", "HEAL life_tap hero=%s heal=%d hero_hp=%d/%d", src.Name(), heal, src.Stats.Health, src.Stats.MaxHealth)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// sourceID is the unit id of a damage source (the hero), "" for none.
func (e *Engine) sourceID(src *d2mapentity.Player) string {
	if src == nil {
		return ""
	}

	return src.ID()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func describe(d *d2combat.Damage) string {
	var parts []string

	add := func(n string, v int32) {
		if v != 0 {
			parts = append(parts, fmt.Sprintf("%s:%.2f", n, float64(v)/256))
		}
	}

	add("phys", d.Physical)
	add("fire", d.Fire)
	add("ltng", d.Lightning)
	add("mag", d.Magic)
	add("cold", d.Cold)
	add("pois", d.Poison)

	if d.StunLen != 0 {
		parts = append(parts, fmt.Sprintf("stun:%df", d.StunLen))
	}

	if d.ColdLen != 0 {
		parts = append(parts, fmt.Sprintf("chill:%df", d.ColdLen))
	}

	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, ",")
}

// ---- simulation ----

// Advance runs the missile simulation for elapsed seconds at 25 Hz, then
// updates the drawing entities.
func (e *Engine) Advance(elapsed float64) {
	if elapsed > maxCatchUp {
		elapsed = maxCatchUp
	}

	e.acc += elapsed

	for e.acc >= frameSeconds {
		e.acc -= frameSeconds
		e.frame++
		e.sim.Step()
		e.runTimers()
		e.tick()
	}

	e.syncVisuals()
}

func (e *Engine) runTimers() {
	keep := e.timers[:0]

	var due []func()

	for _, t := range e.timers {
		if t.frame <= e.frame {
			due = append(due, t.fn)
		} else {
			keep = append(keep, t)
		}
	}

	e.timers = keep

	for _, fn := range due {
		fn()
	}

	for ent, until := range e.fx {
		if until <= e.frame {
			e.mapEngine.RemoveEntity(ent)
			delete(e.fx, ent)
		}
	}
}

func (e *Engine) onSim(ev d2missile.Event) {
	m := ev.Missile
	name := m.Spec.Name

	switch ev.Kind {
	case d2missile.EventCreate:
		e.Counters.Missiles++
		e.emit("missile", "MISSILE create name=%s id=%d skill=%d level=%d at=(%.1f,%.1f) dir=(%.2f,%.2f) vel=%.1f life=%d dmg=%s",
			name, m.ID, m.SkillID, m.Level, m.X, m.Y, m.DX, m.DY, float64(m.Velocity)/256, m.Life, describeDesc(&m.Damage))
	case d2missile.EventHit:
		e.Counters.Hits++
		tname := ev.Target.ID()

		mt, _ := ev.Target.(*monsterTarget)
		if mt != nil {
			tname = mt.m.Label()
		}

		e.emit("hit", "MISSILE hit name=%s id=%d target=%s chance=%d roll=%d %s", name, m.ID, tname, ev.Chance, ev.Roll,
			describe(&ev.Damage))

		if mt != nil && ev.Damage.SumTotal(false) > 0 {
			e.hurt(mt.m, e.owner(m), &ev.Damage, e.skillName(m.SkillID))

			// event 6 (missile): only hits with physical damage dispatch the item events here (UNVERIFIED rule)
			if ev.Damage.Physical > 0 {
				e.itemEvents(mt.m, e.owner(m), false)
			}
		}

		if mt != nil {
			e.splash(m, mt.m, &ev.Damage)
		}
	case d2missile.EventMiss:
		e.Counters.Misses++
		e.emit("hit", "MISSILE miss name=%s id=%d target=%s chance=%d roll=%d", name, m.ID, ev.Target.ID(), ev.Chance, ev.Roll)
	case d2missile.EventPierce:
		e.emit("missile", "MISSILE pierce name=%s id=%d", name, m.ID)
	case d2missile.EventWall:
		e.Counters.Walls++
		e.emit("missile", "MISSILE end name=%s id=%d reason=wall at=(%.1f,%.1f)", name, m.ID, m.X, m.Y)
		e.splashAt(m)
	case d2missile.EventVanish:
		// destroyed without running the hit function (0x5abd00 return 2)
		e.emit("missile", "MISSILE end name=%s id=%d reason=vanish at=(%.1f,%.1f)", name, m.ID, m.X, m.Y)
	case d2missile.EventArea:
		e.areaDamage(ev)
	case d2missile.EventExpire:
		e.Counters.Expired++
		e.emit("missile", "MISSILE end name=%s id=%d reason=expire at=(%.1f,%.1f)", name, m.ID, m.X, m.Y)
		// verified (0x5aba10): expiry runs the hit func with a null target, so
		// area damage (hit func 1) also fires where a missile runs out.
		e.splashAt(m)
	case d2missile.EventExplode:
		e.explosion(ev.Name, m.X, m.Y)
	}

	if m.Dead() {
		if ent := e.visuals[m.ID]; ent != nil {
			e.mapEngine.RemoveEntity(ent)
			delete(e.visuals, m.ID)
		}
	}
}

func (e *Engine) skillName(id int) string {
	if sk := e.pipe.Skills.ByID(id); sk != nil {
		return sk.Name
	}

	return fmt.Sprint(id)
}

func (e *Engine) owner(m *d2missile.Missile) *d2mapentity.Player {
	if h := e.heroes[m.Owner.ID]; h != nil {
		return h.p
	}

	return nil
}

func describeDesc(d *d2missile.DamageDesc) string {
	var parts []string

	add := func(n string, lo, hi int32) {
		if lo != 0 || hi != 0 {
			parts = append(parts, fmt.Sprintf("%s:%.2f-%.2f", n, float64(lo)/256, float64(hi)/256))
		}
	}

	add("phys", d.PhysMin, d.PhysMax)
	add("fire", d.Fire.Min, d.Fire.Max)
	add("ltng", d.Lightning.Min, d.Lightning.Max)
	add("mag", d.Magic.Min, d.Magic.Max)
	add("cold", d.Cold.Min, d.Cold.Max)
	add("pois", d.Poison.Min, d.Poison.Max)

	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, ",")
}

// ---- drawing ----

func (e *Engine) syncVisuals() {
	ms := e.sim.Missiles()
	sort.Slice(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })

	for _, m := range ms {
		if m.Dead() || m.Spec.Explosion {
			continue
		}

		ent := e.visuals[m.ID]
		if ent == nil {
			rec := e.asset.Records.Missiles[m.Spec.ID]
			if rec == nil || rec.Animation.CelFileName == "" {
				continue
			}

			var err error

			if ent, err = e.mapEngine.NewMissile(int(m.X), int(m.Y), rec); err != nil {
				e.Debugf("no graphic for missile %s: %v", m.Spec.Name, err)
				e.visuals[m.ID] = nil
				ent = nil

				continue
			}

			e.visuals[m.ID] = ent
			e.mapEngine.AddEntity(ent)
		}

		if ent != nil {
			ent.Place(m.X, m.Y, m.DX, m.DY)
		}
	}
}

// explosion shows a client side explosion missile for a short time.
func (e *Engine) explosion(name string, x, y float64) {
	rec := e.asset.Records.GetMissileByName(name)
	if rec == nil || rec.Animation.CelFileName == "" {
		return
	}

	ent, err := e.mapEngine.NewMissile(int(x), int(y), rec)
	if err != nil {
		return
	}

	ent.Place(x, y, 0, 0)
	e.mapEngine.AddEntity(ent)

	life := rec.Range
	if life < 1 {
		life = 12
	}

	e.fx[ent] = e.frame + life
}
