package d2herostats

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// Attributes are a hero's allocated attributes (as stored in a save, without
// item bonuses) and level.
type Attributes struct {
	Level                  int
	Str, Dex, Vit, Ene     int
	ItemDefense, ItemToHit int // flat armor class / attack rating from gear, optional
	DefensePct             int // item_armor_percent, optional
}

// Derived is everything a hero's class, level and attributes determine.
type Derived struct {
	MaxLife, MaxMana, MaxStamina int
	AttackRating                 int
	Defense                      int
}

// Derive computes the maxima without items plus attack rating and defense.
// The maxima are Class.BaseMax (verified on the level 94 save: mana 221 and
// stamina 525 exact, life 849 plus the 20 life of the Act 3 quest reward =
// stored 869). Attack rating is (dex-7)*5 + ToHitFactor (+ gear), the notes'
// GetPlayerAttackRating; defense is dex/4 plus gear, via
// d2combat.Defense (verified in the binary).
func Derive(c d2statlist.Class, a Attributes) Derived {
	life, mana, stam := c.BaseMax(a.Level, a.Vit, a.Ene)

	return Derived{
		MaxLife:      life,
		MaxMana:      mana,
		MaxStamina:   stam,
		AttackRating: d2combat.PlayerAttackRating(a.ItemToHit, a.Dex, c.ToHitFactor),
		Defense:      d2combat.Defense(a.ItemDefense, a.Dex, a.DefensePct),
	}
}

// LevelUpGain is the maximum life, mana and stamina one level-up grants in
// quarter points (charstats values are in fourths: 8 means +2 per level).
func LevelUpGain(c d2statlist.Class) (life4, mana4, stamina4 int) {
	return c.LifePerLevel, c.ManaPerLevel, c.StaminaPerLevel
}

// PointGain is what one attribute point grants in quarter points: vitality
// gives life and stamina, energy gives mana. Strength and dexterity grant
// none by themselves.
func PointGain(c d2statlist.Class) (lifePerVit4, staminaPerVit4, manaPerEne4 int) {
	return c.LifePerVit, c.StaminaPerVit, c.ManaPerEne
}
