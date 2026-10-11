package d2monster

// Faithful ports, batch B1 (monai names from N to Z): Vampire, SuccubusWitch,
// Nihlathak, ShadowWarrior, ShadowMaster (+ShadowMasterNoInit) and WillOWisp.
// They replace the generic stand-ins of ai_rest.go. All read the aip columns
// the exe reads and draw from the unit seed in the exe's order (an `&&` that
// short-circuits skips its roll). Addresses are Game.exe 1.14b VAs.
//
// Conventions: Scratch[0..2] are AiGeneral +0x14/+0x18/+0x1c, aipN = b.AIP(N),
// MONAI_IsDataField54Idle (0x5dbff0) is b.Aggressive, STATS_GetLifePercent of
// the monster is b.HPPercent. Bounded rolls whose bound the decompiler hid are
// percent rolls (bound 100, confirmed by the readable disassembly prefix).
// Whatever depends on engine data the Go side does not have (the target's
// stats, the shadow's copy of the hero's skills) goes through the optional
// interfaces below; without them the code takes the documented fallback.

func init() {
	for _, n := range []struct {
		name string
		f    Think
	}{
		{"Vampire", thinkVampire}, {"SuccubusWitch", thinkSuccubusWitch}, {"Nihlathak", thinkNihlathak},
		{"ShadowWarrior", thinkShadowWarrior}, {"ShadowMaster", thinkShadowMaster},
		{"ShadowMasterNoInit", thinkShadowMaster}, {"WillOWisp", thinkWillOWisp},
	} {
		mode, ok := AITargetMode(n.name)
		if !ok {
			mode = TargetStandard
		}

		register(n.name, mode, n.f)
	}
}

// FB1Unit answers questions about a unit that is not the monster itself:
// STATS_GetLifePercent / STATS_GetMaxHitPoints / STATS_GetMaxMana and
// STATS_FindStatListByFlags. Fallbacks without it: full life, 1 max life,
// 0 max mana, no stat list with the flag.
type FB1Unit interface {
	LifePercent(t Target) int
	MaxHP(t Target) int
	MaxMana(t Target) int
	HasStatListFlag(t Target, flag uint32) bool
}

// FB1Scanner is the MONAI_ForEachUnitByModeCallback scan at 0x5e1570 used by
// Overseer and Nihlathak (UNVERIFIED: the callback is not a defined function
// in the database). The argument block holds a squared radius and a 0x32 (50)
// threshold; the callback fills a target. It is read here as "the nearest
// ally within radius2 whose life is at most pct percent".
type FB1Scanner interface {
	ScanWounded(b *Brain, radius2, pct int) (Target, bool)
}

// FB1Revive is SKILL_FindReviveCorpseNearUnit: a corpse the monster's skill in
// the slot can revive, or false.
type FB1Revive interface {
	ReviveCorpse(b *Brain, slot int) (Target, bool)
}

// FB1Spawn is MONSTER_AreSpawnerCellsFree at the monster's own position.
// Fallback: free.
type FB1Spawn interface {
	SpawnCellsFree(b *Brain) bool
}

// FB1OffsetCaster is MONAI_QueueRandomOffsetSkillCast (UNVERIFIED offset
// scheme): cast a slot at a random point near the monster. Fallback: cast at
// the monster's own tile.
type FB1OffsetCaster interface {
	CastAtRandomOffset(b *Brain, slot int) bool
}

// FB1Pet is the owner relation of the shadow minions (MONAI_GetLeaderUnit,
// UNIT_GetDistanceToUnit and MONAI_SelectPetActionByRange 0x5e3510).
type FB1Pet interface {
	// LeaderUnit is the owning player.
	LeaderUnit(b *Brain) (Target, bool)
	// PetAction is MONAI_SelectPetActionByRange(target, leader, inRange, 0, 6):
	// true when it queued a follow/regroup action.
	PetAction(b *Brain, tgt *Target, leader Target, inRange bool) bool
}

