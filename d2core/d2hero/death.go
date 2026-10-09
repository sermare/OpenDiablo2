package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// What happens when a hero dies, as far as the 1.14b binary shows (the player
// mode handler 0x57dcc0 and the corpse builder 0x57d6f0 in Game.exe, the
// experience penalty 0x533410). Each rule says how well it is known.
//
//	VERIFIED (decompiled):
//	- the client flag 0x08 ("died") is set on every death, softcore or
//	  hardcore, and the character is saved at once (single player always; on
//	  the realm only hardcore characters are saved here);
//	- a corpse is created where the hero fell (unit mode 0x11 = dead); the
//	  cursor item and the equipped items (slots 0..12) move onto the corpse,
//	  the inventory and stash stay with the hero;
//	- ALL carried gold is taken away (stat 14 is set to 0) and dropped as gold
//	  piles (at most 2,000,000,000 per pile, at most 32 piles) next to the corpse;
//	- the experience penalty (levels above 1 only) is
//	  DeathExpPenalty percent of the experience span of the current level, never
//	  taking the hero below the start of the level; DeathExpPenalty comes from
//	  DifficultyLevels.txt (this install: Normal 0, Nightmare 5, Hell 10);
//	- the save keeps one corpse: the one whose items are worth most
//	  ('JM' + count 0/1, then 12 header bytes the loader skips, then the items).
//	UNVERIFIED:
//	- the corpse also receives 75% of a stored value (client+0x508) as its
//	  experience stat; its meaning (recoverable experience?) is not known and
//	  is not modelled;
//	- the dead flag is cleared again when a softcore hero respawns (hardcore:
//	  the character stays dead and the loader refuses it);
//	- the life a respawned hero starts with (RespawnLifeFraction);
//	- more than 15 corpses make the game drop the items on the ground instead
//	  (the check is visible, the count is not modelled).

// RespawnLifeFraction is the share of the maximum life a respawned hero has.
// UNVERIFIED: the binary was not searched for the respawn code.
const RespawnLifeFraction = 0.5

// deathExpPenaltyPercent is DeathExpPenalty of DifficultyLevels.txt for normal,
// nightmare and hell (read from this install's patch_d2.mpq).
var deathExpPenaltyPercent = [3]int{0, 5, 10}

// DeathExpPenaltyPercent returns the penalty percentage of a difficulty.
func DeathExpPenaltyPercent(difficulty int) int {
	if difficulty < 0 || difficulty >= len(deathExpPenaltyPercent) {
		return 0
	}

	return deathExpPenaltyPercent[difficulty]
}

// Corpse is what a hero left behind. X and Y are subtile coordinates.
type Corpse struct {
	X         int                            `json:"x"`
	Y         int                            `json:"y"`
	Equipment d2inventory.CharacterEquipment `json:"equipment"`
	// ExpLost and GoldDropped record the penalties of that death.
	ExpLost     int `json:"expLost"`
	GoldDropped int `json:"goldDropped"`
}

// DeathState is the persistent record of a hero's deaths. It rides along
// with the hero state, the save packet and the .d2s export.
type DeathState struct {
	// Deaths counts all deaths of this hero.
	Deaths int `json:"deaths"`
	// Died is the "died" status of the character (d2s status 0x08). For a
	// hardcore hero it is permanent; for a softcore hero it is cleared at respawn.
	Died bool `json:"died"`
	// Corpse is the pending corpse, nil when there is none (or it was recovered).
	Corpse *Corpse `json:"corpse,omitempty"`
}

// DeathInput describes the hero at the moment of death.
type DeathInput struct {
	Hardcore   bool
	Difficulty int // 0 normal, 1 nightmare, 2 hell
	Level      int
	Experience int
	Gold       int
	// ExpStart is the experience at which the current level begins and ExpNext
	// the experience at which the next level begins.
	ExpStart, ExpNext int
	X, Y              int // subtile position of the death
	Equipment         d2inventory.CharacterEquipment
}

// DeathOutcome is the result of a death.
type DeathOutcome struct {
	NewExperience int
	ExpLost       int
	GoldDropped   int // all carried gold
	// CharacterDead is true when the character cannot be played again
	// (hardcore).
	CharacterDead bool
	Corpse        *Corpse
}

// ExpPenalty returns the experience lost to a death: percent of the span of
// the current level, limited so the hero stays in the level (VERIFIED, see
// the rules above). Level 1 never loses experience.
func ExpPenalty(percent, level, exp, start, next int) (lost, newExp int) {
	if level <= 1 || percent <= 0 || next <= start {
		return 0, exp
	}

	lost = percent * (next - start) / 100
	if lost == 0 {
		return 0, exp
	}

	newExp = exp - lost
	if newExp <= start {
		newExp = start + 1
	}

	if newExp > exp { // already at the start of the level: nothing to lose
		return 0, exp
	}

	return exp - newExp, newExp
}

// Die applies a death to the persistent state and returns what happened.
func (s *DeathState) Die(in DeathInput) DeathOutcome {
	lost, newExp := ExpPenalty(DeathExpPenaltyPercent(in.Difficulty), in.Level, in.Experience, in.ExpStart, in.ExpNext)

	out := DeathOutcome{
		NewExperience: newExp,
		ExpLost:       lost,
		GoldDropped:   in.Gold,
		CharacterDead: in.Hardcore,
	}

	s.Deaths++
	s.Died = true
	s.Corpse = &Corpse{X: in.X, Y: in.Y, Equipment: in.Equipment, ExpLost: lost, GoldDropped: in.Gold}
	out.Corpse = s.Corpse

	return out
}

// Respawn ends a softcore death: the hero is alive again (the died flag is
// cleared; UNVERIFIED) and the corpse stays where it is. It returns false for
// a hardcore hero, who cannot respawn.
func (s *DeathState) Respawn(hardcore bool) bool {
	if hardcore {
		return false
	}

	s.Died = false

	return true
}

// Recover hands the corpse's equipment back and removes the corpse. It
// returns nil when there is no corpse.
func (s *DeathState) Recover() *d2inventory.CharacterEquipment {
	if s == nil || s.Corpse == nil {
		return nil
	}

	eq := s.Corpse.Equipment
	s.Corpse = nil

	return &eq
}

// RespawnLife returns the life points of a respawned hero (at least 1).
func RespawnLife(maxLife int) int {
	if n := int(float64(maxLife) * RespawnLifeFraction); n > 1 {
		return n
	}

	return 1
}

// IsDeadHardcore reports whether the hero is a hardcore character that died:
// the game refuses to load it.
func (h *HeroState) IsDeadHardcore() bool {
	return h != nil && h.Hardcore && h.Death != nil && h.Death.Died
}

// EquipmentChanged drops the cached equipment stat items; call it after the
// equipment moved to or came back from a corpse, before RecalcStats.
func (h *HeroState) EquipmentChanged() { h.statEquipped = nil }
