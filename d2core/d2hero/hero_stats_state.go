package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// HeroStatsState is a serializable state of hero stats.
//
// MaxHealth, MaxMana and MaxStamina are the TOTALS with equipment: what the
// globes, the character panel and combat use. A .d2s stores the maxima
// without items, kept in BaseMax* (see RecalcStats); the current values
// Health, Mana and Stamina include the item bonuses, like in the game.
type HeroStatsState struct {
	Level      int `json:"level"`
	Experience int `json:"experience"`

	Strength  int `json:"strength"`
	Energy    int `json:"energy"`
	Dexterity int `json:"dexterity"`
	Vitality  int `json:"vitality"`
	// there are stats and skills points remaining to add.
	StatsPoints int `json:"statsPoints"`
	SkillPoints int `json:"skillPoints"`

	Health     int     `json:"health"`
	MaxHealth  int     `json:"maxHealth"`
	Mana       int     `json:"mana"`
	MaxMana    int     `json:"maxMana"`
	Stamina    float64 `json:"-"` // only MaxStamina is saved, Stamina gets reset on entering world
	MaxStamina int     `json:"maxStamina"`

	// LifeBonus, ManaBonus and StaminaBonus are permanent additions to the
	// class formula that a save carried (the Potion of Life quest reward is
	// +20 life); StatsBonusInit says they have been derived.
	LifeBonus      int  `json:"lifeBonus,omitempty"`
	ManaBonus      int  `json:"manaBonus,omitempty"`
	StaminaBonus   int  `json:"staminaBonus,omitempty"`
	StatsBonusInit bool `json:"statsBonusInit,omitempty"`

	// values which are not saved/loaded(computed)
	NextLevelExp int `json:"-"`

	// BaseMax* are the maxima without items (what a .d2s stores).
	BaseMaxHealth  int `json:"baseMaxHealth,omitempty"`
	BaseMaxMana    int `json:"baseMaxMana,omitempty"`
	BaseMaxStamina int `json:"baseMaxStamina,omitempty"`

	// Totals is the result of the last RecalcStats: defense, attack rating,
	// damage, resists, block and so on. Nil until the first recalculation. It
	// travels to the client with the hero (its stat list does not).
	Totals *d2statlist.Totals `json:"totals,omitempty"`
	// Recalc recomputes Totals and the maxima after strength/dexterity/
	// vitality/energy or equipment changed. Set by the hero state factory.
	Recalc func() `json:"-"`
}

// CreateHeroStatsState generates a running state from a hero stats.
func (f *HeroStateFactory) CreateHeroStatsState(heroClass d2enum.Hero, classStats *d2records.CharStatRecord) *HeroStatsState {
	result := HeroStatsState{
		Level:        1,
		Experience:   0,
		NextLevelExp: f.asset.Records.GetExperienceBreakpoint(heroClass, 1),
		Strength:     classStats.InitStr,
		Dexterity:    classStats.InitDex,
		Vitality:     classStats.InitVit,
		Energy:       classStats.InitEne,
		StatsPoints:  0,
		SkillPoints:  0,
	}

	life, mana, stam := statClass(classStats).BaseMax(1, classStats.InitVit, classStats.InitEne)
	result.MaxHealth, result.MaxMana, result.MaxStamina = life, mana, stam
	result.BaseMaxHealth, result.BaseMaxMana, result.BaseMaxStamina = life, mana, stam

	result.Mana = result.MaxMana
	result.Health = result.MaxHealth
	result.Stamina = float64(result.MaxStamina)

	return &result
}

// statClass converts a charstats record to the stat list's class.
func statClass(c *d2records.CharStatRecord) d2statlist.Class {
	return d2statlist.Class{
		InitStr: c.InitStr, InitDex: c.InitDex, InitVit: c.InitVit, InitEne: c.InitEne,
		InitStamina: c.InitStamina, HpAdd: c.HpAdd,
		LifePerVit: c.LifePerVit, ManaPerEne: c.ManaPerEne, StaminaPerVit: c.StaminaPerVit,
		LifePerLevel: c.LifePerLevel, ManaPerLevel: c.ManaPerLevel, StaminaPerLevel: c.StaminaPerLevel,
		ToHitFactor: c.ToHitFactor, BlockFactor: c.BlockFactor,
	}
}