// FB1ShadowBook is the shadow's copy of the hero's skills. The exe scores every
// skill record (skills.txt kind, states, missile ranges) per tick; the Go
// side has no skills.txt records, so the whole scoring is the host's. All RNG
// must be drawn from b.
type FB1ShadowBook interface {
	// HasSkills reports whether the unit owns any skill (unit+0xa8 != 0).
	HasSkills(b *Brain) bool
	// CastQueued casts the skill id remembered in Scratch[1] (CastChosenSkill).
	CastQueued(b *Brain, skillID int, tgt Target, inRange bool) bool
	// Upkeep is the buff/aura maintenance loop over the unit's skills.
	Upkeep(b *Brain, tgt *Target, inRange bool) bool
	// Choose scores the skills against tgt and casts the best one; it returns
	// the cast skill id and whether it queued an action. calc is the
	// follow-up count for kind 0x13 skills (stored in Scratch[0], with the
	// skill id in Scratch[1]).
	Choose(b *Brain, tgt Target, leader *Target, inRange bool) (skillID int, queued bool)
	// Copy is the ShadowWarrior's copy-the-hero's-attack (right or left skill):
	// returns the skills.txt calc value used for the cooldown and true when it
	// queued an action.
	Copy(b *Brain, leader Target, tgt Target, useLeft, basic, inRange bool) (calc int, ok bool)
}

// tryCast is QueueSkillCast without the fallback sleep: it reports whether the
// request was accepted (the exe continues with the next option when not).
func (c *Ctx) tryCast(slot int, t Target) bool {
	if !c.W.Cast(c.B, slot, t) {
		return false
	}

	c.busy()

	return true
}

func fb1Life(w World, t Target) int {
	if u, ok := w.(FB1Unit); ok {
		return u.LifePercent(t)
	}

	return 100
}

// ---------------------------------------------------------------- Vampire

// thinkVampire is MONAI_Think_Vampire 0x5f3b20. aip1 melee %, aip2 stand
// %, aip3 engage range, aip4 skill %, aip5 skill mask (1: slot1/slot4 at
// range, 2: slot2, 4: slot3). Scratch[0] is the mood (1 fight, 2 wounded:
// keeps away until life > 74%), Scratch[1] the nearest distance seen (while
// aggressive), Scratch[2] the cooldown after slot2/slot3, counting down.
// Flow re-read from the decompilation with its gotos resolved.
func thinkVampire(c *Ctx) {
	b := c.B
	if c.Target == nil {
		c.Sleep(25)

		return
	}

	tgt := *c.Target
	mask := b.AIP(5)
	bit1, bit2, bit4 := mask&1 != 0, mask&2 != 0, mask&4 != 0
	dist := c.Dist

	if b.Scratch[2] > 0 {
		b.Scratch[2]--
	}

	t2, d2, ok2 := c.W.AttackTarget(b)

	// skill1/skill4 at t2 or tgt: 50/50 by a roll
	pick := func(at Target) {
		if b.Roll(100) < 50 {
			c.Cast(0, at)
		} else {
			c.Cast(3, at)
		}
	}
	// the slot1/slot4 option the attack branches share: needs the bit, a
	// second target within 20
	rangedOK := func() bool { return bit1 && ok2 && d2 <= 20 }

	if b.Aggressive {
		if b.Scratch[0] == 0 {
			b.Scratch[0] = 1
		}

		if dist < 30 && b.Scratch[1] < dist {
			b.Scratch[1] = dist
		}

		if c.InRange {
			if b.Roll(100) > 30 || !bit1 {
				c.Attack(ModeAttack1, tgt)
			} else {
				pick(tgt)
			}

			return
		}
	}

	hp := b.HPPercent

	if b.Scratch[0] == 2 {
		vampireWounded(c, tgt, t2, d2, ok2, hp, bit2, bit4, rangedOK, pick)

		return
	}

	if hp < 33 {
		b.Scratch[0] = 2

		if c.WalkAway(tgt, 8) {
			return
		}
	}

	if c.InRange {
		b.Scratch[0] = 1

		if b.Roll(100) < b.AIP(1) {
			if !bit1 || !ok2 || b.Roll(100) > 30 {
				c.Attack(ModeAttack1, tgt)

				return
			}

			if d2 < 21 {
				pick(t2)

				return
			}
		}

		if b.Roll(100) < 33 {
			c.Circle(tgt, 4)

			return
		}

		c.Sleep(10)

		return
	}

	if b.AIP(3) <= dist {
		if b.Scratch[0] != 1 {
			c.Sleep(15)

			return
		}

		c.WalkTo(tgt, 7)

		return
	}

	b.Scratch[0] = 1

	if b.Roll(100) >= b.AIP(2) {
		if dist < 21 {
			if dist < 9 && b.Roll(100) < 50 {
				c.WalkAway(tgt, 8)

				return
			}

			if b.Roll(100) < 50 {
				c.Circle(tgt, 4)

				return
			}

			c.Sleep(10)

			return
		}

		c.WalkTo(tgt, 7)

		return
	}

	// engaged: slot2, then slot3, else the ranged pair, else walk/circle
	if bit2 && b.Scratch[2] < 1 && b.Roll(100) < b.AIP(4) {
		vampireCast(c, 1, tgt)

		return
	}

	if !bit4 || b.Scratch[2] > 0 || b.Roll(100) >= b.AIP(4) {
		if !rangedOK() {
			c.WalkTo(tgt, 7)

			return
		}

		if b.Roll(100) > 74 {
			c.Circle(tgt, 4)

			return
		}

		pick(t2)

		return
	}

	vampireCast(c, 2, tgt)
}

