package d2monsters

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

var colorToken = regexp.MustCompile(`\[[a-z]+\]`)

// handleEvents consumes the animation events a monster produced since the
// last frame.
func (d *Director) handleEvents(u *unit) {
	for _, ev := range u.m.TakeEvents() {
		switch ev.Kind {
		case d2mapentity.MonsterEventHitFrame:
			d.monsterStrike(u, ev.Mode)
		case d2mapentity.MonsterEventModeDone:
			u.mv = nil

			if u.m.Alive() && u.m.Mode() == d2monster.ModeNeutral {
				u.b.WakeNow(d.frame)
			}
		case d2mapentity.MonsterEventDied:
		}
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
// frame: to-hit with d2combat.RollToHit using the monster's own seed (the
// attacker's, as in the binary), then a damage roll between the scaled min
// and max.
func (d *Director) monsterStrike(u *unit, mode d2monster.Mode) {
	atk, ok := attackFor(&u.m.Vitals, mode)
	if !ok {
		return
	}

	p := d.playerFor(u.attackTarget)
	if p == nil || !d.targetable(p) {
		return
	}

	px, py := playerSubtile(p)
	sx, sy := u.m.SubtilePos()
	dist := d2monster.EdgeDistance(sx-px, sy-py, u.b.Size)

	reach := meleeInRange
	if u.m.Stat.IsRanged {
		reach = rangedInRange
	}

	d.Counters.Attacks++

	if dist > reach+heroReach/2 {
		d.emit("attack", "MONSTER attack name=%s id=%d mode=%s hit=false reason=target_moved_away dist=%d",
			u.m.Label(), u.b.ID, mode, dist)

		return
	}

	defense := d2combat.Defense(0, p.Stats.Dexterity, 0)
	in := d2combat.ToHitInput{
		AttackRating:  d2combat.MonsterAttackRating(atk.ToHit, 0, 0),
		Defense:       defense,
		AttackerLevel: u.m.Vitals.Level,
		DefenderLevel: p.Stats.Level,
	}

	hit, chance, roll := d2combat.RollToHit(u.b.Seed, in)
	dmg := 0

	if hit {
		dmg = atk.Min + int(u.b.Seed.Roll(int32(atk.Max-atk.Min+1)))
		if dmg < 1 {
			dmg = 1
		}

		d.Counters.AttackHits++
		p.Stats.Health -= dmg

		if p.Stats.Health < 0 {
			p.Stats.Health = 0
		}
	}

	d.emit("attack", "MONSTER attack name=%s id=%d mode=%s hit=%v chance=%d roll=%d dmg=%d hero_hp=%d/%d",
		u.m.Label(), u.b.ID, mode, hit, chance, roll, dmg, p.Stats.Health, p.Stats.MaxHealth)

	if hit && p.Stats.Health == 0 {
		d.Counters.HeroDeaths++
		d.emit("herodeath", "HERO died name=%s killer=%s", p.Name(), u.m.Label())
	}
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
	d.Counters.HeroHits++

	d.emit("herohit", "HERO swing target=%s hit=true chance=%d roll=%d dmg=%d", m.Label(), chance, roll, dmg)
	d.damage(u, p, dmg)

	return true
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

func (d *Director) damage(u *unit, src *d2mapentity.Player, dmg int) {
	if !u.m.Alive() {
		return
	}

	u.m.Vitals.HP -= dmg

	if u.m.Vitals.HP > 0 {
		d.emit("hit", "MONSTER hit name=%s id=%d dmg=%d hp=%d/%d", u.m.Label(), u.b.ID, dmg, u.m.Vitals.HP, u.m.Vitals.MaxHP)

		// hit recovery: the monster stops what it was doing, and thinks again
		// when the animation ends. Aggro is not otherwise changed.
		u.m.StopMoving()
		u.mv = nil

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

func (d *Director) kill(u *unit, src *d2mapentity.Player) {
	u.m.Die()
	d.Counters.Deaths++

	by := "unknown"
	xp := u.m.Vitals.Experience

	if src != nil {
		by = src.Name()
		src.Stats.Experience += xp
	}

	d.emit("death", "MONSTER death name=%s id=%d by=%s xp=%d", u.m.Label(), u.b.ID, by, xp)
	d.dropLoot(u)
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

	items, err := d.engine.DropItems(tc, diablo2item.DropOptions{
		Seed: u.b.Seed.Step(), ILvl: level, UpgradeLevel: level, Players: 1,
	})
	if err != nil {
		d.emit("drop", "MONSTER drop name=%s tc=%q error=%v", u.m.Label(), tc, err)

		return
	}

	names := make([]string, 0, len(items))
	sx, sy := u.m.SubtilePos()

	for i, it := range items {
		names = append(names, fmt.Sprintf("%s(%s)", it.CommonCode, colorToken.ReplaceAllString(it.Label(), "")))
		d.Counters.Drops++

		ent, err := d.engine.NewDroppedItem(sx+(i%3)-1, sy+(i/3)-1, it)
		if err != nil {
			d.Debugf("no ground graphic for %s: %v", it.CommonCode, err)

			continue
		}

		d.engine.AddEntity(ent)
	}

	d.emit("drop", "MONSTER drop name=%s tc=%q ilvl=%d items=%d [%s]", u.m.Label(), tc, level, len(items),
		strings.Join(names, ", "))
}
