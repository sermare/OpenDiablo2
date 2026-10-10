package d2monsters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// unitTargetBase offsets the target ids of monsters that other monsters can
// hunt (converted monsters, Attract victims, anybody a confused monster picks);
// players use 1..n and mercenaries mercTargetBase+id.
const unitTargetBase = 1 << 17

// confusedRadius is the search radius of a confused monster (the 0x23 = 35
// passed to FUN_005dbe00 by FUN_005dc370, VERIFIED).
const confusedRadius = 35

// ForceState puts a forced condition on a monster for frames game frames (the
// monster side of Terror/Howl, Dim Vision, Taunt, Confuse, Attract and
// Conversion; the skill pipeline calls it, the debug command
// "forcestate <id> <state> <frames>" does too). source is the target id of the
// unit that caused it (0 = the first hero). It returns the monster's AI label
// after the change.
func (d *Director) ForceState(id uint32, kind d2monster.ForcedKind, frames int, source uint32) (string, error) {
	u := d.units[id]

	switch {
	case u == nil:
		return "", fmt.Errorf("no monster with id %d", id)
	case u.merc != nil:
		return "", errors.New("mercenaries are not affected by monster states")
	case !u.m.Alive():
		return "", errors.New("the monster is dead")
	case kind == d2monster.ForcedNone:
		return "", errors.New("unknown state")
	}

	if source == 0 {
		source = 1
	}

	before := u.b.StateLabel()

	// whatever it was walking to is forgotten; a blow in progress finishes
	if m := u.m.Mode(); m == d2monster.ModeWalk || m == d2monster.ModeRun {
		u.m.StopMoving()
	}

	u.mv = nil
	u.b.ApplyForced(kind, d.frame, frames, source)
	d.emit("state", "MONSTER state name=%s id=%d kind=%s frames=%d until=%d source=%d unit_state=%d ai=%s->%s",
		u.m.Label(), u.b.ID, kind, frames, u.b.ForcedUntil, source, kind.UnitStateID(), before, u.b.StateLabel())

	return u.b.StateLabel(), nil
}

// ForceMonster is ForceState for a monster entity (the skill engine holds
// entities, not brain ids).
func (d *Director) ForceMonster(m *d2mapentity.Monster, kind d2monster.ForcedKind, frames int, source uint32) (string, error) {
	u := d.byEntity[m.ID()]
	if u == nil {
		return "", fmt.Errorf("monster %s is not run by the director", m.ID())
	}

	return d.ForceState(u.b.ID, kind, frames, source)
}

// FindArchetype returns the lowest-numbered enabled monstats row whose AI is
// the named archetype (case-insensitive, e.g. "Vulture"), or nil.
func (d *Director) FindArchetype(ai string) *d2records.MonStatRecord {
	var best *d2records.MonStatRecord

	for _, st := range d.asset.Records.Monster.Stats {
		if st.Enabled && strings.EqualFold(st.AiKey, ai) && (best == nil || st.ID < best.ID) {
			best = st
		}
	}

	return best
}

// traceState logs every change of a monster's AI state (the forced states and
// their expiry): "MONSTER aistate ... from=A to=B".
func (d *Director) traceState(u *unit) {
	label := u.b.StateLabel()
	if u.b.Allied {
		label += "+allied"
	}

	if u.b.Attracting {
		label += "+decoy"
	}

	if label == u.lastLabel {
		return
	}

	if u.lastLabel != "" {
		d.emit("aistate", "MONSTER aistate name=%s id=%d frame=%d from=%s to=%s", u.m.Label(), u.b.ID, d.frame, u.lastLabel, label)
	}

	u.lastLabel = label
}

// ---- optional d2monster World extensions ----

// UnitTarget implements d2monster.SourceFinder.
func (d *Director) UnitTarget(b *d2monster.Brain, id uint32) (d2monster.Target, int, bool) {
	t := d2monster.Target{ID: id, Size: 1}

	switch {
	case id == 0:
		return t, 0, false
	case id >= unitTargetBase:
		o := d.units[id-unitTargetBase]
		if o == nil || !o.m.Alive() {
			return t, 0, false
		}

		t.X, t.Y = o.m.SubtilePos()
	case id >= mercTargetBase:
		o := d.units[id-mercTargetBase]
		if o == nil || o.merc == nil || !o.m.Alive() {
			return t, 0, false
		}

		t.X, t.Y = o.m.SubtilePos()
	default:
		p := d.targets[id]
		if p == nil || !d.targetable(p) {
			return t, 0, false
		}

		t.X, t.Y = playerSubtile(p)
		t.IsPlayer = true
	}

	return t, d2monster.EdgeDistance(b.X-t.X, b.Y-t.Y, b.Size), true
}