// vampireCast casts slot at tgt and starts the 11-frame cooldown.
func vampireCast(c *Ctx, slot int, tgt Target) {
	c.Cast(slot, tgt)
	c.B.Scratch[2] = 0xb
}

// vampireWounded is the Scratch[0]==2 branch of thinkVampire.
func vampireWounded(c *Ctx, tgt, t2 Target, d2 int, ok2 bool, hp int, bit2, bit4 bool, rangedOK func() bool,
	pick func(Target)) {
	b := c.B
	dist := c.Dist

	if hp > 74 {
		b.Scratch[0] = 1
		c.WalkTo(tgt, 7)

		return
	}

	if dist < 14 || dist <= b.Scratch[1] {
		speed := 0

		if p := b.Profile; p.Walk > 0 {
			speed = p.Run*100/p.Walk - 100
		}

		if speed < 0 {
			speed = 0
		} else if speed > 0x78 {
			speed = 0x78
		}

		c.SetSpeed(speed)

		if c.WalkAway(tgt, 8) {
			return
		}
	}

	if b.AIP(3) <= dist || b.Roll(100) >= b.AIP(2) {
		c.Sleep(15)

		return
	}

	if bit2 && b.Scratch[2] < 1 && b.Roll(100) < b.AIP(4) {
		vampireCast(c, 1, tgt)

		return
	}

	if !bit4 || b.Scratch[2] > 0 || b.Roll(100) >= b.AIP(4) {
		if !rangedOK() {
			c.Circle(tgt, 4)

			return
		}

		pick(t2)

		return
	}

	vampireCast(c, 2, tgt)
}

// ---------------------------------------------------------------- SuccubusWitch

