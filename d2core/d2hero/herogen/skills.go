package herogen

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// classSkillNames lists the 30 skills of each class in skills.txt order (the
// save keeps the points in this order, see FirstSkillID). The names are the
// "skill" column of the patch_d2 skills.txt, spelling included ("Wearwolf",
// "BloodGolem", "Fire Trauma"); a test compares them with the table.
var classSkillNames = map[d2s.Class][30]string{
	d2s.Amazon: {
		"Magic Arrow", "Fire Arrow", "Inner Sight", "Critical Strike", "Jab", "Cold Arrow", "Multiple Shot", "Dodge",
		"Power Strike", "Poison Javelin", "Exploding Arrow", "Slow Missiles", "Avoid", "Impale", "Lightning Bolt",
		"Ice Arrow", "Guided Arrow", "Penetrate", "Charged Strike", "Plague Javelin", "Strafe", "Immolation Arrow",
		"Dopplezon", "Evade", "Fend", "Freezing Arrow", "Valkyrie", "Pierce", "Lightning Strike", "Lightning Fury",
	},
	d2s.Sorceress: {
		"Fire Bolt", "Warmth", "Charged Bolt", "Ice Bolt", "Frozen Armor", "Inferno", "Static Field", "Telekinesis",
		"Frost Nova", "Ice Blast", "Blaze", "Fire Ball", "Nova", "Lightning", "Shiver Armor", "Fire Wall", "Enchant",
		"Chain Lightning", "Teleport", "Glacial Spike", "Meteor", "Thunder Storm", "Energy Shield", "Blizzard",
		"Chilling Armor", "Fire Mastery", "Hydra", "Lightning Mastery", "Frozen Orb", "Cold Mastery",
	},
	d2s.Necromancer: {
		"Amplify Damage", "Teeth", "Bone Armor", "Skeleton Mastery", "Raise Skeleton", "Dim Vision", "Weaken",
		"Poison Dagger", "Corpse Explosion", "Clay Golem", "Iron Maiden", "Terror", "Bone Wall", "Golem Mastery",
		"Raise Skeletal Mage", "Confuse", "Life Tap", "Poison Explosion", "Bone Spear", "BloodGolem", "Attract",
		"Decrepify", "Bone Prison", "Summon Resist", "IronGolem", "Lower Resist", "Poison Nova", "Bone Spirit",
		"FireGolem", "Revive",
	},
	d2s.Paladin: {
		"Sacrifice", "Smite", "Might", "Prayer", "Resist Fire", "Holy Bolt", "Holy Fire", "Thorns", "Defiance",
		"Resist Cold", "Zeal", "Charge", "Blessed Aim", "Cleansing", "Resist Lightning", "Vengeance", "Blessed Hammer",
		"Concentration", "Holy Freeze", "Vigor", "Conversion", "Holy Shield", "Holy Shock", "Sanctuary", "Meditation",
		"Fist of the Heavens", "Fanaticism", "Conviction", "Redemption", "Salvation",
	},
	d2s.Barbarian: {
		"Bash", "Sword Mastery", "Axe Mastery", "Mace Mastery", "Howl", "Find Potion", "Leap", "Double Swing",
		"Pole Arm Mastery", "Throwing Mastery", "Spear Mastery", "Taunt", "Shout", "Stun", "Double Throw",
		"Increased Stamina", "Find Item", "Leap Attack", "Concentrate", "Iron Skin", "Battle Cry", "Frenzy",
		"Increased Speed", "Battle Orders", "Grim Ward", "Whirlwind", "Berserk", "Natural Resistance", "War Cry",
		"Battle Command",
	},
	d2s.Druid: {
		"Raven", "Plague Poppy", "Wearwolf", "Shape Shifting", "Firestorm", "Oak Sage", "Summon Spirit Wolf",
		"Wearbear", "Molten Boulder", "Arctic Blast", "Cycle of Life", "Feral Rage", "Maul", "Eruption",
		"Cyclone Armor", "Heart of Wolverine", "Summon Fenris", "Rabies", "Fire Claws", "Twister", "Vines", "Hunger",
		"Shock Wave", "Volcano", "Tornado", "Spirit of Barbs", "Summon Grizzly", "Fury", "Armageddon", "Hurricane",
	},
	d2s.Assassin: {
		"Fire Trauma", "Claw Mastery", "Psychic Hammer", "Tiger Strike", "Dragon Talon", "Shock Field",
		"Blade Sentinel", "Quickness", "Fists of Fire", "Dragon Claw", "Charged Bolt Sentry", "Wake of Fire Sentry",
		"Weapon Block", "Cloak of Shadows", "Cobra Strike", "Blade Fury", "Fade", "Shadow Warrior", "Claws of Thunder",
		"Dragon Tail", "Lightning Sentry", "Inferno Sentry", "Mind Blast", "Blades of Ice", "Dragon Flight",
		"Death Sentry", "Blade Shield", "Venom", "Shadow Master", "Royal Strike",
	},
}

// ClassSkillNames returns the 30 skill names of a class in skills.txt order.
func ClassSkillNames(c d2s.Class) [30]string { return classSkillNames[c] }

// SkillID returns the skills.txt id of a class skill by name.
func SkillID(c d2s.Class, name string) (int, bool) {
	for i, n := range classSkillNames[c] {
		if n == name {
			return FirstSkillID(c) + i, true
		}
	}

	return 0, false
}

// SkillName returns the name of a skills.txt id of the class ("" for a foreign id).
func SkillName(c d2s.Class, id int) string {
	if i := id - FirstSkillID(c); i >= 0 && i < 30 {
		return classSkillNames[c][i]
	}

	return ""
}

// SP is a skill and the points put into it (a preset is a list of them).
type SP struct {
	Name   string
	Points int
}

// skillMap turns a preset's skill list into the id -> points map of a Spec.
// An unknown name is a bug of the preset and panics (the tests build every
// preset); a skill listed twice is added up.
func skillMap(c d2s.Class, list ...SP) map[int]int {
	out := map[int]int{}

	for _, s := range list {
		id, ok := SkillID(c, s.Name)
		if !ok {
			panic(fmt.Sprintf("herogen: %v has no skill %q", c, s.Name))
		}

		out[id] += s.Points
	}

	return out
}

// mustSkill returns a class skill id or panics (see skillMap).
func mustSkill(c d2s.Class, name string) int {
	id, ok := SkillID(c, name)
	if !ok {
		panic(fmt.Sprintf("herogen: %v has no skill %q", c, name))
	}

	return id
}

// hotkeys lists skill ids for F1.. by name.
func hotkeys(c d2s.Class, names ...string) []int {
	out := make([]int, len(names))
	for i, n := range names {
		out[i] = mustSkill(c, n)
	}

	return out
}
