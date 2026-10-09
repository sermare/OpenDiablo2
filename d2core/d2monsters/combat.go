package d2monsters

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

var colorToken = regexp.MustCompile(`\[[a-z]+\]`)

// handleEvents consumes the animation events a monster produced since the
// last frame.
func (d *Director) handleEvents(u *unit) {
	blocked := false

	for _, ev := range u.m.TakeEvents() {
		switch ev.Kind {
		case d2mapentity.MonsterEventHitFrame:
			if u.friendly() { // mercenaries and summoned minions share the dispatch
				d.friendlyStrike(u, ev.Mode)

				continue
			}

			d.monsterStrike(u, ev.Mode)
		case d2mapentity.MonsterEventModeDone:
			u.mv = nil

			if u.m.Alive() && u.m.Mode() == d2monster.ModeNeutral {
				u.b.WakeNow(d.frame)
			}
		case d2mapentity.MonsterEventBlocked:
			blocked = true
			d.Counters.BlockedSteps++
		case d2mapentity.MonsterEventDied:
			// the corpse keeps its cell: flag 0x8000 (VERIFIED value), which is
			// in no block mask, so it does not block movement
			x, y := u.m.SubtilePos()
			d.fp.Move(u.b.ID, x, y, d2path.FlagCorpse)
		}
	}

	if !blocked {
		u.blocked = 0

		return
	}

	// A refused step: another unit stands in the way. After a few refusals the
	// monster stops and thinks again, which picks a new path or action.
	if u.blocked++; u.blocked >= blockedRetry {
		u.blocked = 0
		u.m.StopMoving()
		u.mv = nil
		u.b.WakeNow(d.frame)
	}
}

// attackFor picks the damage profile of an animation mode. Skill modes (S1..S4
// and SQ) carry no damage here: their skills.txt effects are not simulated.
func attackFor(v *d2mapentity.MonsterVitals, mode d2monster.Mode) (d2mapentity.MonsterAttack, bool) {
	switch mode {
	case d2monster.ModeAttack1:
		return v.A1, v.A1.Max > 0
	case d2monster.ModeAttack2:
		return v.A2, v.A2.Max > 0
	}

	return d2mapentity.MonsterAttack{}, false
}

// monsterStrike resolves a monster attack when its animation reaches the hit
// frame. Melee attacks hit the target if it is within the reach of the mode;
// attacks with a missile launch a Shot that has to find the hero on its way.
func (d *Director) monsterStrike(u *unit, mode d2monster.Mode) {
	if u.attackTarget >= corpseTargetBase {
		d.raiseCorpse(u, u.attackTarget-corpseTargetBase)

		return
	}

	atk, ok := attackFor(&u.m.Vitals, mode)
	if !ok {
		return
	}

	if u.attackTarget >= unitTargetBase {
		if tu := d.units[u.attackTarget-unitTargetBase]; tu != nil && tu.m.Alive() {
			d.strikeUnit(u, tu, atk, mode)
		}

		return
	}

	if u.attackTarget >= mercTargetBase {
		if tu := d.units[u.attackTarget-mercTargetBase]; tu != nil && tu.merc != nil && tu.m.Alive() {
			d.strikeMerc(u, tu, atk, mode)
		}

		return
	}

	if attackIsRanged(u.m.Stat, mode) {
		d.fireShot(u, mode, atk)

		return
	}

	p := d.playerFor(u.attackTarget)
	if p == nil || !d.targetable(p) {
		return
	}

	px, py := playerSubtile(p)
	sx, sy := u.m.SubtilePos()
	dist := d2monster.EdgeDistance(sx-px, sy-py, u.b.Size)

	d.Counters.Attacks++

	if dist > attackReach(false)+heroReach/2 {
		d.emit("attack", "MONSTER attack name=%s id=%d mode=%s hit=false reason=target_moved_away dist=%d",
			u.m.Label(), u.b.ID, mode, dist)

		return
	}

	d.resolveAttack(u, p, mode, atk, "")
}

