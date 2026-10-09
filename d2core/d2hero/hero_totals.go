package d2hero

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// loadStatBases reads armor.txt and weapons.txt for the stat list (cached).
func (f *HeroStateFactory) loadStatBases() d2statlist.Bases {
	if f.statBases != nil {
		return f.statBases
	}

	armor, err := f.asset.LoadFile(armorTxt)
	if err == nil {
		var weapons []byte
		if weapons, err = f.asset.LoadFile(weaponsTxt); err == nil {
			f.statBases, err = d2statlist.ParseBases(armor, weapons)
		}
	}

	if err != nil {
		fmt.Printf("stats: base item tables unavailable, equipment bonuses limited: %v\n", err)

		f.statBases = d2statlist.Bases{}
	}

	return f.statBases
}

// equippedStatItems returns the equipment of the hero as stat list items:
// the exact items of the imported save when there is one, the base items of
// the equipment otherwise.
func (f *HeroStateFactory) equippedStatItems(state *HeroState) []d2statlist.Item {
	// the equipment of a hero who died is on the corpse: it counts for nothing
	// until the corpse is recovered
	if state.Death != nil && state.Death.Corpse != nil {
		return []d2statlist.Item{}
	}

	if state.statEquipped != nil {
		return state.statEquipped
	}

	bases := f.loadStatBases()
	out := []d2statlist.Item{}

	if len(state.D2SBase) > 0 {
		if tables, err := f.loadD2SItemTables(); err == nil {
			if c, perr := d2s.Parse(state.D2SBase, tables); perr == nil {
				for _, it := range StatItemsFromD2S(c.Items, bases) {
					if !it.Charm {
						out = append(out, it)
					}
				}

				state.statEquipped = out

				return out
			}
		}
	}

	eq := &state.Equipment

	add := func(slot int, code string) {
		if code == "" {
			return
		}

		b := bases[code]
		it := d2statlist.Item{Code: code, Slot: slot, Weapon: b.Weapon, BaseBlock: b.BaseBlock}
		it.Defense = (b.MinAC + b.MaxAC) / 2 // UNVERIFIED: generated items roll minac..maxac, the middle stands in

		out = append(out, it)
	}

	if eq.Head != nil {
		add(d2statlist.SlotHead, eq.Head.ItemCode)
	}

	if eq.Torso != nil {
		add(d2statlist.SlotTorso, eq.Torso.ItemCode)
	}

	if eq.Legs != nil {
		add(d2statlist.SlotFeet, eq.Legs.ItemCode)
	}

	if eq.RightArm != nil {
		add(d2statlist.SlotGloves, eq.RightArm.ItemCode)
	}

	if eq.Shield != nil {
		add(d2statlist.SlotLeftHand, eq.Shield.ItemCode)
	}

	if eq.RightHand != nil {
		add(d2statlist.SlotRightHand, eq.RightHand.GetItemCode())
	}

	if state.statEquipped == nil {
		state.statEquipped = out
	}

	return out
}

// charmStatItems returns the charms in the inventory page of the containers.
func (f *HeroStateFactory) charmStatItems(state *HeroState) []d2statlist.Item {
	if state.Containers == nil {
		return nil
	}

	bases := f.loadStatBases()

	var out []d2s.Item

	for i := range state.Containers.Items {
		s := &state.Containers.Items[i]
		if s.D2S != nil && s.Page == PageInventory && isCharm(s.D2S.Code) {
			it := *s.D2S
			it.Location, it.Page = d2s.LocationStored, PageInventory
			out = append(out, it)
		}
	}

	return StatItemsFromD2S(out, bases)
}

