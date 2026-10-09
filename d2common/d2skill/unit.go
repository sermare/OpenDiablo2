package d2skill

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"

// Unit is the caster as the pipeline sees it.
type Unit interface {
	ID() string
	IsPlayer() bool
	Level() int
	// SkillLevel is the effective level (tree + item bonuses); BaseSkillLevel
	// the points put into the skill only (blvl in calcs, used by synergies).
	SkillLevel(skillID int) int
	BaseSkillLevel(skillID int) int
	// Stat returns a named stat (ItemStatCost name, e.g. "strength",
	// "passive_fire_mastery"); unknown stats are 0.
	Stat(name string) int
	// Mana is the current mana in 8.8 fixed point.
	Mana() int
	SetMana(v int)
	// Pos is the subtile position.
	Pos() (x, y int)
	InTown() bool
	Roller() d2combat.Roller
	// AttackRating is the total attack rating (stat 0x13 plus dexterity and
	// class terms).
	AttackRating() int
	// WeaponDamage is the equipped weapon's damage range in whole points.
	WeaponDamage() (min, max int)
	// RangedWeaponMissile is the missile name of the equipped bow or crossbow
	// ("" for a melee weapon); ThrownMissile the one of a throwing weapon.
	RangedWeaponMissile() string
	ThrownMissile() string
	// HasAmmo reports whether a ranged attack can be made; ConsumeAmmo uses
	// one (decquant skills).
	HasAmmo() bool
	ConsumeAmmo() bool
	// Cooldown is the frame until which the skill is blocked (state 121);
	// SetCooldown sets it.
	Cooldown(skillID int) int
	SetCooldown(skillID, untilFrame int)
}