// resolveAttack rolls to-hit and damage of an attack that reached a hero.
func (d *Director) resolveAttack(u *unit, p *d2mapentity.Player, mode d2monster.Mode, atk d2mapentity.MonsterAttack,
	via string) {
	defense := d2combat.Defense(0, p.Stats.Dexterity, 0)
	blockPct, physResist, reduce := 0, 0, 0

	// the hero's real values from her equipment (d2statlist): defense with items,
	// block chance with her shield, physical resistance and flat reduction
	if t := p.Stats.Totals; t != nil {
		defense, blockPct, physResist, reduce = t.Defense, t.BlockPct, t.PhysResist, t.DamageReduction
	}

	in := d2combat.ToHitInput{
		AttackRating:  d2combat.MonsterAttackRating(atk.ToHit, 0, 0),
		Defense:       defense,
		AttackerLevel: u.m.Vitals.Level,
		DefenderLevel: p.Stats.Level,
	}

	hit, chance, roll := d2combat.RollToHit(u.b.Seed, in)
	dmg := 0

	blocked, note := false, ""

	if hit && blockPct > 0 {
		// shield block comes after the to-hit roll (COMBAT_RollAttackOutcome); the hero
		// is not moving while she is struck here (the running /3 rule is not applied)
		if blocked = d2combat.RollShieldBlock(u.b.Seed, blockPct, false); blocked {
			hit = false
		}
	}

	if hit {
		dmg = atk.Min + int(u.b.Seed.Roll(int32(atk.Max-atk.Min+1)))
		if dmg < 1 {
			dmg = 1
		}

		// flat damage reduction first, then the physical resistance percent (cap 50);
		// the order of the two is UNVERIFIED. A hit never drops below 0 damage.
		dmg = d2combat.ApplyResist(dmg-reduce, physResist)
		if dmg < 0 {
			dmg = 0
		}

		// skill defenses run after the to-hit and shield block steps and the
		// armor reductions: Dodge/Avoid/Evade, Energy Shield, Bone Armor, Thorns
		if d.HeroDefense != nil {
			dmg, note = d.HeroDefense(p, u.m, via == "", dmg)
		}

		d.Counters.AttackHits++
		p.Stats.Health -= dmg

		if d.opt.OnHeroHit != nil {
			d.opt.OnHeroHit(p)
		}

		if p.Stats.Health < 0 {
			p.Stats.Health = 0
		}
	}

	d.emit("attack", "MONSTER attack name=%s id=%d mode=%s%s hit=%v chance=%d roll=%d dmg=%d hero_hp=%d/%d def=%d blocked=%v%s",
		u.m.Label(), u.b.ID, mode, via, hit, chance, roll, dmg, p.Stats.Health, p.Stats.MaxHealth, defense, blocked, note)

	if hit && p.Stats.Health == 0 {
		d.Counters.HeroDeaths++
		d.emit("herodeath", "HERO died name=%s killer=%s", p.Name(), u.m.Label())
	}
}

// SetLauncher replaces the projectile engine (nil restores the built-in
// straight bolt). The hook is the integration point for a missile package.
func (d *Director) SetLauncher(l Launcher) {
	if l == nil {
		l = newBoltLauncher(d.grid)
	}

	d.launcher = l
}

// shotCollideRadius is how far (Chebyshev, subtiles) from a hero's cell a
// shot still hits it (engine choice).
const shotCollideRadius = 1

// fireShot launches a projectile for a ranged attack. A target id of a hero
// aims at where the hero stands now; id 0 aims at the ground point of the
// request (Blood Raven fires at random spots near the hero).
func (d *Director) fireShot(u *unit, mode d2monster.Mode, atk d2mapentity.MonsterAttack) {
	sx, sy := u.m.SubtilePos()
	ax, ay := u.aimX, u.aimY

	if p := d.playerFor(u.attackTarget); p != nil && d.targetable(p) {
		ax, ay = playerSubtile(p)
	}

	var struck *d2mapentity.Player

	shot := Shot{
		Owner: u.b.ID, Mode: mode.String(), From: d2path.Point{X: sx, Y: sy}, To: d2path.Point{X: ax, Y: ay},
		Missile: missileFor(u.m.Stat, mode), Velocity: d.missileVelocity(missileFor(u.m.Stat, mode)),
		Collide: func(x, y int) bool {
			for _, tid := range d.targetIDs() {
				p := d.targets[tid]
				px, py := playerSubtile(p)
				if d.targetable(p) && abs(px-x) <= shotCollideRadius && abs(py-y) <= shotCollideRadius {
					struck = p

					return true
				}
			}

			return false
		},
		Impact: func(x, y int, hit bool) {
			if !hit || struck == nil {
				d.emit("shot", "MONSTER shot name=%s id=%d mode=%s result=miss at=(%d,%d)", u.m.Label(), u.b.ID, mode, x, y)

				return
			}

			d.Counters.ShotHits++
			d.Counters.Attacks++
			d.resolveAttack(u, struck, mode, atk, " shot")
		},
	}

	if d.launcher.Launch(shot) {
		d.Counters.Shots++
		d.emit("shot", "MONSTER shot name=%s id=%d mode=%s missile=%q from=(%d,%d) to=(%d,%d)",
			u.m.Label(), u.b.ID, mode, shot.Missile, sx, sy, ax, ay)

		return
	}

	// a shot at point blank (same subtile) cannot fly: it lands at once
	if p := d.playerFor(u.attackTarget); p != nil && d.targetable(p) {
		d.Counters.Attacks++
		d.resolveAttack(u, p, mode, atk, " shot")
	}
}