// thinkSuccubusWitch is MONAI_Think_SuccubusWitch 0x5e0fa0. aip1 melee %, aip2
// approach %, aip3 support-cast %, aip4 keep-away radius, aip5 slot5/shout %,
// aip6 stall, aip7 min target life % for slot1, aip8 own life % at/below
// which slot2 is used (and the slot5 gate). Slots: 1 and 2 support, 3 and 4
// player-only, 5 ranged. The target's life/mana come from FB1Unit.
func thinkSuccubusWitch(c *Ctx) {
	b := c.B
	if c.Target == nil {
		c.Sleep(25)

		return
	}

	tgt := *c.Target
	dist := c.Dist
	sk := b.Profile.Skills
	u, hasU := c.W.(FB1Unit)

	hasFlag := false
	if hasU {
		hasFlag = u.HasStatListFlag(tgt, 0x20)
	}

	if !hasFlag && dist < b.AIP(4) && b.Roll(100) < b.AIP(3) {
		switch {
		case sk[0].Used() && fb1Life(c.W, tgt) >= b.AIP(7):
			c.Cast(0, tgt)

			return
		case sk[1].Used() && b.HPPercent <= b.AIP(8):
			c.Cast(1, tgt)

			return
		}

		if sk[2].Used() {
			maxHP, maxMana := 1, 0
			if hasU {
				maxHP, maxMana = u.MaxHP(tgt), u.MaxMana(tgt)
			}

			if maxMana < maxHP || !tgt.IsPlayer {
				c.Cast(2, tgt)

				return
			}
		}

		if sk[3].Used() && tgt.IsPlayer {
			c.Cast(3, tgt)

			return
		}
	}

	if c.InRange {
		if b.Roll(100) >= b.AIP(3) || !c.WalkAway(tgt, b.AIP(4)) {
			if b.Roll(100) < b.AIP(1) {
				c.Attack(ModeAttack1, tgt)

				return
			}

			c.Sleep(b.AIP(6))
		}

		return
	}

	if sk[4].Used() && b.AIP(8) > 0 {
		if b.Roll(100) < b.AIP(5) {
			if t2, d, ok := c.W.AttackTarget(b); ok {
				dist = d

				if b.Roll(100) < b.AIP(8) {
					c.Cast(4, t2)

					return
				}
			}
		}
	}

	if dist < b.AIP(4) && b.Roll(100) < b.AIP(3) && c.WalkAway(tgt, b.AIP(4)) {
		return
	}

	if !sk[4].Used() {
		if t3, _, ok := c.W.AttackTarget(b); ok && b.Roll(100) < b.AIP(5) {
			c.Attack(ModeSkill2, t3)

			return
		}
	}

	if b.Roll(100) < b.AIP(2) {
		c.WalkTo(tgt, 0)

		return
	}

	if b.Roll(100) > 49 || !c.Circle(tgt, 6) {
		c.Sleep(b.AIP(6))
	}
}

// ---------------------------------------------------------------- Nihlathak

// thinkNihlathak is MONAI_Think_Nihlathak 0x5ed6d0. aip1 teleport %, aip2
// slot2 %, aip3 corpse-raise %, aip4 help %, aip5 keep-away distance. Slots
// (monstats Skill1..5): 1 teleport (cast at a random offset), 2 helper cast,
// 3 raise, 4 close-range, 5 summon. The exe finds its skills with
// SKILL_FindUnitSkillByIdFwd (id hidden by the decompiler): UNVERIFIED, read
// as "the slot is used". It also clears state 0xc and refreshes the Act 5
// quest log (host business).
func thinkNihlathak(c *Ctx) {
	b := c.B
	sk := b.Profile.Skills

	c.questHook("nihlathak") // QUEST_A5_BetrayalOfHarrogath_RefreshLogOnNihlathakThink, every think

	if c.W.HasState(b, 0xc) {
		if ss, ok := c.W.(StateSetter); ok {
			ss.SetUnitState(b, 0xc, false)
		}
	}

	if c.Target == nil {
		if sk[0].Used() && b.Aggressive {
			nihOffset(c, 0)

			return
		}

		c.Sleep(25)

		return
	}

	tgt := *c.Target

	if sk[0].Used() && c.InRange && b.Roll(100) < b.AIP(1) {
		nihOffset(c, 0)

		return
	}

	if c.Dist < b.AIP(5) && b.Roll(100) < 40 && c.WalkAway(tgt, 5) {
		return
	}

	if sk[2].Used() && b.Roll(100) < b.AIP(3) {
		if rv, ok := c.W.(FB1Revive); ok {
			if corpse, found := rv.ReviveCorpse(b, 2); found && c.tryCast(2, corpse) {
				return
			}
		}
	}

	if sk[3].Used() && b.Roll(100) < 60 && c.Dist < 14 {
		c.Cast(3, tgt)

		return
	}

	if sk[1].Used() && b.Roll(100) < b.AIP(4) {
		if sc, ok := c.W.(FB1Scanner); ok {
			if ally, found := sc.ScanWounded(b, 625, 50); found {
				c.Cast(1, ally)

				return
			}
		}

		if sk[4].Used() {
			free := true
			if sp, ok := c.W.(FB1Spawn); ok {
				free = sp.SpawnCellsFree(b)
			}

			if free {
				// the exe also stores the level's spawn range at AiGeneral+0x3c
				// (not modelled); the engine's summon skill sizes the group
				c.Cast(4, tgt)

				return
			}
		}

		// VERIFIED (batch 5): with the roll passed and nobody to help, the exe
		// wanders (6) whether or not Skill5 exists or has room, and ends the tick.
		c.Wander(6)

		return
	}

	if sk[3].Used() && c.Dist < 14 {
		c.Cast(3, tgt)

		return
	}

	if b.Roll(100) < 60 {
		c.WalkTo(tgt, 0)
	}

	c.Sleep(5)
}

