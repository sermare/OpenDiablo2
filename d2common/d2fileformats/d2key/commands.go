package d2key

// Confidence says how well a command id's meaning is established.
type Confidence int

// Confidence levels.
const (
	// Guess: inferred from the default key only; not confirmed.
	Guess Confidence = iota
	// Likely: the default key and the position in the id list agree with
	// the known in-game Configure Controls screen. Still not read from the
	// binary: names are not stored in default.key.
	Likely
)

// Command describes one command id.
type Command struct {
	ID         uint32
	Name       string
	Confidence Confidence
}

// Commands lists the command ids whose meaning is inferred, keyed by id.
// Ids 8-11 (F9-F12) and 46-53 are deliberately absent: unidentified.
var Commands = map[uint32]Command{
	0:  {0, "CharacterPanel", Likely},
	1:  {1, "InventoryPanel", Likely},
	2:  {2, "PartyPanel", Likely},
	3:  {3, "MessageLog", Likely},
	4:  {4, "QuestLog", Likely},
	5:  {5, "ChatBox", Likely},
	6:  {6, "HelpScreen", Likely},
	7:  {7, "Automap", Likely},
	12: {12, "SkillTree", Likely},
	13: {13, "RightSkillSelector", Guess},
	14: {14, "UseSkill1", Likely}, 15: {15, "UseSkill2", Likely}, 16: {16, "UseSkill3", Likely},
	17: {17, "UseSkill4", Likely}, 18: {18, "UseSkill5", Likely}, 19: {19, "UseSkill6", Likely},
	20: {20, "UseSkill7", Likely}, 21: {21, "UseSkill8", Likely},
	22: {22, "ToggleBelts", Likely},
	23: {23, "UseBeltSlot1", Likely}, 24: {24, "UseBeltSlot2", Likely},
	25: {25, "UseBeltSlot3", Likely}, 26: {26, "UseBeltSlot4", Likely},
	27: {27, "SayHelp", Guess}, 28: {28, "SayFollowMe", Guess}, 29: {29, "SayThisIsForYou", Guess},
	30: {30, "SayThanks", Guess}, 31: {31, "SaySorry", Guess}, 32: {32, "SayBye", Guess},
	33: {33, "SayNowYouDie", Guess}, 55: {55, "SayRetreat", Guess},
	34: {34, "HoldRun", Guess},
	35: {35, "ToggleRunWalk", Likely},
	36: {36, "HoldStandStill", Likely},
	37: {37, "HoldShowGroundItems", Likely},
	38: {38, "ClearScreen", Guess},
	39: {39, "SelectPreviousSkill", Likely},
	40: {40, "SelectNextSkill", Likely},
	41: {41, "ClearMessages", Guess},
	42: {42, "TakeScreenShot", Guess},
	43: {43, "HoldShowPortraits", Guess},
	44: {44, "SwapWeapons", Likely},
	45: {45, "ToggleMiniMap", Guess},
	54: {54, "HirelingPanel", Guess},
	56: {56, "GameMenu", Likely},
}
