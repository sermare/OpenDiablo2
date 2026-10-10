package d2object

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"

// Overlay puts the active buffs on a hero's Totals. The combat code reads
// HeroStatsState.Totals, and RecalcStats replaces that pointer whenever the
// equipment changes; the overlay therefore remembers the pointer it modified
// and the values before, restores them and re-applies when asked again.
type Overlay struct {
	Buffs Buffs

	target *d2statlist.Totals
	base   d2statlist.Totals
}

// Apply refreshes the overlay on t: the buffs that ended by now are removed
// (and returned), the others are applied to t. Call it every frame; it is
// cheap when nothing changed. t may be nil.
func (o *Overlay) Apply(t *d2statlist.Totals, now float64) (expired []Buff) {
	expired = o.Buffs.Expire(now)

	if t == nil {
		o.target = nil
		return expired
	}

	if o.target == t {
		*t = o.base // undo the previous application
	}

	o.target, o.base = t, *t

	ApplyBuffs(t, &o.Buffs)

	return expired
}

// ApplyBuffs adds the stats of the active buffs to the totals:
//   - skill_armor_percent raises Defense by that percent of the current value;
//   - to hit percent raises AttackRating, damage percent raises DamageMin/Max;
//   - resist stats raise Resist and ResistShown (capped at MaxResist);
//   - all skills raises AllSkills.
//
// Mana recovery percent raises Totals.ManaRecoveryPct (natural regeneration);
// stamina and experience are not in Totals, read them with Buffs.Total
// (StatSkillStaminaPct, StatExperience).
func ApplyBuffs(t *d2statlist.Totals, b *Buffs) {
	if pct := int(b.Total(StatSkillArmorPct)); pct != 0 {
		t.Defense += t.Defense * pct / 100
	}

	if pct := int(b.Total(d2statlist.StatToHitPct)); pct != 0 {
		t.AttackRating += t.AttackRating * pct / 100
	}

	if pct := int(b.Total(d2statlist.StatDamagePct)); pct != 0 {
		t.DamageMin += t.DamageMin * pct / 100
		t.DamageMax += t.DamageMax * pct / 100
	}

	for i, st := range [4]int{d2statlist.StatFireResist, d2statlist.StatColdResist, d2statlist.StatLightResist,
		d2statlist.StatPoisonResist} {
		if d := int(b.Total(st)); d != 0 {
			t.Resist[i] += d

			shown := t.ResistShown[i] + d
			if t.MaxResist[i] > 0 && shown > t.MaxResist[i] {
				shown = t.MaxResist[i]
			}

			t.ResistShown[i] = shown
		}
	}

	t.AllSkills += int(b.Total(d2statlist.StatAllSkills))
	t.ManaRecoveryPct += int(b.Total(d2statlist.StatManaRecovery))
}

// ExperiencePct is the bonus experience percent of the active buffs.
func (o *Overlay) ExperiencePct() int { return int(o.Buffs.Total(StatExperience)) }

// UnlimitedStamina reports whether the stamina shrine runs.
func (o *Overlay) UnlimitedStamina() bool { return o.Buffs.Total(StatSkillStaminaPct) > 0 }

// ManaRecoveryPct is the mana recovery bonus percent of the active buffs.
func (o *Overlay) ManaRecoveryPct() int { return int(o.Buffs.Total(d2statlist.StatManaRecovery)) }