// NearestAny implements d2monster.ConfusedSenses: the nearest hero, merc or
// monster other than the confused one within the search radius.
func (d *Director) NearestAny(b *d2monster.Brain) (d2monster.Target, int, bool) {
	var (
		best     d2monster.Target
		bestDist = confusedRadius + 1
		found    bool
	)

	consider := func(t d2monster.Target) {
		if dist := d2monster.EdgeDistance(b.X-t.X, b.Y-t.Y, b.Size); dist < bestDist {
			best, bestDist, found = t, dist, true
		}
	}

	for _, o := range d.sortedUnits() {
		if o.b == b || !o.m.Alive() {
			continue
		}

		x, y := o.m.SubtilePos()
		id := unitTargetBase + o.b.ID

		if o.merc != nil {
			id = mercTargetBase + o.b.ID
		}

		consider(d2monster.Target{ID: id, X: x, Y: y, Size: 1})
	}

	for id := uint32(1); id <= uint32(len(d.targets)); id++ {
		if p := d.targets[id]; p != nil && d.targetable(p) {
			x, y := playerSubtile(p)
			consider(d2monster.Target{ID: id, X: x, Y: y, Size: 1, IsPlayer: true})
		}
	}

	return best, bestDist, found
}

// NearestDecoy implements d2monster.DecoyFinder: the nearest monster under
// Attract.
func (d *Director) NearestDecoy(b *d2monster.Brain) (d2monster.Target, int, bool) {
	var (
		best     d2monster.Target
		bestDist int
		found    bool
	)

	for _, o := range d.sortedUnits() {
		if o.b == b || !o.b.Attracting || !o.m.Alive() {
			continue
		}

		x, y := o.m.SubtilePos()
		dist := d2monster.EdgeDistance(b.X-x, b.Y-y, b.Size)

		if !found || dist < bestDist {
			best, bestDist, found = d2monster.Target{ID: unitTargetBase + o.b.ID, X: x, Y: y, Size: 1}, dist, true
		}
	}

	return best, bestDist, found
}

// InTown implements d2monster.TownChecker: hostile monsters are never placed
// in town here.
func (d *Director) InTown(*d2monster.Brain) bool { return false }

// HasMode implements d2monster.ModeChecker (the monstats2 mode bits): a class
// runs when it has a run velocity; the other modes are assumed present (a
// missing animation makes the request fail and the AI sleeps instead).
func (d *Director) HasMode(b *d2monster.Brain, m d2monster.Mode) bool {
	if m == d2monster.ModeRun {
		return b.Profile.Run > 0
	}

	return true
}

// ClearState implements d2monster.StateClearer (unit states are not modelled).
func (d *Director) ClearState(*d2monster.Brain, int) {}

// ---- monster against monster ----

// strikeUnit resolves a monster's attack on another monster (a converted
// monster against a hostile one or the other way round, or a confused one
// against whoever stands next to it). It uses the same to-hit roll as the
// attacks on mercenaries; damage credit goes to the hero who owns a converted
// attacker.
func (d *Director) strikeUnit(u, tu *unit, atk d2mapentity.MonsterAttack, mode d2monster.Mode) {
	tx, ty := tu.m.SubtilePos()
	sx, sy := u.m.SubtilePos()
	dist := d2monster.EdgeDistance(sx-tx, sy-ty, u.b.Size)

	reach := meleeInRange
	if attackIsRanged(u.m.Stat, mode) {
		reach = rangedInRange
	}

	d.Counters.Attacks++

	if dist > reach+heroReach/2 {
		d.emit("attack", "MONSTER attack name=%s id=%d mode=%s hit=false reason=target_moved_away dist=%d", u.m.Label(), u.b.ID, mode, dist)

		return
	}

	defLevel := tu.m.Vitals.Level
	if tu.merc != nil {
		defLevel = tu.merc.level
	}

	in := d2combat.ToHitInput{
		AttackRating:  d2combat.MonsterAttackRating(atk.ToHit, 0, 0),
		Defense:       tu.m.Vitals.Defense,
		AttackerLevel: u.m.Vitals.Level,
		DefenderLevel: defLevel,
	}

	hit, chance, roll := d2combat.RollToHit(u.b.Seed, in)
	dmg := 0

	if hit {
		span := atk.Max - atk.Min + 1
		if span < 1 {
			span = 1
		}

		dmg = atk.Min + int(u.b.Seed.Roll(int32(span)))
		if dmg < 1 {
			dmg = 1
		}

		d.Counters.AttackHits++
	}

	d.Counters.UnitFights++
	d.emit("attack", "MONSTER attack name=%s id=%d mode=%s target=%s(%d) hit=%v chance=%d roll=%d dmg=%d target_hp=%d/%d",
		u.m.Label(), u.b.ID, mode, tu.m.Label(), tu.b.ID, hit, chance, roll, dmg, maxInt(tu.m.Vitals.HP-dmg, 0), tu.m.Vitals.MaxHP)

	if !hit {
		return
	}

	if tu.merc != nil {
		d.damageMerc(tu, d.takenByMerc(tu, dmg), u.m.Label())

		return
	}

	var credit *d2mapentity.Player
	if u.b.Allied {
		credit = d.targets[u.b.OwnerID]
	}

	d.damage(tu, credit, dmg)
}
