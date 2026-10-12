package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
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
//	- the carried gold is taken away (stat 14 is set to 0); min(level,20)% of carried+stash is lost for
//	  good and the rest is dropped as gold piles (at most 32) next to the corpse (DeathGoldPenalty, 0x5331e0);
//	- the experience penalty (levels above 1 only) is
//	  DeathExpPenalty percent of the experience span of the current level, never
//	  taking the hero below the start of the level; DeathExpPenalty comes from
//	  DifficultyLevels.txt (this install: Normal 0, Nightmare 5, Hell 10);
//	- the save keeps one corpse: the one whose items are worth most
//	  ('JM' + count 0/1, then 12 header bytes the loader skips, then the items).
//	UNVERIFIED:
//	- the corpse also receives 75% of a stored value (client+0x508) as its
//	  experience stat (Corpse.Experience, CorpseExperience); its meaning
//	  (recoverable experience?) is not known;
//	- the dead flag is cleared again when a softcore hero respawns (hardcore:
//	  the character stays dead and the loader refuses it);
//	  RESOLVED by the second gap pass (w20): the stat is the experience that looting the corpse gives back
//	  (PLAYER_ServerLootCorpse 0x57db90: owner touching the corpse with stat 0xd set zeroes it and awards it),
//	  and the stored value is the experience the penalty took (0x533410, DeathPenaltyValue below);
//	VERIFIED too: more than 15 corpses make the game drop the items on the
//	  ground instead (MaxCorpses; DeathInput.CorpseCount, DeathOutcome.DropToFloor).

// RespawnLifeFraction is the share of the maximum life a respawned hero has.
// VERIFIED (SERVER_HandleC2S41_RespawnPlayer 0x549f10): life, mana and stamina (stats 6, 8, 10) are all set to
// their maximum; the hero is moved to the town of the act he is in (mode 1, state 0x36 toggled on and off).
const RespawnLifeFraction = 1.0

// DeathPenaltyValue is the number the exe stores with CLIENT_SetDeathPenaltyValue in the penalty routine
// (0x533410, VERIFIED from the disassembly): the nominal loss percent*span/100 and, when the hero would fall
// to or below the start of his level, exp-start (0 when negative). The hero is then put at start+1, so the
// value can be one more than the experience really lost. Level 1 and a zero loss store nothing (0).
func DeathPenaltyValue(percent, level, exp, start, next int) int {
	if level <= 1 || next <= start || percent <= 0 {
		return 0
	}

	nominal := percent * (next - start) / 100
	if nominal == 0 {
		return 0
	}

	if exp-nominal <= start {
		if v := exp - start; v > 0 {
			return v
		}

		return 0
	}

	return nominal
}

// DeathExpPenaltyPercent returns the penalty percentage of a difficulty (the
// DeathExpPenalty column, see d2difficulty).
func DeathExpPenaltyPercent(difficulty int) int {
	if difficulty < 0 || difficulty >= int(d2difficulty.Count) {
		return 0
	}

	return d2difficulty.DeathExpPenalty(d2difficulty.Level(difficulty))
}

// Corpse is what a hero left behind. X and Y are subtile coordinates.
type Corpse struct {
	X         int                            `json:"x"`
	Y         int                            `json:"y"`
	Equipment d2inventory.CharacterEquipment `json:"equipment"`
	// ExpLost and GoldDropped record the penalties of that death.
	ExpLost     int `json:"expLost"`
	GoldDropped int `json:"goldDropped"`
	// Experience is the corpse's experience stat (stat 0xd): 75% of the
	// death-penalty value of the dying client (CorpseExperience). Its use is
	// UNVERIFIED; it is only recorded.
	Experience int `json:"experience,omitempty"`
	// Items are the worn items (Page PageEquipped, X the .d2s body location) and the item that was on the cursor
	// (any other page) the corpse holds; the .d2s corpse section round-trips them (w50). Empty for corpses of older
	// saves, whose Equipment field is all there is.
	Items []StoredItem `json:"items,omitempty"`
}

// MaxCorpses is the corpse count above which the exe makes no new corpse and
// drops the items on the floor instead (0x57d6f0: more than 0xf entries,
// VERIFIED).
const MaxCorpses = 15

// CorpseOverCap reports whether a hero that already owns corpses corpses
// gets no new one.
func CorpseOverCap(corpses int) bool { return corpses > MaxCorpses }

// CorpseExperience is the experience stat of a new corpse: 75% (0x4b/100) of
// the client's death-penalty value, rounded down. The exe has three arithmetic
// paths so big values do not overflow; this computes in 64 bits, which agrees
// except possibly in the last unit for values above 0x100000 (UNVERIFIED).
func CorpseExperience(penaltyValue int) int {
	if penaltyValue <= 0 {
		return 0
	}

	return int(int64(penaltyValue) * 75 / 100)
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
	// CorpseCount is how many corpses the hero already owns in the game and
	// PenaltyValue the client's death-penalty value (see CorpseExperience).
	CorpseCount, PenaltyValue int
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
	// DropToFloor is true when no corpse was made (MaxCorpses reached): the
	// cursor item, the equipped items and the gold lie on the floor.
	DropToFloor bool
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

	if CorpseOverCap(in.CorpseCount) {
		out.DropToFloor = true

		return out
	}

	if in.PenaltyValue == 0 {
		in.PenaltyValue = DeathPenaltyValue(DeathExpPenaltyPercent(in.Difficulty), in.Level, in.Experience, in.ExpStart, in.ExpNext)
	}

	s.Corpse = &Corpse{X: in.X, Y: in.Y, Equipment: in.Equipment, ExpLost: lost, GoldDropped: in.Gold,
		Experience: CorpseExperience(in.PenaltyValue)}
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

// RespawnLife returns the life points of a respawned hero (at least 1): the maximum.
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
