package herogen

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// Skills.txt ids of the Barbarian skills the preset uses.
const (
	SkillBash           = 126
	SkillSwordMastery   = 127
	SkillHowl           = 130
	SkillLeap           = 132
	SkillDoubleSwing    = 133
	SkillTaunt          = 137
	SkillShout          = 138
	SkillStun           = 139
	SkillDoubleThrow    = 140
	SkillIncreasedStam  = 141
	SkillLeapAttack     = 143
	SkillConcentrate    = 144
	SkillIronSkin       = 145
	SkillBattleCry      = 146
	SkillFrenzy         = 147
	SkillIncreasedSpeed = 148
	SkillBattleOrders   = 149
	SkillWhirlwind      = 151
	SkillNaturalResist  = 153
)

// Barbarian is a level 94 Barbarian who has finished the game: a Frenzy
// fighter (Battle Orders on the right button) with a unique two-hand sword and a full
// set of unique gear that a level 94 hero can wear (every item is a
// non-ladder UniqueItems row whose level requirement is at most 94 and whose
// strength and dexterity requirements the attributes meet).
//
// Attributes: 465 level-up points and the 5 of Lam Esen's Tome are all spent.
// Strength 189 and dexterity 110 are the requirements of The Grandfather
// (strength 189, dexterity 110), vitality takes the remaining 221 points and
// energy stays at the class start. With the gear the hero has about 1,800
// life, which is the point of the preset: it survives the scripted fights.
//
// Skills: the 105 points (93 from levels, 12 from quests) go to Frenzy (the
// main attack, on the left button), Sword Mastery and Battle Orders at 20,
// Double Swing and Taunt at 6 (Frenzy's synergies), Whirlwind and Natural
// Resistance at 10, the skills they need as prerequisites at 1 (Iron Skin
// among them) and Increased Speed at 2 (Increased Stamina 1).
func Barbarian(name string) Spec {
	return Spec{
		Name: name, Class: d2s.Barbarian, Level: 94,
		Strength: 189, Dexterity: 110, Vitality: 25 + 221, Energy: 10,
		Skills: map[int]int{
			SkillBash: 1, SkillSwordMastery: 20, SkillHowl: 1, SkillDoubleSwing: 6, SkillTaunt: 6, SkillShout: 1,
			SkillStun: 1, SkillDoubleThrow: 1, SkillIncreasedStam: 1, SkillLeap: 1, SkillLeapAttack: 1,
			SkillConcentrate: 1, SkillIronSkin: 1, SkillBattleCry: 1, SkillFrenzy: 20, SkillIncreasedSpeed: 2,
			SkillBattleOrders: 20, SkillWhirlwind: 10, SkillNaturalResist: 10,
		},
		// Frenzy on the left button is what a scripted fight swings with (the first version of the preset
		// used the plain attack and Whirlwind on the right button, and passed the Act 1 to 5 chain that way);
		// Battle Orders on the right button is the buff a fight starts with
		Left: SkillFrenzy, Right: SkillBattleOrders,
		Hotkeys: []int{SkillFrenzy, SkillBattleOrders, SkillWhirlwind, SkillLeap, SkillShout},
		Gold:    250000,
		Items: []ItemSpec{
			{Unique: "The Grandfather", Place: Worn(SlotRightHand)},
			{Unique: "Arreat's Face", Place: Worn(SlotHead)},
			{Unique: "Arkaine's Valor", Place: Worn(SlotTorso)},
			{Unique: "Dracul's Grasp", Place: Worn(SlotGloves)},
			{Unique: "Verdugo's Hearty Cord", Place: Worn(SlotBelt)},
			{Unique: "Sandstorm Trek", Place: Worn(SlotFeet)},
			{Unique: "Mara's Kaleidoscope", Place: Worn(SlotAmulet)},
			{Unique: "Bul Katho's Wedding Band", Place: Worn(SlotRingRight)},
			{Unique: "Raven Frost", Place: Worn(SlotRingLeft)},
			{Unique: "Annihilus", ILvl: 110, Place: InInventory(0, 0)},
			{Code: "tbk", Quantity: 20, Place: InInventory(1, 0)},
			{Code: "ibk", Quantity: 20, Place: InInventory(2, 0)},
			// the belt (Mithril Coil, four rows): super healing and rejuvenation in the front row of the first
			// two columns would be drunk first, so every row repeats the pattern healing, healing, rejuvenation,
			// mana
			{Code: "hp5", Place: InBelt(0)}, {Code: "hp5", Place: InBelt(1)}, {Code: "rvl", Place: InBelt(2)},
			{Code: "mp5", Place: InBelt(3)},
			{Code: "hp5", Place: InBelt(4)}, {Code: "hp5", Place: InBelt(5)}, {Code: "rvl", Place: InBelt(6)},
			{Code: "mp5", Place: InBelt(7)},
			{Code: "hp5", Place: InBelt(8)}, {Code: "hp5", Place: InBelt(9)}, {Code: "rvl", Place: InBelt(10)},
			{Code: "mp5", Place: InBelt(11)},
			{Code: "hp5", Place: InBelt(12)}, {Code: "hp5", Place: InBelt(13)}, {Code: "rvl", Place: InBelt(14)},
			{Code: "mp5", Place: InBelt(15)},
		},
	}
}