// missileVelocity is the flight speed in subtiles per frame: the missiles.txt
// Vel divided by 16 (VERIFIED conversion), or the default.
func (d *Director) missileVelocity(name string) float64 {
	if name != "" {
		if rec := d.asset.Records.GetMissileByName(name); rec != nil && rec.Velocity > 0 {
			return float64(rec.Velocity) / 16
		}
	}

	return DefaultShotVelocity
}

// HeroStrike resolves a hero melee swing against a monster at the swing's hit
// frame. The to-hit uses the hero's attack rating (d2combat.PlayerAttackRating
// with the class factor; equipment attack rating is not read) and the damage
// is the weapon's damage (or 1-2 bare handed). Returns true on a hit.
func (d *Director) HeroStrike(p *d2mapentity.Player, m *d2mapentity.Monster) bool {
	u := d.byEntity[m.ID()]
	if u == nil || !m.Alive() {
		return false
	}

	d.Counters.HeroSwings++

	st := d.asset.Records.Character.Stats[p.Class]
	ar := d2combat.PlayerAttackRating(0, p.Stats.Dexterity, st.ToHitFactor)

	if t := p.Stats.Totals; t != nil {
		ar = t.AttackRating // with the equipment's attack rating, dexterity and AR percent
	}

	hit, chance, roll := d2combat.RollToHit(d.heroRoller(), d2combat.ToHitInput{
		AttackRating: ar, Defense: m.Vitals.Defense,
		AttackerLevel: p.Stats.Level, DefenderLevel: m.Vitals.Level,
	})

	if !hit {
		d.emit("herohit", "HERO swing target=%s hit=false chance=%d roll=%d", m.Label(), chance, roll)

		return false
	}

	min, max := d.heroDamage(p)
	dmg := min + int(d.heroRoller().Roll(int32(max-min+1)))
	crit := false

	// deadly strike and critical strike both double physical damage (verified)
	if t := p.Stats.Totals; t != nil {
		if crit = d2combat.RollStrike(d.heroRoller(), d2combat.StrikeInput{
			SkipWeapon: true, CriticalChance: t.CriticalStrike, DeadlyChance: t.DeadlyStrike,
		}); crit {
			dmg *= 2
		}
	}

	d.Counters.HeroHits++

	d.emit("herohit", "HERO swing target=%s hit=true chance=%d roll=%d dmg=%d crit=%v ar=%d", m.Label(), chance, roll, dmg,
		crit, ar)
	d.damage(u, p, dmg)

	if d.opt.OnHeroStrike != nil {
		d.opt.OnHeroStrike(p)
	}

	return true
}

// PvPStrike is the result of a hero's melee swing at another hero.
type PvPStrike struct {
	Hit    bool
	Chance int
	Roll   int
	Crit   bool
	Raw    int // weapon damage before the player-versus-player scale
	Scaled int // after it (d2combat.PvPDamage); the defender subtracts its own resistances
}

// HeroStrikePlayer resolves a hero's melee swing at a hostile hero with the
// same pipeline as HeroStrike (attack rating, to-hit roll, weapon damage,
// deadly/critical strike), then applies the player-versus-player damage scale
// (17 percent, d2combat.PvPDamage). The defender's client applies the result
// (block, resistances, life); this resolves only the attacker's part.
func (d *Director) HeroStrikePlayer(att, def *d2mapentity.Player) PvPStrike {
	d.Counters.HeroSwings++

	st := d.asset.Records.Character.Stats[att.Class]
	ar := d2combat.PlayerAttackRating(0, att.Stats.Dexterity, st.ToHitFactor)

	if t := att.Stats.Totals; t != nil {
		ar = t.AttackRating
	}

	defense := d2combat.Defense(0, def.Stats.Dexterity, 0)
	if t := def.Stats.Totals; t != nil {
		defense = t.Defense
	}

	var out PvPStrike

	r := d.pvpRoller(att)

	out.Hit, out.Chance, out.Roll = d2combat.RollToHit(r, d2combat.ToHitInput{
		AttackRating: ar, Defense: defense, AttackerLevel: att.Stats.Level, DefenderLevel: def.Stats.Level,
	})

	if !out.Hit {
		return out
	}

	min, max := d.heroDamage(att)
	out.Raw = min + int(r.Roll(int32(max-min+1)))

	if t := att.Stats.Totals; t != nil {
		if out.Crit = d2combat.RollStrike(r, d2combat.StrikeInput{
			SkipWeapon: true, CriticalChance: t.CriticalStrike, DeadlyChance: t.DeadlyStrike,
		}); out.Crit {
			out.Raw *= 2
		}
	}

	out.Scaled = d2combat.PvPDamage(out.Raw)
	d.Counters.HeroHits++

	if d.opt.OnHeroStrike != nil {
		d.opt.OnHeroStrike(att)
	}

	return out
}

