package d2monsters

import (
	"fmt"
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Summoned minions, traps, totems and walls (skills: Raise Skeleton, golems,
// Raven, spirit wolves, Blade Sentinel, Bone Wall...). They are monstats units
// run by the Director like any monster but allied to a hero: they follow it,
// fight the nearest hostile monster and are never listed by Monsters.
//
// UNVERIFIED simplifications: hostile monsters never target minions (the real
// AI does), minion damage is the monstats A1 range scaled by the skill's
// damagepercent, summoned level is the owner's character level, and the
// minion AI is a plain chase-and-hit (no ranged attackers: the Skeleton Mage
// melees).

const (
	minionAggro   = 24 // subtiles in which a minion notices a hostile
	minionLeash   = 10 // distance to the owner before an idle minion follows
	minionThink   = 20 // frames between target choices
	allyCorpseSec = 1.5
)

// MinionOptions describe a summon.
type MinionOptions struct {
	Owner *d2mapentity.Player
	// Kind is "minion", "trap", "totem" or "wall"; only minions move.
	Kind string
	// HPPct is added to the monster's life in percent, HPFlat in points.
	HPPct, HPFlat int
	// DamagePct scales the monster's damage, ToHit and ArmorClass are added to
	// its attack rating and defense (the skill's aurastat columns).
	DamagePct, ToHit, ArmorClass int
	// Frames is the lifetime (0: until killed).
	Frames int
	// Level overrides the monster level (0: the owner's level).
	Level int
	// Tag is free for the caller (the skill name).
	Tag string
	// Stats, when set, replace the monstats-derived life, defense, attack
	// rating and damage (d2summon.Compute); HPPct/HPFlat/DamagePct/ToHit/
	// ArmorClass are then ignored for those numbers.
	Stats *d2summon.Stats
}

type allyState struct {
	owner    *d2mapentity.Player
	kind     string
	opt      MinionOptions
	until    int
	target   *unit
	strikeAt *unit
	thought  int
}

// SpawnMinion creates an allied unit near a subtile.
func (d *Director) SpawnMinion(stat *d2records.MonStatRecord, subX, subY int, opt MinionOptions) (*d2mapentity.Monster, error) {
	p, ok := d2path.NearestFree(d.grid, d2path.MaskMonster, d2path.Point{X: subX, Y: subY}, 6)
	if !ok {
		return nil, fmt.Errorf("no free cell near (%d,%d)", subX, subY)
	}

	if opt.Kind == "" {
		opt.Kind = "minion"
	}

	st := &allyState{owner: opt.Owner, kind: opt.Kind, opt: opt}
	if opt.Frames > 0 {
		st.until = d.frame + opt.Frames
	}

	// the minion level follows its owner (UNVERIFIED)
	if lvl := opt.Level; lvl > 0 {
		d.forceLevel = lvl
	} else if opt.Owner != nil {
		d.forceLevel = opt.Owner.Stats.Level
	}

	m, err := d.spawn(stat, p.X, p.Y, st)
	d.forceLevel = 0

	if err != nil {
		return nil, err
	}

	v := &m.Vitals

	if s := opt.Stats; s != nil {
		v.MaxHP, v.Defense = s.MaxHP, s.Defense
		v.A1 = d2mapentity.MonsterAttack{ToHit: s.AR, Min: s.DmgMin, Max: s.DmgMax}
		v.HP = v.MaxHP
	} else {
		v.MaxHP += v.MaxHP*opt.HPPct/100 + opt.HPFlat
		v.HP = v.MaxHP
		v.Defense += opt.ArmorClass
	}

	v.Experience = 0
	v.TreasureClass = ""

	d.Counters.Minions++
	d.emit("minion", "MINION spawn name=%s key=%s kind=%s owner=%s level=%d hp=%d pos=(%d,%d) tag=%s", m.Label(), stat.Key,
		opt.Kind, ownerName(opt.Owner), v.Level, v.MaxHP, p.X, p.Y, opt.Tag)

	return m, nil
}

func ownerName(p *d2mapentity.Player) string {
	if p == nil {
		return "-"
	}

	return p.Name()
}

// Minions returns the living allied units.
func (d *Director) Minions() []*d2mapentity.Monster {
	var out []*d2mapentity.Monster

	for _, u := range d.sortedUnits() {
		if u.ally != nil && u.m.Alive() {
			out = append(out, u.m)
		}
	}

	return out
}

// MinionKind returns the kind and tag of a minion ("" if m is not one).
func (d *Director) MinionKind(m *d2mapentity.Monster) (kind, tag string) {
	if u := d.byEntity[m.ID()]; u != nil && u.ally != nil {
		return u.ally.kind, u.ally.opt.Tag
	}

	return "", ""
}

// Corpses returns the dead hostile monsters whose corpse is still there.
func (d *Director) Corpses() []*d2mapentity.Monster {
	var out []*d2mapentity.Monster

	for _, u := range d.sortedUnits() {
		if !u.friendly() && !u.m.Alive() {
			out = append(out, u.m)
		}
	}

	return out
}

// RemoveCorpse takes a corpse off the map (Raise Skeleton, Corpse Explosion,
// Redemption).
func (d *Director) RemoveCorpse(m *d2mapentity.Monster) {
	if u := d.byEntity[m.ID()]; u != nil && !u.m.Alive() {
		d.engine.RemoveEntity(u.m)
		d.forget(u)
	}
}

// Kill removes a minion at once (Unsummon, lifetime end).
func (d *Director) Kill(m *d2mapentity.Monster) {
	if u := d.byEntity[m.ID()]; u != nil && u.ally != nil {
		d.expire(u)
	}
}

func (d *Director) expire(u *unit) {
	if u.m.Alive() {
		u.m.Die()
	}

	d.fp.Remove(u.b.ID)
	d.engine.RemoveEntity(u.m)
	d.forget(u)
	d.emit("minion", "MINION end name=%s", u.m.Label())
}

// ---- stun, fear and slow (skills states) ----

// Stun holds a monster still for frames (stun and freeze).
func (d *Director) Stun(m *d2mapentity.Monster, frames int) {
	if u := d.byEntity[m.ID()]; u != nil && m.Alive() && frames > 0 {
		if end := d.frame + frames; end > u.stunUntil {
			u.stunUntil = end
		}

		u.m.StopMoving()
		u.m.DropHitEvents()
		u.mv = nil
	}
}

// Flee makes a monster run away from the heroes for frames (Terror, Howl).
func (d *Director) Flee(m *d2mapentity.Monster, frames int) {
	if u := d.byEntity[m.ID()]; u != nil && m.Alive() && frames > 0 {
		if end := d.frame + frames; end > u.fleeUntil {
			u.fleeUntil = end
		}

		u.lastFlee = 0
	}
}

// SetSlow changes a monster's movement speed in percent.
func (d *Director) SetSlow(m *d2mapentity.Monster, pct int) {
	if u := d.byEntity[m.ID()]; u != nil && u.slowPct != pct {
		u.slowPct = pct
		m.SetSlow(pct)
	}
}

// Held reports whether a monster is stunned or afraid right now.
func (d *Director) Held(m *d2mapentity.Monster) (stunned, afraid bool) {
	if u := d.byEntity[m.ID()]; u != nil {
		return u.stunUntil > d.frame, u.fleeUntil > d.frame
	}

	return false, false
}

// held runs the stun and fear state of a hostile monster; true means its AI
// must not run this frame.
func (d *Director) held(u *unit) bool {
	if u.stunUntil > d.frame {
		return true
	}

	if u.stunUntil != 0 {
		u.stunUntil = 0
		u.b.WakeNow(d.frame)
	}

	if u.fleeUntil > d.frame {
		d.flee(u)

		return true
	}

	if u.fleeUntil != 0 {
		u.fleeUntil = 0
		u.m.StopMoving()
		u.b.WakeNow(d.frame)
	}

	return false
}

func (d *Director) flee(u *unit) {
	if d.frame-u.lastFlee < 8 && u.m.Moving() {
		return
	}

	u.lastFlee = d.frame

	var players []*d2mapentity.Player
	if d.players != nil {
		players = d.players()
	}

	if len(players) == 0 {
		return
	}

	sx, sy := u.m.SubtilePos()
	px, py := playerSubtile(players[0])
	dx, dy := float64(sx-px), float64(sy-py)

	n := math.Hypot(dx, dy)
	if n < 1 {
		dx, dy, n = 1, 0, 1
	}

	dest, ok := d2path.NearestFree(d.grid, d2path.MaskMonster, d2path.Point{X: sx + int(dx/n*12), Y: sy + int(dy/n*12)}, 6)
	if !ok {
		return
	}

	d.moveTo(u, d2monster.Point{X: dest.X, Y: dest.Y}, nil, 0, true)
}

// ---- minion AI ----

// petTick runs the ported pet AI for a minion that has one and reports whether
// it did (the caller then skips the generic nearest-hostile minion logic).
func (d *Director) petTick(u *unit) bool {
	if !usesPetAI(u) {
		return false
	}

	d.followIntent(u)
	d2monster.Tick(d, u.b)

	return true
}

func (d *Director) allyStep(u *unit) {
	a := u.ally
	m := u.m

	if !m.Alive() {
		if m.CorpseAge() > allyCorpseSec {
			d.engine.RemoveEntity(m)
			d.forget(u)
		}

		return
	}

	if a.until > 0 && d.frame >= a.until {
		d.expire(u)

		return
	}

	if a.kind != "minion" || d.held(u) {
		return
	}

	switch m.Mode() {
	case d2monster.ModeNeutral, d2monster.ModeWalk, d2monster.ModeRun:
	default:
		return
	}

	if d.petTick(u) {
		return
	}

	sx, sy := m.SubtilePos()

	if a.target == nil || !a.target.m.Alive() || d.frame-a.thought >= minionThink {
		a.thought = d.frame
		a.target = d.nearestHostile(sx, sy)
	}

	if t := a.target; t != nil {
		tx, ty := t.m.SubtilePos()

		if d2monster.EdgeDistance(sx-tx, sy-ty, t.b.Size) <= meleeInRange {
			m.StopMoving()
			m.Face(float64(tx), float64(ty))

			if m.SetMode(d2monster.ModeAttack1) {
				a.strikeAt = t
			}

			return
		}

		if !m.Moving() || u.mv == nil || abs(tx-u.mv.plannedAtX) >= 3 || abs(ty-u.mv.plannedAtY) >= 3 {
			d.moveTo(u, d2monster.Point{X: tx, Y: ty}, &d2monster.Target{X: tx, Y: ty, Size: t.b.Size}, meleeInRange, true)
		}

		return
	}

	if a.owner == nil {
		return
	}

	ox, oy := playerSubtile(a.owner)
	if d2monster.Distance(sx-ox, sy-oy) > minionLeash && !m.Moving() {
		d.moveTo(u, d2monster.Point{X: ox, Y: oy}, &d2monster.Target{X: ox, Y: oy, IsPlayer: true}, 5, false)
	}
}

// nearestHostile finds the closest living hostile monster within aggro range.
func (d *Director) nearestHostile(x, y int) *unit {
	var best *unit

	bd := minionAggro + 1

	for _, h := range d.sortedUnits() {
		if h.friendly() || !h.m.Alive() {
			continue
		}

		hx, hy := h.m.SubtilePos()
		if dist := d2monster.Distance(hx-x, hy-y); dist < bd {
			best, bd = h, dist
		}
	}

	return best
}

// allyStrike resolves a minion attack at its hit frame.
func (d *Director) allyStrike(u *unit, _ d2monster.Mode) {
	a := u.ally
	t := a.strikeAt

	if t == nil || !t.m.Alive() {
		return
	}

	sx, sy := u.m.SubtilePos()
	tx, ty := t.m.SubtilePos()
	d.Counters.MinionAttacks++

	if d2monster.EdgeDistance(sx-tx, sy-ty, t.b.Size) > meleeInRange+heroReach/2 {
		return
	}

	atk := u.m.Vitals.A1
	if atk.Max < 1 {
		atk = d2mapentity.MonsterAttack{ToHit: atk.ToHit, Min: 1, Max: 3}
	}

	ar := d2combat.MonsterAttackRating(atk.ToHit, 0, 0) + a.opt.ToHit
	hit, chance, roll := d2combat.RollToHit(u.b.Seed, d2combat.ToHitInput{
		AttackRating: ar, Defense: t.m.Vitals.Defense, AttackerLevel: u.m.Vitals.Level, DefenderLevel: t.m.Vitals.Level,
	})

	dmg := 0

	if hit {
		dmg = atk.Min + int(u.b.Seed.Roll(int32(atk.Max-atk.Min+1)))
		dmg += dmg * a.opt.DamagePct / 100

		if dmg < 1 {
			dmg = 1
		}

		d.Counters.MinionHits++
	}

	d.emit("minion", "MINION attack name=%s target=%s hit=%v chance=%d roll=%d dmg=%d", u.m.Label(), t.m.Label(), hit, chance,
		roll, dmg)

	if hit {
		d.damage(t, a.owner, dmg)
	}
}

// friendly reports whether a unit fights for a hero: a hired mercenary (kept
// across games, has an owner and a save) or a summoned minion (owned by a
// skill, temporary). Both are excluded from Monsters and from the targets of
// the other friendly units; they keep their own listings (Merc, Minions).
func (u *unit) friendly() bool { return u.merc != nil || u.ally != nil }

// friendlyStrike resolves the attack of a friendly unit at its hit frame.
func (d *Director) friendlyStrike(u *unit, mode d2monster.Mode) {
	if u.merc != nil {
		d.mercStrike(u, mode)

		return
	}

	d.allyStrike(u, mode)
}
