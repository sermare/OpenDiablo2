package d2quest

import "errors"

// The Cow Level portal of the Horadric Cube (recipe output "Cow Portal"), wave 45, read from Game.exe 1.14b
// (cube output function 0x591ea0, reached through the thunk 0x563870; own words).
//
// The checks, in the order the exe makes them; a failed check makes the recipe do nothing (the items stay in the
// cube and the hero gets message 0x14 of the server's feedback table):
//  1. the game-wide quest record (not the hero's) has slot 4 bit 11 clear: only ONE cow portal per game;
//  2. the hero's record of the game difficulty has slot 4 bit 10 (the Cow King's kill, FlagCustom6) clear;
//  3. the hero's record of the game difficulty shows the final boss done (bit 0 of slot 26, Terror's End, in a
//     classic game; bit 0 of slot 40, Eve of Destruction, in an expansion game);
//  4. the hero stands in level 1 (the Rogue Encampment): the room's level id must be 1.
//
// On success a permanent portal (object class 60) to level 39 is placed at the hero (level 39 is the only target
// the portal factory 0x56ae80 lets through in a town besides the Pandemonium levels), the game-wide bit 11 of slot
// 4 is set, and the cube's contents are consumed.
const (
	cowQuestSlot     = 4
	cowPortalMadeBit = 11 // game-wide, "a cow portal was made in this game"
	cowTown          = 1  // Rogue Encampment
)

// Errors of CowPortalCheck.
var (
	ErrCowPortalMade   = errors.New("the Cow Level portal was already opened in this game")
	ErrCowKingKilled   = errors.New("the Cow King was already killed on this difficulty")
	ErrCowBossNotDone  = errors.New("the final boss of the game type is not done on this difficulty")
	ErrCowNotInTown    = errors.New("the Cow Level portal opens only in the Rogue Encampment")
	errCowNoQuestState = errors.New("no quest record")
)

// CowPortalCheck says whether the cube may open the Cow Level portal for a hero standing in level.
func (g *Game) CowPortalCheck(level int) error {
	if g == nil || g.Rec == nil {
		return errCowNoQuestState
	}

	if g.global.Get(cowQuestSlot, cowPortalMadeBit) {
		return ErrCowPortalMade
	}

	if g.Rec.Get(cowQuestSlot, FlagCustom6) {
		return ErrCowKingKilled
	}

	slot := 26 // A4Q3 Terror's End
	if g.Expansion {
		slot = 40 // A5Q6 Eve of Destruction
	}

	if !g.Rec.Get(slot, FlagRewardGranted) {
		return ErrCowBossNotDone
	}

	if level != cowTown {
		return ErrCowNotInTown
	}

	return nil
}

// CowPortalMade records that the portal was made (game-wide bit).
func (g *Game) CowPortalMade() { g.global.Set(cowQuestSlot, cowPortalMadeBit) }