// RecalcStats recomputes the hero's maxima and derived values from the class
// formula, the attributes and the active equipment (equipped items and
// charms in the inventory) and stores them in state.Stats.
//
// The .d2s stores life, mana and stamina WITHOUT items; the current values
// it holds include them. Verified on the real level 94 Sorceress: stored
// maxima 869/221/525, current 1241/464/801; the totals computed here are
// 1241/477/801, i.e. life and stamina match the (full) current values
// exactly and the mana (464) fits below its total (it was not full).
func (f *HeroStateFactory) RecalcStats(state *HeroState) {
	st := state.Stats
	rec := f.asset.Records.Character.Stats[state.HeroType]

	if st == nil || rec == nil {
		return
	}

	class := statClass(rec)

	for c, h := range d2sClassToHero {
		if h == state.HeroType {
			class.ID = int(c) // .d2s class ids are charstats order: 0 Amazon .. 6 Assassin
		}
	}

	life, mana, stam := class.BaseMax(st.Level, st.Vitality, st.Energy)

	if !st.StatsBonusInit {
		// a hero imported from a .d2s: its stored maxima are the class formula
		// plus permanent additions (quest rewards); native heroes have none
		if len(state.D2SBase) > 0 {
			st.LifeBonus, st.ManaBonus, st.StaminaBonus = st.MaxHealth-life, st.MaxMana-mana, st.MaxStamina-stam
		}

		st.StatsBonusInit = true
	}

	life, mana, stam = life+st.LifeBonus, mana+st.ManaBonus, stam+st.StaminaBonus
	st.BaseMaxHealth, st.BaseMaxMana, st.BaseMaxStamina = life, mana, stam

	items := append(append([]d2statlist.Item{}, f.equippedStatItems(state)...), f.charmStatItems(state)...)

	tot := d2statlist.Compute(d2statlist.Hero{
		Class: class, Level: st.Level, Str: st.Strength, Dex: st.Dexterity, Vit: st.Vitality, Ene: st.Energy,
		BaseLife: life, BaseMana: mana, BaseStam: stam, Difficulty: int(state.Difficulty),
	}, items, nil)

	st.Totals = &tot
	st.MaxHealth, st.MaxMana, st.MaxStamina = tot.MaxLife, tot.MaxMana, tot.MaxStamina

	if st.Health > st.MaxHealth {
		st.Health = st.MaxHealth
	}

	if st.Mana > st.MaxMana {
		st.Mana = st.MaxMana
	}

	if st.Stamina > float64(st.MaxStamina) {
		st.Stamina = float64(st.MaxStamina)
	}

	st.Recalc = func() { f.RecalcStats(state) }
}

// storedMax is the maximum to write to a save: the item-free base when the
// hero has been recalculated, the plain value otherwise.
func storedMax(base, total int) int {
	if base > 0 {
		return base
	}

	return total
}

// StatsSummary is the `PANEL character` log line for the autotests.
func StatsSummary(st *HeroStatsState) string {
	t := st.Totals
	if t == nil {
		return fmt.Sprintf("level=%d life=%d/%d mana=%d/%d stamina=%d/%d (no totals)", st.Level,
			st.Health, st.MaxHealth, st.Mana, st.MaxMana, int(st.Stamina), st.MaxStamina)
	}

	return fmt.Sprintf("level=%d str=%d dex=%d vit=%d ene=%d def=%d ar=%d dmg=%d-%d block=%d "+
		"res=fire:%d,cold:%d,light:%d,poison:%d life=%d/%d mana=%d/%d stamina=%d/%d "+
		"mf=%d gf=%d ias=%d fcr=%d fhr=%d fbr=%d frw=%d",
		st.Level, t.Str, t.Dex, t.Vit, t.Ene, t.Defense, t.AttackRating, t.DamageMin, t.DamageMax, t.BlockPct,
		t.ResistShown[0], t.ResistShown[1], t.ResistShown[2], t.ResistShown[3],
		st.Health, st.MaxHealth, st.Mana, st.MaxMana, int(st.Stamina), st.MaxStamina,
		t.MagicFind, t.GoldFind, t.FasterAttack, t.FasterCast, t.FasterHit, t.FasterBlock, t.FasterRun)
}
