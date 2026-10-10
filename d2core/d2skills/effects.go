package d2skills

import (
	"fmt"
	"math"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// Applying the non-missile outcomes of casts (d2skill.Effect): states, curses,
// auras, area hits, rains and storms, summons and traps, movement. The pure
// rules live in d2skill (what a skill does) and d2state (what a state means);
// this file connects them to the hero, the monsters and the monster director.

const (
	auraPulse    = 25 // frames between aura pulses (U: perdelay is 50 in skills.txt)
	defaultCurse = 600
	trapRange    = 20
	trapPeriod   = 20
)

// watch is a recurring callback; it stops when fn returns false.
type watch struct {
	every, next int
	fn          func() bool
}

type auraRun struct {
	p    *d2mapentity.Player
	ef   d2skill.Effect
	next int
}

type stormRun struct {
	p     *d2mapentity.Player
	u     *heroUnit
	ef    d2skill.Effect
	until int
	next  int
}

type trapRun struct {
	m         *d2mapentity.Monster
	u         *heroUnit
	skillID   int
	missile   string
	skillName string
	next      int
}

// ---- helpers ----

func (e *Engine) heroPos(u *heroUnit) (int, int) { return u.Pos() }

// monstersNear lists the living monsters within a Chebyshev radius.
func (e *Engine) monstersNear(x, y, r int) []*d2mapentity.Monster {
	var out []*d2mapentity.Monster

	for _, m := range e.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if chebyshev(mx-x, my-y) <= r {
			out = append(out, m)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })

	return out
}

// near implements d2skill.Pipeline.Near.
func (e *Engine) near(x, y, r int) []d2skill.Foe {
	var out []d2skill.Foe

	for _, m := range e.monstersNear(x, y, r) {
		mx, my := m.SubtilePos()
		out = append(out, d2skill.Foe{Target: e.target(m), X: mx, Y: my})
	}

	return out
}

func statMods(in []d2skill.StatMod) []d2state.StatMod {
	out := make([]d2state.StatMod, len(in))
	for i, m := range in {
		out[i] = d2state.StatMod{Stat: m.Stat, Value: m.Value}
	}

	return out
}

func describeMods(in []d2skill.StatMod) string {
	s := ""

	for _, m := range in {
		s += fmt.Sprintf(" %s=%d", m.Stat, m.Value)
	}

	if s == "" {
		return " -"
	}

	return s
}