// pvpRoller is the random sequence of a hero's swings at other heroes. It is
// seeded with the game seed and the hero's id: two processes of one game share
// the game seed, and heroRoller alone would give both heroes the same rolls.
func (d *Director) pvpRoller(att *d2mapentity.Player) *d2rand.Seed {
	if d.pvp == nil {
		d.pvp = map[string]*d2rand.Seed{}
	}

	r := d.pvp[att.ID()]
	if r == nil {
		h := fnv.New32a()
		_, _ = h.Write([]byte(att.ID()))
		r = d2rand.New(d.opt.Seed ^ 0x70767000 ^ h.Sum32())
		d.pvp[att.ID()] = r
	}

	return r
}

func (d *Director) heroRoller() *d2rand.Seed {
	if d.hero == nil {
		d.hero = d2rand.New(d.opt.Seed ^ 0x6865726f)
	}

	return d.hero
}

// heroDamage is the weapon's damage range, or 1..2 bare handed (VERIFIED
// fallback min>=1, max>=2 in the damage build).
func (d *Director) heroDamage(p *d2mapentity.Player) (min, max int) {
	min, max = 1, 2

	if t := p.Stats.Totals; t != nil && t.DamageMax > 0 {
		return t.DamageMin, t.DamageMax // weapon, enhanced damage, added damage, strength bonus
	}

	if p.Equipment != nil && p.Equipment.RightHand != nil {
		if rec := d.asset.Records.Item.Weapons[p.Equipment.RightHand.GetItemCode()]; rec != nil && rec.MaxDamage > 0 {
			min, max = rec.MinDamage, rec.MaxDamage
		}
	}

	if min < 1 {
		min = 1
	}

	if max < min+1 {
		max = min + 1
	}

	return min, max
}

// Damage applies damage to a monster from a source hero (nil if unknown).
func (d *Director) Damage(m *d2mapentity.Monster, dmg int, src *d2mapentity.Player) {
	if u := d.byEntity[m.ID()]; u != nil {
		d.damage(u, src, dmg)
	}
}

// DamageOverTime applies poison or burn damage: like Damage but the monster is
// not interrupted (hit recovery) by the tick.
func (d *Director) DamageOverTime(m *d2mapentity.Monster, dmg int, src *d2mapentity.Player) {
	u := d.byEntity[m.ID()]
	if u == nil || !m.Alive() || dmg <= 0 {
		return
	}

	if u.m.Vitals.HP-dmg > 0 {
		u.m.Vitals.HP -= dmg

		return
	}

	u.m.Vitals.HP = 0
	d.kill(u, src)
}

func (d *Director) damage(u *unit, src *d2mapentity.Player, dmg int) {
	if !u.m.Alive() {
		return
	}

	u.m.Vitals.HP -= dmg

	if u.m.Vitals.HP > 0 {
		d.emit("hit", "MONSTER hit name=%s id=%d dmg=%d hp=%d/%d", u.m.Label(), u.b.ID, dmg, u.m.Vitals.HP, u.m.Vitals.MaxHP)
		d.playPlans(u, hitPlans(d.soundRecord(u)))

		// hit recovery: the monster drops what it was doing (a blow that was
		// winding up is lost), plays GH and thinks again when it ends. Aggro
		// is not otherwise changed. Whether every hit interrupts, and how long
		// the recovery lasts (monstats2 / aidel), is UNVERIFIED: every hit does.
		u.m.StopMoving()
		u.m.DropHitEvents()
		u.mv = nil
		d.Counters.HitRecoveries++

		if u.m.SetMode(d2monster.ModeGetHit) {
			u.b.Wake = 1 << 30
		} else {
			u.b.WakeNow(d.frame)
		}

		return
	}

	u.m.Vitals.HP = 0
	d.kill(u, src)
}

