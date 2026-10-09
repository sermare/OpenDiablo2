package d2player

import (
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// heroInputs are the hero-dependent values the damage, summon-life and curse
// rows of a skill tooltip need.
type heroInputs struct {
	// Weapon is the hero's physical damage range per hit: Totals.DamageMin/Max,
	// the engine's own main-hand computation (d2statlist weaponDamage: base or
	// two-hand damage, ethereal, enhanced damage %, item +damage, strength and
	// dexterity bonus; bare hands 1-2). Kind 9 takes SrcDam/128 of it.
	// LIMITS: the engine only models the right-hand slot, so a thrown or ranged
	// weapon uses its melee-style range, and there is no class-specific case.
	Weapon [2]int
	// Templates and Diff give the summoned monster's life (kind 13).
	Templates *d2summon.Templates
	Diff      d2summon.Difficulty
	// CurseDiv is AiCurseDivisor of the current difficulty (kind 31). UNVERIFIED:
	// that the +0x1c field of the 0x58-byte record at 0x610fd0 is that column is
	// inferred from the record order, not read from the binary's loader.
	CurseDiv int
}

// heroInputsFrom reads the inputs from the hero's stats. A nil hero, or one
// whose stats were never computed, gives no weapon part and difficulty Normal.
// curseDiv maps a difficulty (0..2) to AiCurseDivisor, nil for none.
func heroInputsFrom(st *d2hero.HeroStatsState, tpl *d2summon.Templates, curseDiv func(diff int) int) heroInputs {
	in := heroInputs{Templates: tpl}

	if st == nil {
		return in
	}

	if t := st.Totals; t != nil {
		in.Weapon = [2]int{t.DamageMin, t.DamageMax}
	}

	in.Diff = d2summon.Difficulty(st.Difficulty)
	if st.Difficulty < 0 || st.Difficulty > 2 {
		in.Diff = d2summon.Normal
	}

	if curseDiv != nil {
		in.CurseDiv = curseDiv(int(in.Diff))
	}

	return in
}

var (
	monstatsMu    sync.Mutex
	monstatsCache = map[*d2asset.AssetManager]*d2summon.Templates{}
)

// monstatsTemplates loads (once per asset manager) the monstats templates.
func monstatsTemplates(asset *d2asset.AssetManager) *d2summon.Templates {
	monstatsMu.Lock()
	defer monstatsMu.Unlock()

	if tpl, ok := monstatsCache[asset]; ok {
		return tpl
	}

	var tpl *d2summon.Templates

	if data, err := asset.LoadFile("/data/global/excel/monstats.txt"); err == nil {
		tpl, _ = d2summon.LoadTemplates(data)
	}

	monstatsCache[asset] = tpl

	return tpl
}

// curseDivisors reads AiCurseDivisor from DifficultyLevels.txt.
func curseDivisors(asset *d2asset.AssetManager) func(int) int {
	if asset == nil || asset.Records == nil {
		return nil
	}

	return func(diff int) int {
		if r := asset.Records.DifficultyLevels[d2enum.DifficultyType(diff)]; r != nil {
			return r.AiCurseDivisor
		}

		return 0
	}
}
