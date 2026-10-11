package d2mp

import (
	"math"
	"sort"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

const (
	meleeReach    = 1.8
	interactReach = 2.4
	pathEveryMs   = 300
	corpseMs      = 2500
	monsterRadius = 0.4
	waypointReach = 5.0
)

// Tick advances the simulation to server time now (ms) in TickMs steps.
func (s *Sim) Tick(now uint32) {
	for s.now+s.cfg.TickMs <= now {
		s.now += s.cfg.TickMs
		s.step()
	}
}

func (s *Sim) step() {
	s.runTimers()
	s.advancePaths()

	for _, p := range s.sortedPlayers() {
		s.playerGoal(p)
	}

	for _, id := range s.sortedMonsterIDs() {
		m := s.mons[id]
		if !m.u.Dead && s.hasPlayers(m.u.Level) {
			s.monsterThink(m)
		}
	}

	for _, id := range s.sortedMissileIDs() {
		s.missileStep(s.mis[id])
	}
}

func (s *Sim) sortedMonsterIDs() []uint32 {
	out := make([]uint32, 0, len(s.mons))
	for id := range s.mons {
		out = append(out, id)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}

func (s *Sim) sortedMissileIDs() []uint32 {
	out := make([]uint32, 0, len(s.mis))
	for id := range s.mis {
		out = append(out, id)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}

func (s *Sim) after(ms uint32, fn func()) { s.timers = append(s.timers, timer{s.now + ms, fn}) }

func (s *Sim) runTimers() {
	var keep []timer

	due := s.timers
	s.timers = nil

	for _, t := range due {
		if t.at <= s.now {
			t.fn()
		} else {
			keep = append(keep, t)
		}
	}

	s.timers = append(keep, s.timers...)
}

// ---- player commands ----

// Walk orders a hero to walk or run to a location (tiles).
func (s *Sim) Walk(id uint32, run bool, x, y float64) {
	p, ok := s.pl[id]
	if !ok || p.u.Dead {
		return
	}

	p.goal, p.target = goalNone, 0
	s.cancelTrade(id, "moved away")

	speed := s.cfg.WalkSpeed
	if run {
		speed = s.cfg.RunSpeed
	}

	s.walkTo(p.u, x, y, speed)
}

// SelectSkill sets the skill of a hand.
func (s *Sim) SelectSkill(id uint32, skill uint16, right bool) {
	p, ok := s.pl[id]
	if !ok {
		return
	}

	if right {
		p.right = skill
	} else {
		p.left = skill
	}
}

// Cast uses the selected skill of a hand on a location (tiles).
func (s *Sim) Cast(id uint32, right bool, x, y float64) {
	p, ok := s.pl[id]
	if !ok || p.u.Dead {
		return
	}

	skill := p.left
	if right {
		skill = p.right
	}

	s.castSkill(p, skill, x, y)
}

func (s *Sim) castSkill(p *pstate, skill uint16, x, y float64) {
	def, ok := s.rules.Skill(skill)
	if !ok {
		s.msg(p.u.ID, "unknown skill %d", skill)

		return
	}

	switch def.Kind {
	case SkillMelee:
		var best *Unit

		bd := 1.5

		for _, id := range s.sortedMonsterIDs() {
			m := s.mons[id]
			if m.u.Dead || m.u.Level != p.u.Level {
				continue
			}

			mx, my := s.pos(m.u)
			if d := dist(mx, my, x, y); d < bd {
				best, bd = m.u, d
			}
		}

		if best == nil {
			s.Walk(p.u.ID, false, x, y)

			return
		}

		p.goal, p.target = goalAttack, best.ID
	case SkillMissile:
		if s.now < p.nextCast {
			return
		}

		p.nextCast = s.now + def.Cooldown
		p.goal, p.target = goalNone, 0
		s.cancelTrade(p.u.ID, "busy")
		s.stopAt(p.u)
		s.castMissile(p, skill, def, x, y)
	case SkillPortal:
		if s.now < p.nextCast {
			return
		}

		p.nextCast = s.now + def.Cooldown
		s.stopAt(p.u)
		s.castPortal(p)
	}
}

func (s *Sim) castMissile(p *pstate, skill uint16, def SkillDef, x, y float64) {
	px, py := s.pos(p.u)
	x0, y0 := Snap(px), Snap(py)
	dx, dy := x-px, y-py

	if n := math.Hypot(dx, dy); n < 0.01 {
		dx, dy = 1, 0
	} else {
		dx, dy = dx/n, dy/n
	}

	ex, ey := s.march(s.levelDef(p.u.Level), x0, y0, px+dx*def.Range, py+dy*def.Range)
	u := &Unit{ID: s.newID(), Kind: KindMissile, Type: skill, Level: p.u.Level, Owner: p.u.ID,
		Segs: []Seg{{X0: x0, Y0: y0, X1: ex, Y1: ey, Speed: def.Speed, T0: s.now}}}
	s.units[u.ID] = u
	s.mis[u.ID] = &misstate{u: u, def: def, owner: p.u.ID, last: s.now}
	s.Stats.Casts++

	s.toLevel(p.u.Level, Event{Type: EvAttack, ID: p.u.ID, A: int32(skill), B: int32(Sub(x)) | int32(Sub(y))<<16}, 0)
	s.toLevel(p.u.Level, s.spawnEvent(u), 0)
}

func segPointDist(ax, ay, bx, by, px, py float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy

	if l2 == 0 {
		return dist(ax, ay, px, py)
	}

	t := math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l2))

	return dist(ax+t*dx, ay+t*dy, px, py)
}

func (s *Sim) removeUnit(u *Unit) {
	delete(s.units, u.ID)
	delete(s.mis, u.ID)
	delete(s.mons, u.ID)
	s.toLevel(u.Level, Event{Type: EvRemove, ID: u.ID}, 0)
}

func (s *Sim) missileStep(m *misstate) {
	seg := m.u.Segs[0]
	t := float64(s.now)
	ax, ay := seg.PosAt(float64(m.last))
	bx, by := seg.PosAt(t)
	m.last = s.now

	for _, id := range s.sortedMonsterIDs() {
		mo := s.mons[id]
		if mo.u.Dead || mo.u.Level != m.u.Level {
			continue
		}

		mx, my := s.pos(mo.u)
		if segPointDist(ax, ay, bx, by, mx, my) > m.def.Radius+monsterRadius {
			continue
		}

		// impact
		hit := []*mstate{mo}

		if m.def.Splash > 0 {
			hit = nil

			for _, id2 := range s.sortedMonsterIDs() {
				o := s.mons[id2]
				if o.u.Dead || o.u.Level != m.u.Level {
					continue
				}

				ox, oy := s.pos(o.u)
				if dist(ox, oy, mx, my) <= m.def.Splash {
					hit = append(hit, o)
				}
			}
		}

		for _, o := range hit {
			s.damage(o.u, m.owner, m.def.Min+s.rng.Intn(m.def.Max-m.def.Min+1))
		}

		s.removeUnit(m.u)

		return
	}

	if !seg.Moving(t) {
		s.removeUnit(m.u)
	}
}

// ---- damage and death ----

func (s *Sim) damage(t *Unit, attacker uint32, dmg int) {
	if t.Dead || dmg <= 0 {
		return
	}

	if t.Kind == KindPlayer {
		if p := s.pl[t.ID]; p != nil && p.protectUntil > s.now {
			return // state 0x6c (levelmove.go)
		}
	}

	t.HP -= int32(dmg)
	if t.HP < 0 {
		t.HP = 0
	}

	s.Stats.Hits++
	s.toLevel(t.Level, Event{Type: EvHit, ID: t.ID, Other: attacker, A: int32(dmg), B: t.HP}, 0)

	if t.HP == 0 {
		s.kill(t, attacker)
	}
}

func (s *Sim) kill(t *Unit, killer uint32) {
	t.Dead = true
	s.stopAt(t)
	s.toLevel(t.Level, Event{Type: EvDeath, ID: t.ID, Other: killer}, 0)

	switch t.Kind {
	case KindMonster:
		s.Stats.Kills++
		m := s.mons[t.ID]

		for _, p := range s.sortedPlayers() {
			if p.target == t.ID {
				p.goal, p.target = goalNone, 0
			}
		}

		if _, ok := s.pl[killer]; ok {
			s.awardKill(killer, m.def)
		}

		x, y := s.pos(t)
		for i, code := range s.rules.Drops(s.rng, m.def) {
			s.dropItem(code, t.Level, x+float64(i)*0.6, y)
		}

		s.after(corpseMs, func() { s.removeUnit(t) })
	case KindPlayer:
		s.Stats.PlayerDeaths++
		p := s.pl[t.ID]
		p.goal, p.target = goalNone, 0
		s.cancelTrade(t.ID, "died")

		for _, m := range s.mons {
			if m.target == t.ID {
				m.target = 0
			}
		}
	}
}

func (s *Sim) awardKill(killer uint32, def MonsterDef) {
	for _, sh := range s.roster.ShareKillXP(sid(killer), def.XP, def.Level, s.cfg.MaxLevel) {
		n, _ := strconv.ParseUint(sh.ID, 10, 32)
		id := uint32(n)
		p, ok := s.pl[id]
		if !ok || sh.XP <= 0 {
			continue
		}

		p.u.XP += uint32(sh.XP)
		s.send(id, Event{Type: EvXP, ID: id, A: int32(sh.XP), B: int32(p.u.XP)})
	}
}

func (s *Sim) dropItem(code string, level uint16, x, y float64) *Unit {
	lv := s.levelDef(level)
	x, y = Snap(x), Snap(y)

	if !lv.Walkable(x, y) {
		x, y = lv.SpawnX, lv.SpawnY
	}

	u := &Unit{ID: s.newID(), Kind: KindItem, Name: code, Level: level, Segs: []Seg{{X0: x, Y0: y, X1: x, Y1: y}}}
	s.units[u.ID] = u
	s.Stats.Drops++
	s.toLevel(level, s.spawnEvent(u), 0)

	return u
}

// ---- goals: attack, interact, pick up ----

// Interact is the d2gs InteractUnit order: attack a monster, use an object.
func (s *Sim) Interact(id uint32, kind Kind, target uint32) {
	p, ok := s.pl[id]
	if !ok || p.u.Dead {
		return
	}

	t := s.units[target]
	if t == nil || t.Kind != kind || t.Level != p.u.Level {
		return
	}

	switch kind {
	case KindMonster:
		p.goal = goalAttack
	case KindObject:
		p.goal = goalInteract
	case KindItem:
		p.goal = goalPickup
	default:
		return
	}

	p.target = target
	p.nextPath = 0
}

// PickUp is the d2gs PickUpUnit order.
func (s *Sim) PickUp(id uint32, item uint32) { s.Interact(id, KindItem, item) }

func (s *Sim) playerGoal(p *pstate) {
	if p.goal == goalNone || p.u.Dead {
		return
	}

	t := s.units[p.target]
	if t == nil || t.Level != p.u.Level || (p.goal == goalAttack && t.Dead) {
		p.goal, p.target = goalNone, 0

		return
	}

	px, py := s.pos(p.u)
	tx, ty := s.pos(t)
	d := dist(px, py, tx, ty)

	reach := interactReach
	if p.goal == goalAttack {
		reach = meleeReach
	}

	if d > reach {
		if s.now >= p.nextPath {
			p.nextPath = s.now + pathEveryMs
			s.walkTo(p.u, tx, ty, s.cfg.RunSpeed)
		}

		return
	}

	if p.u.Moving(float64(s.now)) {
		s.stopAt(p.u)
	}

	switch p.goal {
	case goalAttack:
		if s.now < p.nextAtk {
			return
		}

		def, _ := s.rules.Skill(SkillAttack)
		p.nextAtk = s.now + def.Cooldown
		s.toLevel(p.u.Level, Event{Type: EvAttack, ID: p.u.ID, Other: t.ID, A: int32(SkillAttack)}, 0)
		s.damage(t, p.u.ID, p.meleeMin+s.rng.Intn(p.meleeMax-p.meleeMin+1))
	case goalInteract:
		p.goal, p.target = goalNone, 0
		s.useObject(p, t)
	case goalPickup:
		p.goal, p.target = goalNone, 0
		s.pickUp(p, t)
	}
}

func (s *Sim) pickUp(p *pstate, it *Unit) {
	if it.Kind != KindItem || it.Level == 0 {
		return
	}

	if it.Name == "gld" {
		p.u.Gold += uint32(10 + s.rng.Intn(40))
	} else {
		p.inv = append(p.inv, invItem{it.ID, it.Name})
	}

	s.toLevel(it.Level, Event{Type: EvRemove, ID: it.ID}, 0)

	if it.Name == "gld" {
		delete(s.units, it.ID)
	} else {
		it.Level, it.Owner, it.Segs = 0, p.u.ID, nil
	}

	s.sendInv(p)
}

// Drop puts an inventory item (by id) on the ground at the hero's feet.
func (s *Sim) Drop(id uint32, item uint32) {
	p, ok := s.pl[id]
	if !ok || p.u.Dead {
		return
	}

	for i, it := range p.inv {
		if it.id != item {
			continue
		}

		p.inv = append(p.inv[:i], p.inv[i+1:]...)
		u := s.units[item]
		x, y := s.pos(p.u)
		x, y = Snap(x+0.8), Snap(y)

		if !s.levelDef(p.u.Level).Walkable(x, y) {
			x, y = Snap(px(p.u, s)), Snap(py(p.u, s))
		}

		u.Level, u.Owner, u.Segs = p.u.Level, 0, []Seg{{X0: x, Y0: y, X1: x, Y1: y}}
		s.Stats.Drops++
		s.toLevel(u.Level, s.spawnEvent(u), 0)
		s.sendInv(p)

		return
	}
}

func px(u *Unit, s *Sim) float64 { x, _ := s.pos(u); return x }
func py(u *Unit, s *Sim) float64 { _, y := s.pos(u); return y }

// ---- objects, portals, waypoints ----

func (s *Sim) useObject(p *pstate, o *Unit) {
	switch o.Type {
	case ObjChest:
		if o.State == ObjOpen {
			return
		}

		o.State = ObjOpen
		s.toLevel(o.Level, Event{Type: EvObject, ID: o.ID, A: int32(ObjOpen)}, 0)

		x, y := s.pos(o)
		for i, code := range s.rules.ChestLoot(s.rng) {
			s.dropItem(code, o.Level, x+0.8+float64(i)*0.6, y+0.8)
		}
	case ObjWaypoint:
		if !p.wps[o.Level] {
			p.wps[o.Level] = true
			s.msg(p.u.ID, "waypoint %d activated", o.Level)
		}
	case ObjPortal:
		s.usePortal(p, o)
	}
}

func (s *Sim) partnerPortal(from *Unit, destLevel uint16) (x, y float64) {
	s.levelDef(destLevel)

	for _, u := range s.unitsIn(destLevel) {
		if u.Kind == KindObject && u.Type == ObjPortal && u.Dest == from.Level {
			px, py := s.pos(u)

			return px + 1.2, py
		}
	}

	lv := s.levelDef(destLevel)

	return lv.SpawnX, lv.SpawnY
}

func (s *Sim) usePortal(p *pstate, o *Unit) {
	if o.Dest == 0 {
		return
	}

	x, y := s.partnerPortal(o, o.Dest)
	s.cancelTrade(p.u.ID, "left the level")
	s.enterLevel(p, o.Dest, x, y, false)
}

func (s *Sim) removePortals(p *pstate) {
	for i, id := range p.tpPortals {
		if u, ok := s.units[id]; ok {
			s.removeUnit(u)
		}

		p.tpPortals[i] = 0
	}
}

func (s *Sim) castPortal(p *pstate) {
	level := p.u.Level
	town := uint16(d2level.TownPortalDestination(int(level)))

	if town == 0 || d2level.IsTown(int(level)) {
		s.msg(p.u.ID, "no town portal needed here")

		return
	}

	s.removePortals(p)

	x, y := s.pos(p.u)
	a := &Unit{ID: s.newID(), Kind: KindObject, Type: ObjPortal, Level: level, Dest: town, Name: p.u.Name}
	ax, ay := Snap(x+1.4), Snap(y)

	if !s.levelDef(level).Walkable(ax, ay) {
		ax, ay = Snap(x), Snap(y)
	}

	a.Segs = []Seg{{X0: ax, Y0: ay, X1: ax, Y1: ay}}

	tv := s.levelDef(town)
	b := &Unit{ID: s.newID(), Kind: KindObject, Type: ObjPortal, Level: town, Dest: level, Name: p.u.Name}
	bx, by := Snap(tv.SpawnX+2), Snap(tv.SpawnY-2.4)
	b.Segs = []Seg{{X0: bx, Y0: by, X1: bx, Y1: by}}

	s.units[a.ID], s.units[b.ID] = a, b
	p.tpPortals = [2]uint32{a.ID, b.ID}
	s.toLevel(level, s.spawnEvent(a), 0)
	s.toLevel(town, s.spawnEvent(b), 0)
}

// UseWaypoint takes a hero standing at a waypoint to another activated one.
func (s *Sim) UseWaypoint(id uint32, dest uint16) {
	p, ok := s.pl[id]
	if !ok || p.u.Dead {
		return
	}

	if _, hasBit := d2level.WaypointBit(int(dest)); !hasBit && !d2level.IsTown(int(dest)) {
		s.msg(id, "no waypoint in level %d", dest)

		return
	}

	if !p.wps[dest] {
		s.msg(id, "waypoint %d not activated", dest)

		return
	}

	px, py := s.pos(p.u)
	near := false

	for _, o := range s.unitsIn(p.u.Level) {
		if o.Kind == KindObject && o.Type == ObjWaypoint {
			ox, oy := s.pos(o)
			near = near || dist(px, py, ox, oy) <= waypointReach
		}
	}

	if !near {
		s.msg(id, "not at a waypoint")

		return
	}

	s.levelDef(dest)

	x, y := 0.0, 0.0

	for _, o := range s.unitsIn(dest) {
		if o.Kind == KindObject && o.Type == ObjWaypoint {
			ox, oy := s.pos(o)
			x, y = ox-1.6, oy

			break
		}
	}

	s.cancelTrade(id, "left the level")
	s.enterLevel(p, dest, x, y, false)
}

// Respawn brings a dead hero back to the town of its act with half its life.
func (s *Sim) Respawn(id uint32) {
	p, ok := s.pl[id]
	if !ok || !p.u.Dead {
		return
	}

	act := d2level.ActOfLevel(int(p.u.Level))
	town := uint16(d2level.ActStartLevel(act))

	if town == 0 {
		town = uint16(d2level.RogueEncampment)
	}

	p.u.Dead, p.u.HP = false, p.u.MaxHP/2
	s.enterLevel(p, town, 0, 0, false)
	s.toLevel(town, Event{Type: EvVitals, ID: id, A: p.u.HP, B: p.u.MaxHP}, 0)
}

// ---- monsters ----

func (s *Sim) monsterThink(m *mstate) {
	u := m.u
	mx, my := s.pos(u)

	var tgt *pstate

	if t, ok := s.pl[m.target]; ok && !t.u.Dead && t.u.Level == u.Level {
		tgt = t
	}

	if tgt == nil {
		m.target = 0
		best := m.def.Aggro

		for _, p := range s.sortedPlayers() {
			if p.u.Dead || p.u.Level != u.Level {
				continue
			}

			if d := dist(mx, my, px(p.u, s), py(p.u, s)); d < best {
				best, tgt = d, p
			}
		}

		if tgt == nil {
			return
		}

		m.target = tgt.u.ID
	}

	tx, ty := s.pos(tgt.u)
	d := dist(mx, my, tx, ty)

	switch {
	case d > m.def.Aggro*2.5:
		m.target = 0
		s.stopAt(u)
	case d <= meleeReach:
		if u.Moving(float64(s.now)) {
			s.stopAt(u)
		}

		if s.now >= m.nextAtk {
			m.nextAtk = s.now + m.def.Cooldown
			s.toLevel(u.Level, Event{Type: EvAttack, ID: u.ID, Other: tgt.u.ID}, 0)
			s.damage(tgt.u, u.ID, m.def.Min+s.rng.Intn(m.def.Max-m.def.Min+1))
		}
	case s.now >= m.nextPath:
		m.nextPath = s.now + pathEveryMs
		gx, gy := tx-(tx-mx)/d*1.2, ty-(ty-my)/d*1.2
		s.walkTo(u, gx, gy, m.def.Speed)
	}
}