func (e *Engine) firstHero() *d2mapentity.Player {
	ids := make([]string, 0, len(e.heroes))
	for id := range e.heroes {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	if len(ids) == 0 {
		return nil
	}

	return e.heroes[ids[0]].p
}

// ---- dispatch ----

func (e *Engine) effect(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	switch ef.Kind {
	case "self_state":
		e.selfState(p, sk, ef)
	case "area_state":
		e.areaState(p, u, sk, ef)
	case "area_damage":
		e.staticField(p, u, sk, ef)
	case "area_hit":
		e.areaHit(p, u, sk, ef)
	case "strikes":
		e.strikes(p, u, sk, ef)
	case "storm":
		e.startStorm(p, u, sk, ef)
	case "aura":
		e.startAura(p, sk, ef)
	case "summon":
		e.summon(p, u, sk, ef)
	case "move":
		e.move(p, u, sk, ef)
	case "convert":
		e.convert(p, sk, ef)
	case "shield":
		e.shield(p, u, sk, ef)
	case "loot":
		e.loot(p, sk, ef)
	case "ward":
		e.ward(p, sk, ef)
	case "whirl":
		e.whirl(p, u, sk, ef)
	case "self_damage":
		loss := p.Stats.MaxHealth * ef.SelfDamagePct / 100
		p.Stats.Health = maxInt(p.Stats.Health-loss, 1)
		e.emit("damage", "DAMAGE skill=%q self hero=%s dmg=%d hero_hp=%d/%d", sk.Name, p.Name(), loss, p.Stats.Health,
			p.Stats.MaxHealth)
	case "clear_state":
		e.setOf(p.ID()).Remove(ef.State)
		e.emit("state", "STATE clear skill=%q unit=%s state=%s", sk.Name, p.Name(), ef.State)
	}
}

// ---- states ----

// selfState puts a timed (or indefinite) state on the hero. Stack > 0 makes it
// a counter: the stat named like the state holds the count (assassin charges).
func (e *Engine) selfState(p *d2mapentity.Player, sk *d2skill.Skill, ef *d2skill.Effect) {
	set := e.setOf(p.ID())
	inst := d2state.Instance{Name: ef.State, Source: p.ID(), SkillID: sk.ID, Level: ef.Level, Mods: statMods(ef.Stats), Count: 1}

	if ef.Frames > 0 {
		inst.Until = e.frame + ef.Frames
	}

	if ef.Stack > 0 {
		if prev := set.Get(e.frame, ef.State); prev != nil {
			inst.Count = minInt(prev.Count+1, ef.Stack)
		}

		for i := range inst.Mods {
			if inst.Mods[i].Stat == ef.State {
				inst.Mods[i].Value = inst.Count
			}
		}
	}

	// SRVDO_FrozenArmorState (0x5c7540) ends every state of the same States.txt
	// group (itself included) before it builds the new statlist (0x56a480).
	set.ClearGroup(e.frame, ef.State)
	set.Apply(e.frame, inst)
	e.emit("state", "STATE apply skill=%q unit=%s state=%s frames=%d stacks=%d stats=%s chill_attackers=%d", sk.Name, p.Name(),
		ef.State, ef.Frames, inst.Count, describeMods(ef.Stats), ef.Chill)

	if ef.Missile != "" && ef.Interval > 0 { // Blaze: fire is left behind
		e.trail(p, sk, ef)
	}
}

// trail drops a missile at the hero's feet every Interval frames while the
// state lasts (Blaze).
func (e *Engine) trail(p *d2mapentity.Player, sk *d2skill.Skill, ef *d2skill.Effect) {
	u := e.hero(p)
	end := e.frame + ef.Frames
	name := ef.Missile

	e.watches = append(e.watches, &watch{every: ef.Interval, next: e.frame, fn: func() bool {
		if e.frame >= end {
			return false
		}

		x, y := u.Pos()
		e.pipe.CastTrap(u, sk.ID, name, x, y, d2skill.Target{X: x, Y: y})

		return true
	}})
}

// applyMonsterState puts a state on a monster and makes the monster react to
// it (terror flees, stun/freeze hold, chill slows).
func (e *Engine) applyMonsterState(m *d2mapentity.Monster, inst d2state.Instance) {
	e.target(m)
	e.setOf(m.ID()).Apply(e.frame, inst)
	e.syncMonster(m)
	e.forceFromMods(m, inst)
}

// syncMonster pushes the states of a monster to the director: speed, stun or
// freeze, fear.
func (e *Engine) syncMonster(m *d2mapentity.Monster) {
	set := e.setOf(m.ID())

	e.monsters.SetSlow(m, set.SpeedPct(e.frame))

	for _, name := range []string{d2state.Stun, d2state.Freeze} {
		if in := set.Get(e.frame, name); in != nil && in.Until > e.frame {
			e.monsters.Stun(m, in.Until-e.frame)
		}
	}

	if in := set.Get(e.frame, d2state.Terror); in != nil && in.Until > e.frame {
		e.monsters.Flee(m, in.Until-e.frame)
	}
}

// areaState is the curse / shout style effect on every monster in a radius
// around the caster or the aim point.
func (e *Engine) areaState(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	cx, cy := u.Pos()
	if ef.Origin == "aim" {
		cx, cy = ef.X, ef.Y
	}

	frames := ef.Frames
	if frames <= 0 {
		frames = defaultCurse
	}

	n := 0

	for _, m := range e.monstersNear(cx, cy, ef.Radius) {
		n++

		e.applyMonsterState(m, d2state.Instance{Name: ef.State, Until: e.frame + frames, Mods: statMods(ef.Stats),
			Source: p.ID(), SkillID: sk.ID, Level: ef.Level})
		e.emit("state", "STATE apply skill=%q unit=%s state=%s frames=%d stats=%s", sk.Name, m.Label(), ef.State, frames,
			describeMods(ef.Stats))
	}

	for _, rv := range e.rivalsNear(cx, cy, ef.Radius) {
		n++

		e.setOf(rv.ID()).Apply(e.frame, d2state.Instance{Name: ef.State, Until: e.frame + frames, Mods: statMods(ef.Stats),
			Source: p.ID(), SkillID: sk.ID, Level: ef.Level})
		e.emit("state", "STATE apply skill=%q unit=%s state=%s frames=%d (rival, local only)", sk.Name, rv.Name(), ef.State, frames)
	}

	e.emit("state", "STATE area skill=%q state=%s radius=%d at=(%d,%d) affected=%d", sk.Name, ef.State, ef.Radius, cx, cy, n)
}

// ---- area damage ----

// rollDesc rolls an area damage descriptor with the hero's generator.
func (e *Engine) rollDesc(u *heroUnit, d *d2missile.DamageDesc) d2combat.Damage {
	return d.Roll(u.seed)
}

func (e *Engine) areaHit(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	cx, cy := ef.X, ef.Y
	if ef.Origin == "self" {
		cx, cy = u.Pos()
	}

	run := func() { e.hitArea(p, u, sk.Name, cx, cy, ef.Radius, ef.Desc, ef.CorpseID) }

	if ef.Delay > 0 {
		e.after(ef.Delay, run)

		return
	}

	run()
}

// hitArea damages every monster within radius of a point.
func (e *Engine) hitArea(p *d2mapentity.Player, u *heroUnit, name string, cx, cy, radius int, d *d2missile.DamageDesc,
	corpseID string) int {
	if corpseID != "" {
		for _, c := range e.monsters.Corpses() {
			if c.ID() == corpseID {
				e.monsters.RemoveCorpse(c)
				e.emit("state", "CORPSE consumed skill=%q at=(%d,%d)", name, cx, cy)
			}
		}
	}

	n := 0

	if d != nil {
		for _, m := range e.monstersNear(cx, cy, radius) {
			dmg := e.rollDesc(u, d)
			n++

			e.target(m)
			e.hurt(m, p, &dmg, name)
		}

		n += e.hitRivals(p, cx, cy, radius, func() d2combat.Damage { return e.rollDesc(u, d) }, name)
	}

	e.Counters.AreaHits += n
	e.emit("hit", "SKILL area skill=%q at=(%d,%d) radius=%d targets=%d", name, cx, cy, radius, n)

	return n
}

func (e *Engine) strikes(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	origin := ef.Strikes
	if len(origin) == 0 {
		return
	}

	e.emit("hit", "SKILL strikes skill=%q count=%d first=(%d,%d) last_delay=%d", sk.Name, len(origin), origin[0].X,
		origin[0].Y, origin[len(origin)-1].Delay)

	for _, s := range origin {
		s := s
		e.after(s.Delay, func() { e.hitArea(p, u, sk.Name, s.X, s.Y, s.Radius, ef.Desc, "") })
	}
}

// ---- storms ----

func (e *Engine) startStorm(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	frames := ef.Frames
	if frames <= 0 {
		frames = 25 * 20
	}

	if ef.State != "" {
		e.setOf(p.ID()).Apply(e.frame, d2state.Instance{Name: ef.State, Until: e.frame + frames, Source: p.ID(),
			SkillID: sk.ID, Level: ef.Level})
	}

	e.storms = append(e.storms, &stormRun{p: p, u: u, ef: *ef, until: e.frame + frames, next: e.frame + ef.Interval})
	e.emit("state", "STATE storm skill=%q mode=%s frames=%d interval=%d radius=%d", sk.Name, ef.Mode, frames, ef.Interval,
		ef.Radius)
}

func (e *Engine) stormsTick() {
	live := e.storms[:0]

	for _, s := range e.storms {
		if e.frame >= s.until {
			continue
		}

		live = append(live, s)

		if e.frame < s.next {
			continue
		}

		s.next = e.frame + maxInt(s.ef.Interval, 1)
		hx, hy := s.u.Pos()

		switch s.ef.Mode {
		case "nearest":
			var best *d2mapentity.Monster

			bd := s.ef.Radius + 1

			for _, m := range e.monstersNear(hx, hy, s.ef.Radius) {
				mx, my := m.SubtilePos()
				if d := chebyshev(mx-hx, my-hy); d < bd {
					best, bd = m, d
				}
			}

			if best != nil {
				bx, by := best.SubtilePos()
				e.hitArea(s.p, s.u, s.ef.SkillName, bx, by, 2, s.ef.Desc, "")
			}
		case "scatter":
			a := float64(s.u.seed.Roll(360)) * math.Pi / 180
			r := math.Sqrt(float64(s.u.seed.Roll(1000))/1000) * float64(s.ef.Radius)
			e.hitArea(s.p, s.u, s.ef.SkillName, hx+int(math.Round(math.Cos(a)*r)), hy+int(math.Round(math.Sin(a)*r)), 3, s.ef.Desc, "")
		default:
			e.hitArea(s.p, s.u, s.ef.SkillName, hx, hy, s.ef.Radius, s.ef.Desc, "")
		}
	}

	e.storms = live
}

// ---- auras ----

func (e *Engine) startAura(p *d2mapentity.Player, sk *d2skill.Skill, ef *d2skill.Effect) {
	if old := e.auras[p.ID()]; old != nil {
		e.setOf(p.ID()).Remove(old.ef.State)
	}

	e.auras[p.ID()] = &auraRun{p: p, ef: *ef, next: e.frame}
	e.emit("state", "STATE aura skill=%q mode=%s state=%s radius=%d stats=%s target_state=%s target_stats=%s", sk.Name, ef.Mode,
		ef.State, ef.Radius, describeMods(ef.Stats), ef.TargetState, describeMods(ef.TargetStats))
	e.pulseAura(e.auras[p.ID()])
}

func (e *Engine) aurasTick() {
	ids := make([]string, 0, len(e.auras))
	for id := range e.auras {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, id := range ids {
		if a := e.auras[id]; e.frame >= a.next {
			e.pulseAura(a)
		}
	}
}

func (e *Engine) pulseAura(a *auraRun) {
	a.next = e.frame + auraPulse
	ef := &a.ef
	u := e.hero(a.p)
	hx, hy := u.Pos()
	until := e.frame + auraPulse + 3

	if len(ef.Stats) > 0 || ef.State != "" && ef.Mode == "friendly" {
		e.setOf(a.p.ID()).Apply(e.frame, d2state.Instance{Name: ef.State, Until: until, Mods: statMods(ef.Stats), Source: a.p.ID(),
			SkillID: ef.SkillID, Level: ef.Level})

		for _, m := range ef.Stats {
			if m.Stat == "hitpoints" && m.Value > 0 {
				a.p.Stats.Health = minInt(a.p.Stats.Health+maxInt(m.Value>>8, 1), a.p.Stats.MaxHealth)
			}
		}
	}

	switch ef.Mode {
	case "enemy", "damage":
		for _, m := range e.monstersNear(hx, hy, ef.Radius) {
			if ef.TargetState != "" {
				e.applyMonsterState(m, d2state.Instance{Name: ef.TargetState, Until: until, Mods: statMods(ef.TargetStats),
					Source: a.p.ID(), SkillID: ef.SkillID, Level: ef.Level})
			}

			if ef.Mode == "damage" && ef.Desc != nil {
				dmg := e.rollDesc(u, ef.Desc)
				e.target(m)
				e.hurt(m, a.p, &dmg, ef.SkillName)
			}
		}

		for _, rv := range e.rivalsNear(hx, hy, ef.Radius) {
			if ef.TargetState != "" {
				e.setOf(rv.ID()).Apply(e.frame, d2state.Instance{Name: ef.TargetState, Until: until, Mods: statMods(ef.TargetStats),
					Source: a.p.ID(), SkillID: ef.SkillID, Level: ef.Level})
			}

			if ef.Mode == "damage" && ef.Desc != nil {
				dmg := e.rollDesc(u, ef.Desc)
				e.hurtPlayer(rv, a.p, &dmg, ef.SkillName)
			}
		}
	case "redemption":
		for _, c := range e.monsters.Corpses() {
			cx, cy := c.SubtilePos()
			if chebyshev(cx-hx, cy-hy) > ef.Radius {
				continue
			}

			if int(u.seed.Roll(100)) < ef.Stack {
				e.monsters.RemoveCorpse(c)
				a.p.Stats.Health = minInt(a.p.Stats.Health+ef.Heal, a.p.Stats.MaxHealth)
				a.p.Stats.Mana = minInt(a.p.Stats.Mana+ef.Dist, a.p.Stats.MaxMana)
				e.emit("state", "STATE redemption hero=%s life+%d mana+%d hero_hp=%d/%d", a.p.Name(), ef.Heal, ef.Dist,
					a.p.Stats.Health, a.p.Stats.MaxHealth)
			}

			break
		}
	}
}

// ---- movement ----

func (e *Engine) setHeroPos(p *d2mapentity.Player, x, y int) {
	p.StopMoving()
	p.Position = d2vector.NewPosition(float64(x)+0.5, float64(y)+0.5)
	p.StopMoving()
}

// move teleports the hero at once, or slides it to the destination in a few
// frames (leap, charge).
func (e *Engine) move(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	fx, fy := u.Pos()
	e.emit("state", "MOVE skill=%q mode=%s from=(%d,%d) to=(%d,%d)", sk.Name, ef.Mode, fx, fy, ef.X, ef.Y)

	if ef.Mode == "teleport" {
		e.setHeroPos(p, ef.X, ef.Y)
		return
	}

	const steps = 8

	for i := 1; i <= steps; i++ {
		i := i
		e.after(i, func() {
			e.setHeroPos(p, fx+(ef.X-fx)*i/steps, fy+(ef.Y-fy)*i/steps)
		})
	}
}

// ---- summons ----

func (e *Engine) alivePets(heroID, petType string) []*d2mapentity.Monster {
	var live []*d2mapentity.Monster

	for _, m := range e.pets[heroID] {
		if m.Alive() {
			if k, _ := e.monsters.MinionKind(m); k != "" {
				live = append(live, m)
			}
		}
	}

	e.pets[heroID] = live

	var out []*d2mapentity.Monster

	for _, m := range live {
		if _, tag := e.monsters.MinionKind(m); tag == petType {
			out = append(out, m)
		}
	}

	return out
}

func (e *Engine) summon(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect) {
	o := ef.Summon
	key := o.Key

	stat := e.monsters.FindStat(key)
	if stat == nil {
		e.emit("summon", "SUMMON skill=%q key=%s refused=unknown_monster", sk.Name, key)
		return
	}

	opt := d2monsters.MinionOptions{Owner: p, Kind: o.Kind, HPPct: o.HPPct, HPFlat: o.HPFlat, Frames: o.Frames, Tag: o.PetType,
		Level: o.Level}
	if opt.Tag == "" || opt.Tag == "none" {
		opt.Tag = stat.Key // the tag petworld.go gives such minions
	}

	for _, m := range o.Stats {
		switch m.Stat {
		case "damagepercent":
			opt.DamagePct += m.Value
		case "tohit":
			opt.ToHit += m.Value
		case "armorclass":
			opt.ArmorClass += m.Value
		}
	}

	corpseLevel := 0

	if o.CorpseID != "" {
		for _, c := range e.monsters.Corpses() {
			if c.ID() == o.CorpseID {
				corpseLevel = c.Vitals.Level

				if o.UseCorpseType {
					stat = c.Stat
				}

				e.monsters.RemoveCorpse(c)
			}
		}
	}

	hx, hy := u.Pos()
	x, y := o.X, o.Y

	if o.Kind == "minion" && o.CorpseID == "" {
		x, y = hx, hy
	}

	// d2summon.Roster decides how many minions appear and which old ones make
	// room; the Summoner computes their stats (d2monsters/petworld.go).
	ids, plan := e.monsters.Summoner().Cast(p.ID(), stat.Key, o, e.pipe.SummonPassives(u, sk.ID), func(i, n int) (int, int) {
		if o.Kind == "wall" {
			return wallCell(hx, hy, x, y, i, n, o.Mode)
		}

		if i < len(o.Cells) {
			return x + o.Cells[i][0], y + o.Cells[i][1]
		}

		return x, y
	})

	if len(ids) == 0 {
		e.emit("summon", "SUMMON skill=%q key=%s refused=limit max=%d evicted=%d", sk.Name, stat.Key, o.Max, len(plan.Evict))
		return
	}

	made := 0

	for _, id := range ids {
		m := e.monsters.MinionByBrainID(id)
		if m == nil {
			continue
		}

		made++

		if o.OwnerHPPct > 0 { // Dopplezon: life is a percent of the owner's maximum life
			hp := maxInt(p.Stats.MaxHealth*o.OwnerHPPct/100, 1)
			m.Vitals.MaxHP, m.Vitals.HP = hp, hp
		}

		e.pets[p.ID()] = append(e.pets[p.ID()], m)

		if o.UseCorpseType {
			reviveCap(m, corpseLevel, p.Stats.Level)
		}
		e.target(m)

		switch o.Kind {
		case "trap":
			e.registerTrap(u, sk, ef, o, m)
		case "totem":
			e.registerTotem(p, u, sk, ef, o, m)
		}
	}

	e.emit("summon", "SUMMON skill=%q key=%s kind=%s created=%d alive=%d max=%d hp_pct=%d damage_pct=%d at=(%d,%d)", sk.Name,
		stat.Key, o.Kind, made, len(e.alivePets(p.ID(), opt.Tag)), o.Max, o.HPPct, opt.DamagePct, x, y)
}

// wallCell is the i-th of n wall pieces: a line across the cast direction, or
// a ring of radius 2 around the aim.
func wallCell(hx, hy, x, y, i, n int, mode string) (int, int) {
	if mode == "ring" {
		a := 2 * math.Pi * float64(i) / float64(n)
		return x + int(math.Round(math.Cos(a)*2)), y + int(math.Round(math.Sin(a)*2))
	}

	dx, dy := float64(x-hx), float64(y-hy)

	d := math.Hypot(dx, dy)
	if d == 0 {
		dx, dy, d = 1, 0, 1
	}

	off := float64(i) - float64(n-1)/2

	return x + int(math.Round(-dy/d*off*2)), y + int(math.Round(dx/d*off*2))
}

// registerTrap makes a sentry shoot: every trapPeriod frames the nearest
// monster within trapRange gets a missile of the trap skill from the sentry's
// own position (U: the sentries use the missile of the skill named in sumskill1
// and the damage of the skill that placed them).
func (e *Engine) registerTrap(u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect, o *d2skill.SummonOrder, m *d2mapentity.Monster) {
	missile := sk.SrvMissile

	if missile == "" {
		missile = sk.SrvMissileA
	}

	if ts := e.pipe.Skills.ByName(o.TrapSkill); ts != nil {
		switch {
		case ts.SrvMissile != "":
			missile = ts.SrvMissile
		case ts.SrvMissileA != "":
			missile = ts.SrvMissileA
		}
	}

	if missile == "" {
		e.emit("summon", "TRAP skill=%q has no missile", sk.Name)
		return
	}

	e.traps = append(e.traps, &trapRun{m: m, u: u, skillID: sk.ID, missile: missile, skillName: sk.Name, next: e.frame + trapPeriod})
	e.emit("summon", "TRAP armed skill=%q missile=%s period=%d range=%d", sk.Name, missile, trapPeriod, trapRange)
}

// trapCaster is the part of d2skill.Pipeline a trap shot needs (a fake in tests).
type trapCaster interface {
	CastTrap(u d2skill.Unit, skillID int, missile string, fromX, fromY int, tgt d2skill.Target) *d2missile.Missile
}

// trapCaster returns the pipeline, or the fake a test injected.
func (e *Engine) trapCaster() trapCaster {
	if e.fakeCaster != nil {
		return e.fakeCaster
	}

	return e.pipe
}

// fireTrapAt is the one place a sentry shot is built: used by trapsTick and by
// the Director callback (FireTrap), so both fire the same missile with the
// same damage.
func (e *Engine) fireTrapAt(c trapCaster, t *trapRun, tx, ty int, best *d2mapentity.Monster) bool {
	bx, by := best.SubtilePos()
	tg := d2skill.Target{X: bx, Y: by, Unit: e.target(best), UX: bx, UY: by}

	return c.CastTrap(t.u, t.skillID, t.missile, tx, ty, tg) != nil
}

// FireTrap implements d2monsters.SkillFirer (UNVERIFIED path, see
// d2monsters/trapfire.go: nothing arms traps with it yet).
func (e *Engine) FireTrap(s d2monsters.TrapShot) bool {
	if s.Owner == nil || s.Target == nil {
		return false
	}

	t := &trapRun{m: s.Trap, u: e.hero(s.Owner), skillID: s.Spec.SkillID, missile: s.Spec.Missile, skillName: s.Spec.SkillName}
	if !e.fireTrapAt(e.trapCaster(), t, s.FromX, s.FromY, s.Target) {
		return false
	}

	e.emit("summon", "TRAP fire skill=%q missile=%s target=%s", t.skillName, t.missile, s.Target.Label())

	return true
}

func (e *Engine) trapsTick() {
	live := e.traps[:0]

	for _, t := range e.traps {
		if !t.m.Alive() {
			continue
		}

		live = append(live, t)

		if e.frame < t.next {
			continue
		}

		t.next = e.frame + trapPeriod
		tx, ty := t.m.SubtilePos()

		var best *d2mapentity.Monster

		bd := trapRange + 1

		for _, m := range e.monstersNear(tx, ty, trapRange) {
			mx, my := m.SubtilePos()
			if d := chebyshev(mx-tx, my-ty); d < bd {
				best, bd = m, d
			}
		}

		if best == nil {
			continue
		}

		if e.fireTrapAt(e.trapCaster(), t, tx, ty, best) {
			e.emit("summon", "TRAP fire skill=%q missile=%s target=%s", t.skillName, t.missile, best.Label())
		}
	}

	e.traps = live
}

// registerTotem gives the hero the aura of the totem's skill (sumskill1: Oak
// Sage Aura...) as a state refreshed once a second for as long as the totem
// stands. The aura stats are evaluated at the level of the summoning skill.
func (e *Engine) registerTotem(p *d2mapentity.Player, u *heroUnit, sk *d2skill.Skill, ef *d2skill.Effect, o *d2skill.SummonOrder,
	m *d2mapentity.Monster) {
	ts := e.pipe.Skills.ByName(o.TrapSkill)
	if ts == nil {
		return
	}

	env := d2skill.NewEnv(ts, ef.Level, u, e.pipe.Skills)

	var mods []d2state.StatMod

	var desc string

	for i := 1; i <= 6; i++ {
		if ts.AuraStat[i] != "" {
			v := env.Eval(ts.AuraStatCalc[i])
			mods = append(mods, d2state.StatMod{Stat: ts.AuraStat[i], Value: v})
			desc += fmt.Sprintf(" %s=%d", ts.AuraStat[i], v)
		}
	}

	name := ts.AuraState
	if name == "" {
		name = ts.Name
	}

	apply := func() bool {
		if !m.Alive() {
			e.setOf(p.ID()).Remove(name)
			return false
		}

		e.setOf(p.ID()).Apply(e.frame, d2state.Instance{Name: name, Until: e.frame + auraPulse + 3, Mods: mods,
			Source: p.ID(), SkillID: sk.ID, Level: ef.Level})

		return true
	}

	apply()
	e.watches = append(e.watches, &watch{every: auraPulse, next: e.frame + auraPulse, fn: apply})
	e.emit("state", "STATE totem skill=%q aura=%q state=%s stats=%s", sk.Name, ts.Name, name, desc)
}

// ---- per-frame upkeep ----

type dotTotal struct{ poison, burn int }

func (e *Engine) tick() {
	ids := make([]string, 0, len(e.sets))
	for id := range e.sets {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, id := range ids {
		res := e.sets[id].Tick(e.frame)
		mt := e.targets[id]

		if mt == nil || !mt.m.Alive() {
			continue
		}

		if res.Poison+res.Burn > 0 {
			e.dot(mt.m, res.Poison, res.Burn)
		}

		if len(res.Expired) > 0 {
			e.syncMonster(mt.m)
		}
	}

	if e.frame%auraPulse == 0 {
		e.flushDot()
	}

	e.aurasTick()
	e.stormsTick()
	e.trapsTick()

	live := e.watches[:0]

	for _, w := range e.watches {
		if e.frame >= w.next {
			w.next = e.frame + maxInt(w.every, 1)

			if !w.fn() {
				continue
			}
		}

		live = append(live, w)
	}

	e.watches = live
}

// dot applies one frame of poison and burn damage, ignoring resists (they
// were applied to the stream when it started).
func (e *Engine) dot(m *d2mapentity.Monster, poison, burn int) {
	whole := poison + burn
	src := e.firstHero()

	e.dots[m.ID()] = dotTotal{e.dots[m.ID()].poison + poison, e.dots[m.ID()].burn + burn}
	e.Counters.Damage += whole
	e.Counters.DotDamage += whole
	e.monsters.DamageOverTime(m, whole, src)

	if !m.Alive() {
		e.Counters.Kills++
		e.setOf(m.ID()).Death("monster")
		e.flushDotFor(m)
		e.emit("damage", "KILL skill=%q target=%s", "damage over time", m.Label())
	}
}

func (e *Engine) flushDotFor(m *d2mapentity.Monster) {
	if t, ok := e.dots[m.ID()]; ok && t.poison+t.burn > 0 {
		e.emit("damage", "DAMAGE dot target=%s poison=%d burn=%d hp=%d/%d", m.Label(), t.poison, t.burn, maxInt(m.Vitals.HP, 0),
			m.Vitals.MaxHP)
		delete(e.dots, m.ID())
	}
}

func (e *Engine) flushDot() {
	ids := make([]string, 0, len(e.dots))
	for id := range e.dots {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, id := range ids {
		if mt := e.targets[id]; mt != nil {
			e.flushDotFor(mt.m)
		}
	}
}

// ---- hero defense ----

// heroAvoid is the Director's HeroAvoid hook: dodge / avoid / evade
// (d2combat.RollAvoid), rolled before the damage roll.
func (e *Engine) heroAvoid(p *d2mapentity.Player, _ *d2mapentity.Monster, melee bool) (bool, string) {
	u := e.hero(p)
	vel := p.GetVelocity()

	switch out := d2combat.RollAvoid(u.seed, d2combat.AvoidInput{
		Moving:      !vel.IsZero(),
		EvadeChance: u.Stat("passive_evade"), DodgeChance: u.Stat("passive_dodge"), AvoidChance: u.Stat("passive_avoid"),
		IsMissile: !melee,
	}); out {
	case d2combat.AvoidAvoided, d2combat.AvoidDodged, d2combat.AvoidEvaded:
		return true, fmt.Sprintf(" avoided=%d", out)
	}

	return false, ""
}

// heroDefense is the Director's HeroDefense hook: Energy Shield, Bone Armor
// type pools, Thorns (dodge / avoid / evade moved to heroAvoid).
func (e *Engine) heroDefense(p *d2mapentity.Player, attacker *d2mapentity.Monster, melee bool, dmg int) (int, string) {
	set := e.setOf(p.ID())

	note := ""

	// Bone Armor, Cyclone Armor: a pool of absorbed damage (the stat is in 8.8)
	if pool := set.Stat(e.frame, "bonearmor"); pool > 0 {
		taken := set.Drain(e.frame, "bonearmor", dmg<<8) >> 8
		dmg -= taken
		note += fmt.Sprintf(" absorbed_by_armor=%d", taken)
	}

	// Energy Shield: x_energyshield_pct of the damage costs mana instead (U ratio)
	if pct := set.Stat(e.frame, "x_energyshield_pct"); pct > 0 && dmg > 0 {
		absorb := dmg * pct / 100
		ratio := maxInt(set.Stat(e.frame, "x_energyshield_ratio"), 1)
		cost := absorb * ratio / 16
		have := p.Stats.Mana

		if cost > have {
			absorb = absorb * have / maxInt(cost, 1)
			cost = have
		}

		p.Stats.Mana -= cost
		dmg -= absorb
		note += fmt.Sprintf(" absorbed_by_shield=%d mana=%d", absorb, cost)
	}

	if pct := set.ThornsPct(e.frame); pct > 0 && melee && dmg > 0 {
		back := dmg * pct / 100
		e.monsters.Damage(attacker, back, p)
		note += fmt.Sprintf(" thorns=%d", back)
	}

	if dmg < 0 {
		dmg = 0
	}

	if note != "" {
		e.emit("damage", "DEFENSE hero=%s%s dmg_through=%d", p.Name(), note, dmg)
	}

	return dmg, ""
}

// ---- splash ----

// splashRadius is the radius of the glacial spike style splash (hit func 13:
// aurarangecalc); hit funcs 1 and 14 are exact area hits (EventArea).
func (e *Engine) splashRadius(m *d2missile.Missile) int {
	h := e.heroes[m.Owner.ID]
	if h == nil {
		return 0
	}

	if m.Spec.SrvHitFunc == 13 {
		return e.pipe.AuraRange(h, m.SkillID)
	}

	return 0
}

// areaDamage applies an area hit function (1: Fire Ball style, 14: Meteor,
// 0x5a7500 / 0x5a8680): one damage roll for every monster whose subtile is
// within radius subtiles of the missile (squared distance <= radius^2). Hit
// function 1 deals no direct damage to the unit that was struck, only this.
func (e *Engine) areaDamage(ev d2missile.Event) {
	m := ev.Missile

	h := e.heroes[m.Owner.ID]
	if h == nil || ev.Radius <= 0 {
		return
	}

	n := 0

	for _, o := range e.monstersNear(int(m.X), int(m.Y), ev.Radius) {
		d := ev.Damage
		n++

		e.target(o)
		e.hurt(o, h.p, &d, e.skillName(m.SkillID)+" area")
	}

	// hostile heroes in the radius take the same roll (PvP: scaled by hurtPlayer)
	for _, o := range e.rivalsNear(int(m.X), int(m.Y), ev.Radius) {
		d := ev.Damage
		n++

		e.hurtPlayer(o, h.p, &d, e.skillName(m.SkillID)+" area")
	}

	if n > 0 {
		e.Counters.AreaHits += n
		e.emit("hit", "SKILL area skill=%q at=(%d,%d) radius=%d targets=%d", e.skillName(m.SkillID), int(m.X), int(m.Y), ev.Radius, n)
	}
}

// splash damages the other monsters near the one a missile just hit with the
// same rolled damage.
func (e *Engine) splash(m *d2missile.Missile, primary *d2mapentity.Monster, dmg *d2combat.Damage) {
	r := e.splashRadius(m)
	if r <= 0 {
		return
	}

	px, py := primary.SubtilePos()
	n := 0

	for _, o := range e.monstersNear(px, py, r) {
		if o == primary {
			continue
		}

		d := *dmg
		n++

		e.target(o)
		e.hurt(o, e.owner(m), &d, e.skillName(m.SkillID)+" splash")
	}

	for _, o := range e.rivalsNear(px, py, r) {
		d := *dmg
		n++

		e.hurtPlayer(o, e.owner(m), &d, e.skillName(m.SkillID)+" splash")
	}

	if n > 0 {
		e.Counters.AreaHits += n
		e.emit("hit", "SKILL splash skill=%q at=(%d,%d) radius=%d targets=%d", e.skillName(m.SkillID), px, py, r, n)
	}
}

// splashAt is the splash of a missile that ended on a wall or ran out.
func (e *Engine) splashAt(m *d2missile.Missile) {
	r := e.splashRadius(m)
	if r <= 0 {
		return
	}

	h := e.heroes[m.Owner.ID]
	n := 0

	for _, o := range e.monstersNear(int(m.X), int(m.Y), r) {
		d := m.Damage.Roll(m.Owner.Roller)
		n++

		e.target(o)
		e.hurt(o, h.p, &d, e.skillName(m.SkillID)+" splash")
	}

	for _, o := range e.rivalsNear(int(m.X), int(m.Y), r) {
		d := m.Damage.Roll(m.Owner.Roller)
		n++

		e.hurtPlayer(o, h.p, &d, e.skillName(m.SkillID)+" splash")
	}

	if n > 0 {
		e.Counters.AreaHits += n
		e.emit("hit", "SKILL splash skill=%q at=(%d,%d) radius=%d targets=%d", e.skillName(m.SkillID), int(m.X), int(m.Y), r, n)
	}
}