// awardKillXP gives the hero the experience of his kill (also the kills of his
// merc and pets, which are credited to the owner at full value) and returns the
// amount after the shrine bonus. There is NO level-difference penalty here: the
// original scales a kill by the character/monster level difference
// (0x0057c300, table near 0x006e2960, UNVERIFIED, see d2-re-notes xp notes), which
// is not modelled. Experience is capped later, at row MaxLvl-1 of Experience.txt
// (VERIFIED 0x0057c510, hero_levelup.go).
func (d *Director) awardKillXP(src *d2mapentity.Player, xp int, label string) int {
	if d.ExpBonusPct != nil { // shrine experience boost (d2object), percent
		xp += xp * d.ExpBonusPct() / 100
	}

	// in a network party the server splits the experience among the members
	// that share the level (d2party.ShareXP); the awards come back as packets
	if d.PartyXP == nil || !d.PartyXP(src, xp, label) {
		src.Stats.Experience += xp
	}

	return xp
}

func (d *Director) kill(u *unit, src *d2mapentity.Player) {
	u.m.Die()
	d.fp.Remove(u.b.ID) // a dying monster stops blocking (UNVERIFIED); the corpse flag is set when DT ends
	d.Counters.Deaths++
	d.playPlans(u, deathPlans(d.soundRecord(u)))
	d.leaderDied(u)

	by := "unknown"
	xp := u.m.Vitals.Experience

	if src != nil {
		by = src.Name()
		xp = d.awardKillXP(src, xp, u.m.Label())
	}

	if k := d.killer; k != nil {
		d.creditMerc(k.merc, k, xp)
	}

	d.emit("death", "MONSTER death name=%s id=%d by=%s xp=%d", u.m.Label(), u.b.ID, by, xp)
	d.dropLoot(u)

	if d.OnKill != nil {
		d.OnKill(KillEvent{Monster: u.m, Class: u.b.Class, Label: u.m.Label(), ByHero: src != nil})
	}
}

// dropLoot rolls the monster's treasure class with d2drop and puts the items
// on the ground around the corpse.
func (d *Director) dropLoot(u *unit) {
	tc := u.m.Vitals.TreasureClass
	if tc == "" {
		d.emit("drop", "MONSTER drop name=%s tc=- items=0", u.m.Label())

		return
	}

	level := u.m.Vitals.Level

	// the treasure class moves along its level group with the monster level
	// only in the expansion above Normal (VERIFIED, 0x558d80)
	upgrade := 0
	if d2difficulty.UpgradesTreasureClass(d.opt.Expansion, d2difficulty.Level(d.opt.Difficulty), true) {
		upgrade = level
	}

	loot, err := d.engine.DropLoot(tc, diablo2item.DropOptions{
		Seed: u.b.Seed.Step(), ILvl: level, UpgradeLevel: upgrade, Players: 1,
	}, 0)
	if err != nil {
		d.emit("drop", "MONSTER drop name=%s tc=%q error=%v", u.m.Label(), tc, err)

		return
	}

	names := make([]string, 0, len(loot.Entries))
	sx, sy := u.m.SubtilePos()

	for i, e := range loot.Entries {
		x, y := sx+(i%3)-1, sy+(i/3)-1

		if e.Item == nil { // a gold pile, not an item named "gld"
			names = append(names, fmt.Sprintf("gold %d", e.Gold))

			ent, err := d.engine.NewGoldPile(e.Gold, d.asset.TranslateString("gld"), x, y)
			if err != nil {
				d.Debugf("no ground graphic for a gold pile: %v", err)

				continue
			}

			d.Counters.Drops++
			d.engine.AddEntity(ent)

			continue
		}

		it := e.Item
		names = append(names, fmt.Sprintf("%s(%s)", it.CommonCode, colorToken.ReplaceAllString(it.Label(), "")))
		d.Counters.Drops++

		ent, err := d.engine.NewDroppedItem(x, y, it)
		if err != nil {
			d.Debugf("no ground graphic for %s: %v", it.CommonCode, err)

			continue
		}

		d.engine.AddEntity(ent)
	}

	d.emit("drop", "MONSTER drop name=%s tc=%q ilvl=%d items=%d [%s]", u.m.Label(), tc, level, len(loot.Entries),
		strings.Join(names, ", "))
}