func nihOffset(c *Ctx, slot int) {
	if oc, ok := c.W.(FB1OffsetCaster); ok {
		if oc.CastAtRandomOffset(c.B, slot) {
			c.busy()
		} else {
			c.Sleep(10)
		}

		return
	}

	c.Cast(slot, Target{X: c.B.X, Y: c.B.Y})
}

// ---------------------------------------------------------------- ShadowWarrior

// thinkShadowWarrior is MONAI_Think_ShadowWarrior 0x5e9ff0. The warrior is the
// Assassin's shadow: it follows its owner and copies the owner's right or left
// skill. aip1 max target distance, aip2 max owner distance, aip3 base %
// to use the plain attack, aip4 the per-tick decay of the fatigue counter
// Scratch[1] (cap 64 * clamp(aip8 of Hell, 1..256)). Scratch[0] is the frame
// until which the cast cooldown lasts (calc/3+18), Scratch[2] the skill level
// the Pre-hook 0x5ea4f0 stores (>=1; the Pre-hook's skill copy is the host's).
// The skill record work (levels, copies, animation type) is FB1ShadowBook.Copy.
// UNVERIFIED: the aip8 cap reads the Hell column of the exe at a fixed
// offset; the port uses the current difficulty's column.
func thinkShadowWarrior(c *Ctx) {
	b := c.B

	pet, ok := c.W.(FB1Pet)
	if !ok {
		c.Sleep(100)

		return
	}

	leader, ok := pet.LeaderUnit(b)
	if !ok {
		c.Sleep(100)

		return
	}

	limit := b.AIP(8)
	if limit < 1 {
		limit = 1
	} else if limit > 256 {
		limit = 256
	}

	b.Scratch[1] += -1 - b.AIP(4)
	if b.Scratch[1] < 0 || limit*64 < b.Scratch[1] {
		b.Scratch[1] = 0
	}

	ownerDist := EdgeDistance(b.X-leader.X, b.Y-leader.Y, b.Size)

	var tgt *Target
	if c.Target != nil {
		tgt = c.Target
	}

	if b.AIP(1) < c.Dist || b.AIP(2) < ownerDist {
		tgt = nil
	}

	if pet.PetAction(b, tgt, leader, c.InRange) {
		return
	}

	if tgt != nil {
		if book, ok := c.W.(FB1ShadowBook); ok {
			useLeft := b.Roll(2) != 0

			level := b.Scratch[2]
			if level < 2 {
				level = 1
			}

			pct := b.AIP(3) - level*2
			if pct < 6 {
				pct = 5
			} else if pct > 99 {
				pct = 100
			}

			basic := false
			if c.InRange && b.Roll(100) < pct {
				basic = true
			}

			if calc, ok := book.Copy(b, leader, *tgt, useLeft, basic, c.InRange); ok {
				b.Scratch[0] = calc/3 + 18 + c.W.Frame()
				c.busy()

				return
			}

			// the copy failed: the exe has no further fallback besides the
			// plain RunToTarget when no animation applies
			if !c.InRange && c.RunTo(*tgt, 0) {
				return
			}
		}
	}

	c.Sleep(25)
}

// ---------------------------------------------------------------- ShadowMaster

// thinkShadowMaster is MONAI_Think_ShadowMaster 0x5ea9f0 (ShadowMasterNoInit
// shares it; their Pre-hooks 0x5ea4f0/0x5ea620 only reset Scratch[0..1] to 0,
// set Scratch[2]=1 and copy the owner's skills - the zero Scratch[2] reads as
// the same level 1 here, the skill copy is the host's). aip1 caster range,
// aip2 owner leash (squared: the exe compares a distance squared with
// aip2^2) and engage range, aip3 base % to cast a chosen skill.
//
// The outer flow is ported: no skills sleeps 100; owner pull-back; the queued
// follow-up cast (Scratch[0] count, Scratch[1] skill id); the leash that drops
// the target; the owner's own target when idle; the cast roll
// (aip3 - 2*level, clamped 5..100). The per-skill scoring tables (skills.txt
// kinds 0..0xd, ~300 lines of the exe) and the upkeep loop need the skill
// records and are the host's (FB1ShadowBook); without it the monster closes
// in and attacks. UNVERIFIED: exact use of the aip1/aip2 columns, which the
// exe reads at fixed per-difficulty offsets.
func thinkShadowMaster(c *Ctx) {
	b := c.B
	book, hasBook := c.W.(FB1ShadowBook)

	if hasBook && !book.HasSkills(b) {
		c.Sleep(100)

		return
	}

	pet, hasPet := c.W.(FB1Pet)

	var leader *Target

	if hasPet {
		if l, ok := pet.LeaderUnit(b); ok {
			leader = &l
		}
	}

	tgt := c.Target

	if leader != nil && sqDist(b.X, b.Y, leader.X, leader.Y) > shadowWord(b, 2, shadowLeash)*shadowWord(b, 2, shadowLeash) &&
		pet.PetAction(b, nil, *leader, c.InRange) {
		return
	}

	if b.Scratch[0] > 0 {
		if tgt == nil {
			b.Scratch[0], b.Scratch[1] = 0, 0
		} else {
			b.Scratch[0]--

			if hasBook && book.CastQueued(b, b.Scratch[1], *tgt, c.InRange) {
				c.busy()

				return
			}
		}
	}

	// out of the aip2 range the target is dropped; with none, the buff/aura
	// upkeep loop runs (it may cast and end the tick)
	if shadowWord(b, 2, shadowIgnoreRange) < c.Dist {
		tgt = nil
	}

	if tgt == nil && hasBook && book.Upkeep(b, nil, c.InRange) {
		c.busy()

		return
	}

	// the owner's own hostile, living target becomes the shadow's target
	var ownerTgt *Target

	if st, ok := c.W.(ShadowTargeting); ok && leader != nil {
		if ot, found := st.OwnerTarget(b, *leader); found {
			ownerTgt = &ot
			tgt = &ot
		}
	}

	if leader != nil && sqDist(b.X, b.Y, leader.X, leader.Y) < 0x91 &&
		pet.PetAction(b, tgt, *leader, c.InRange) {
		return
	}

	if tgt == nil {
		c.Sleep(25)

		return
	}

	level := b.Scratch[2]
	if level < 2 {
		level = 1
	}

	pct := b.AIP(3) - level*2
	if pct < 6 {
		pct = 5
	} else if pct > 99 {
		pct = 100
	}

	// the plain attack (skill id 0 through CastChosenSkill) on a roll
	if c.InRange && b.Roll(100) < pct && c.W.Attack(b, ModeAttack1, *tgt) {
		c.busy()

		return
	}

	if env, ok := c.W.(ShadowEnv); ok && hasBook {
		if shadowTail(c, env, book, leader, tgt, ownerTgt) {
			return
		}
	} else if hasBook {
		if id, queued := book.Choose(b, *tgt, leader, c.InRange); queued {
			b.Scratch[1] = id
			c.busy()

			return
		}
	}

	if c.InRange {
		c.Attack(ModeAttack1, *tgt)

		return
	}

	if !c.WalkTo(*tgt, 0) {
		c.Sleep(15)
	}
}

// ---------------------------------------------------------------- WillOWisp
