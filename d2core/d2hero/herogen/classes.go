package herogen

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// The seven level 94 presets. Each one is a hero that has finished the game on
// all three difficulties, built the way a player of the class builds it:
//
//   - attributes: the 470 points of the levels and Lam Esen's Tome are all
//     spent (strength and dexterity for the gear the hero wears, energy where
//     the class casts, the rest in vitality);
//   - skills: the 105 points (93 of the levels, 12 of the quests) go to one
//     main attack at 20 (the left button), the skills that attack needs as
//     prerequisites at 1, the synergies it grows from (the EDmgSymPerCalc
//     columns of skills.txt), and three to five support skills (a summon, an
//     aura, a buff, a passive) one of which is on the right button;
//   - gear: unique items of the UniqueItems table that are not ladder only,
//     that a level 94 hero can wear (the game's equip rules are applied to
//     the worn set, see Tables.CheckWorn), among them the class items the
//     class is made for (javelins, orbs, voodoo heads, auric shields, primal
//     helms, pelts, claws);
//   - a full belt (four rows) of healing, rejuvenation and mana potions, two
//     tomes and Annihilus.
//
// Hotkeys F1.. hold the main attack first, then the supports; Left is what a
// scripted fight casts at its target (see d2gamescreen/fightskill.go).

// presetNames maps the OD2_HERO names to classes.
var presetNames = map[string]d2s.Class{
	"amazon": d2s.Amazon, "ama": d2s.Amazon,
	"sorc": d2s.Sorceress, "sorceress": d2s.Sorceress,
	"necro": d2s.Necromancer, "necromancer": d2s.Necromancer,
	"paladin": d2s.Paladin, "pal": d2s.Paladin,
	"barb": d2s.Barbarian, "barbarian": d2s.Barbarian,
	"druid": d2s.Druid, "dru": d2s.Druid,
	"assassin": d2s.Assassin, "sin": d2s.Assassin, "ass": d2s.Assassin,
}

// PresetNames lists the canonical OD2_HERO names, in class order.
func PresetNames() []string {
	return []string{"amazon", "sorc", "necro", "paladin", "barb", "druid", "assassin"}
}

// ClassOfPreset resolves an OD2_HERO name (amazon, sorc, necro, paladin, barb,
// druid, assassin, and the long class names).
func ClassOfPreset(name string) (d2s.Class, bool) {
	c, ok := presetNames[strings.ToLower(strings.TrimSpace(name))]

	return c, ok
}

// DefaultName is the character name of a preset ("NokkaBarb"): letters only,
// two to fifteen characters, like the game demands.
func DefaultName(c d2s.Class) string {
	return "Nokka" + [...]string{"Ama", "Sorc", "Necro", "Pala", "Barb", "Druid", "Sin"}[c]
}

// Preset returns the preset of a class.
func Preset(c d2s.Class, name string) (Spec, error) {
	switch c {
	case d2s.Amazon:
		return Amazon(name), nil
	case d2s.Sorceress:
		return Sorceress(name), nil
	case d2s.Necromancer:
		return Necromancer(name), nil
	case d2s.Paladin:
		return Paladin(name), nil
	case d2s.Barbarian:
		return Barbarian(name), nil
	case d2s.Druid:
		return Druid(name), nil
	case d2s.Assassin:
		return Assassin(name), nil
	}

	return Spec{}, fmt.Errorf("%w: no preset for class %v", ErrSpec, c)
}

// beltOf fills the four rows of a 16-cell belt: column c holds front[c] in every row.
func beltOf(front [4]string) []ItemSpec {
	var out []ItemSpec

	for row := uint8(0); row < 4; row++ {
		for col := uint8(0); col < 4; col++ {
			out = append(out, ItemSpec{Code: front[col], Place: InBelt(row*4 + col)})
		}
	}

	return out
}

// Melee and ranged fighters drink life first, casters keep a second column of mana.
var (
	fighterBelt = [4]string{"hp5", "hp5", "rvl", "mp5"}
	casterBelt  = [4]string{"hp5", "rvl", "mp5", "mp5"}
)

// pack is what every hero carries: Annihilus, the two tomes and the belt.
func pack(belt [4]string) []ItemSpec {
	out := []ItemSpec{
		{Unique: "Annihilus", ILvl: 110, Place: InInventory(0, 0)},
		{Code: "tbk", Quantity: 20, Place: InInventory(1, 0)},
		{Code: "ibk", Quantity: 20, Place: InInventory(2, 0)},
	}

	return append(out, beltOf(belt)...)
}

// worn lists unique items by slot, in the order head, amulet, torso, right
// hand, left hand, right ring, left ring, belt, boots, gloves ("" skips the slot).
func worn(head, amulet, torso, rhand, lhand, rring, lring, belt, boots, gloves string) []ItemSpec {
	var out []ItemSpec

	for _, w := range []struct {
		name string
		slot uint8
	}{
		{head, SlotHead}, {amulet, SlotAmulet}, {torso, SlotTorso}, {rhand, SlotRightHand}, {lhand, SlotLeftHand},
		{rring, SlotRingRight}, {lring, SlotRingLeft}, {belt, SlotBelt}, {boots, SlotFeet}, {gloves, SlotGloves},
	} {
		if w.name != "" {
			out = append(out, ItemSpec{Unique: w.name, Place: Worn(w.slot)})
		}
	}

	return out
}

// Amazon is a level 94 Javelin Amazon: Lightning Fury at 20 thrown from
// Titan's Revenge, fed by Lightning Bolt, Power Strike, Charged Strike and
// Lightning Strike (its synergies), with Valkyrie as the summon, Critical
// Strike as the passive and Titan's Revenge's dexterity requirement met.
//
// Attributes: strength 100 (Blackoak Shield), dexterity 110 (Titan's Revenge
// needs 109), energy at the start, vitality takes the rest.
func Amazon(name string) Spec {
	const c = d2s.Amazon

	return Spec{
		Name: name, Class: c, Level: 94,
		Strength: 100, Dexterity: 110, Vitality: 20 + 305, Energy: 15,
		Skills: skillMap(c,
			SP{"Lightning Fury", 20}, SP{"Lightning Bolt", 20}, SP{"Power Strike", 20}, SP{"Charged Strike", 8},
			SP{"Lightning Strike", 5}, SP{"Jab", 1}, SP{"Poison Javelin", 1}, SP{"Plague Javelin", 1},
			SP{"Inner Sight", 1}, SP{"Slow Missiles", 1}, SP{"Dopplezon", 1}, SP{"Dodge", 1}, SP{"Avoid", 1},
			SP{"Evade", 1}, SP{"Valkyrie", 10}, SP{"Critical Strike", 13},
		),
		Left: mustSkill(c, "Lightning Fury"), Right: mustSkill(c, "Valkyrie"),
		Hotkeys: hotkeys(c, "Lightning Fury", "Lightning Bolt", "Charged Strike", "Valkyrie", "Dopplezon", "Jab"),
		Gold:    250000,
		Items: append(worn("Harlequin Crest", "Highlord's Wrath", "Skin of the Vipermagi", "Titan's Revenge",
			"Blackoak Shield", "Bul Katho's Wedding Band", "Raven Frost", "Nosferatu's Coil", "Waterwalk", "Dracul's Grasp"),
			pack(fighterBelt)...),
	}
}

// Sorceress is a level 94 Fire Sorceress: Fire Ball at 20 with Fire Bolt and
// Meteor (its synergies) at 20, Fire Wall as the prerequisite of Meteor, Fire
// Mastery and Warmth for damage and mana, Teleport and Frozen Armor as
// support, and Eschuta's Temper in the hand.
//
// Attributes: strength 60 (Lidless Wall needs 58), dexterity and the rest
// at the start except energy, which takes 100 points for the mana pool.
func Sorceress(name string) Spec {
	const c = d2s.Sorceress

	return Spec{
		Name: name, Class: c, Level: 94,
		Strength: 60, Dexterity: 25, Vitality: 10 + 320, Energy: 35 + 100,
		Skills: skillMap(c,
			SP{"Fire Ball", 20}, SP{"Fire Bolt", 20}, SP{"Meteor", 20}, SP{"Inferno", 1}, SP{"Blaze", 1},
			SP{"Fire Wall", 5}, SP{"Warmth", 15}, SP{"Fire Mastery", 15}, SP{"Telekinesis", 1}, SP{"Teleport", 1},
			SP{"Frozen Armor", 6},
		),
		Left: mustSkill(c, "Fire Ball"), Right: mustSkill(c, "Frozen Armor"),
		Hotkeys: hotkeys(c, "Fire Ball", "Fire Bolt", "Meteor", "Fire Wall", "Teleport", "Frozen Armor"),
		Gold:    250000,
		Items: append(worn("Harlequin Crest", "Mara's Kaleidoscope", "Skin of the Vipermagi", "Eschuta's temper",
			"Lidless Wall", "Bul Katho's Wedding Band", "Raven Frost", "Arachnid Mesh", "Waterwalk", "Magefist"),
			pack(casterBelt)...),
	}
}

// Necromancer is a level 94 Bone Necromancer: Bone Spear at 20 with Teeth,
// Bone Spirit, Bone Wall and Bone Prison (its synergies), Corpse Explosion and
// Teeth as its prerequisites, Bone Armor on the right button, Raise Skeleton
// with its mastery and a Clay Golem as the summons, Boneshade (+bone skills)
// in the hand and the voodoo head Boneflame in the other.
//
// Attributes: strength 95 (Boneflame), dexterity at the start, energy 100.
func Necromancer(name string) Spec {
	const c = d2s.Necromancer

	return Spec{
		Name: name, Class: c, Level: 94,
		Strength: 95, Dexterity: 25, Vitality: 15 + 315, Energy: 25 + 75,
		Skills: skillMap(c,
			SP{"Bone Spear", 20}, SP{"Teeth", 20}, SP{"Corpse Explosion", 1}, SP{"Bone Spirit", 15},
			SP{"Bone Armor", 15}, SP{"Bone Wall", 1}, SP{"Bone Prison", 1}, SP{"Raise Skeleton", 15},
			SP{"Skeleton Mastery", 15}, SP{"Amplify Damage", 1}, SP{"Clay Golem", 1},
		),
		Left: mustSkill(c, "Bone Spear"), Right: mustSkill(c, "Bone Armor"),
		Hotkeys: hotkeys(c, "Bone Spear", "Bone Spirit", "Teeth", "Raise Skeleton", "Clay Golem", "Amplify Damage"),
		Gold:    250000,
		Items: append(worn("Harlequin Crest", "Mara's Kaleidoscope", "Ormus' Robes", "Boneshade", "Boneflame",
			"Bul Katho's Wedding Band", "Raven Frost", "Nosferatu's Coil", "Silkweave", "Dracul's Grasp"),
			pack(casterBelt)...),
	}
}

// Paladin is a level 94 Zealot: Zeal at 20 with Sacrifice (its synergy) at 20,
// Fanaticism at 20 on the right button (Might, Blessed Aim and Concentration
// lead to it), Vigor for the stamina, Vengeance, the Divine Scepter Hand of
// Blessed Light and the auric shield Alma Negra.
//
// Attributes: strength 110 (Alma Negra needs 109, Hand of Blessed Light 103),
// dexterity 75 for the attack rating, energy at the start.
func Paladin(name string) Spec {
	const c = d2s.Paladin

	return Spec{
		Name: name, Class: c, Level: 94,
		Strength: 110, Dexterity: 75, Vitality: 25 + 330, Energy: 15,
		Skills: skillMap(c,
			SP{"Zeal", 20}, SP{"Sacrifice", 20}, SP{"Smite", 1}, SP{"Might", 9}, SP{"Blessed Aim", 1},
			SP{"Concentration", 1}, SP{"Fanaticism", 20}, SP{"Prayer", 1}, SP{"Cleansing", 1}, SP{"Defiance", 1},
			SP{"Vigor", 20}, SP{"Vengeance", 10},
		),
		Left: mustSkill(c, "Zeal"), Right: mustSkill(c, "Fanaticism"),
		Hotkeys: hotkeys(c, "Zeal", "Sacrifice", "Vengeance", "Fanaticism", "Vigor", "Might"),
		Gold:    250000,
		Items: append(worn("Harlequin Crest", "Highlord's Wrath", "Duriel's Shell", "Hand of Blessed Light",
			"Alma Negra", "Bul Katho's Wedding Band", "Raven Frost", "Verdugo's Hearty Cord", "Sandstorm Trek",
			"Dracul's Grasp"), pack(fighterBelt)...),
	}
}

// Druid is a level 94 Fire Druid: Firestorm at 20 with Molten Boulder, Eruption
// (Fissure) and Volcano at 20 (a chain of prerequisites that are also the
// synergies), Oak Sage on the right button, Cyclone Armor, the wolf summon and
// the two shapeshifts, the Elder Staff Ondal's Wisdom and the pelt Jalal's Mane.
//
// Attributes: strength 65 (Jalal's Mane needs 65), dexterity 40 (Ondal's Wisdom
// needs 37), energy 80.
func Druid(name string) Spec {
	const c = d2s.Druid

	return Spec{
		Name: name, Class: c, Level: 94,
		Strength: 65, Dexterity: 40, Vitality: 25 + 340, Energy: 20 + 60,
		Skills: skillMap(c,
			SP{"Firestorm", 20}, SP{"Molten Boulder", 20}, SP{"Eruption", 20}, SP{"Volcano", 20}, SP{"Oak Sage", 10},
			SP{"Arctic Blast", 1}, SP{"Cyclone Armor", 5}, SP{"Raven", 1}, SP{"Summon Spirit Wolf", 6},
			SP{"Wearbear", 1}, SP{"Wearwolf", 1},
		),
		Left: mustSkill(c, "Firestorm"), Right: mustSkill(c, "Oak Sage"),
		Hotkeys: hotkeys(c, "Firestorm", "Molten Boulder", "Eruption", "Volcano", "Cyclone Armor", "Summon Spirit Wolf"),
		Gold:    250000,
		Items: append(worn("Jalal's Mane", "Mara's Kaleidoscope", "Skin of the Vipermagi", "Ondal's Wisdom", "",
			"Bul Katho's Wedding Band", "Raven Frost", "Arachnid Mesh", "Silkweave", "Dracul's Grasp"),
			pack(casterBelt)...),
	}
}

// Assassin is a level 94 Martial Artist with two claws: Dragon Talon at 20
// with the Dragon Claw and Dragon Tail kicks behind it, Claw Mastery at 20,
// the charge-ups (Fists of Fire, Claws of Thunder, Blades of Ice) that lead to
// Royal Strike, Burst of Speed (Quickness), Fade and a Shadow Warrior on the
// right button, wielding the claws Jadetalon and Firelizard's Talons.
//
// Attributes: strength 115 and dexterity 115 (the claws need 105 and 113).
func Assassin(name string) Spec {
	const c = d2s.Assassin

	return Spec{
		Name: name, Class: c, Level: 94,
		Strength: 115, Dexterity: 115, Vitality: 20 + 280, Energy: 25,
		Skills: skillMap(c,
			SP{"Dragon Talon", 20}, SP{"Claw Mastery", 20}, SP{"Quickness", 15}, SP{"Fade", 15},
			SP{"Weapon Block", 1}, SP{"Psychic Hammer", 1}, SP{"Cloak of Shadows", 1}, SP{"Shadow Warrior", 15},
			SP{"Dragon Claw", 1}, SP{"Dragon Tail", 10}, SP{"Fists of Fire", 1}, SP{"Claws of Thunder", 1},
			SP{"Blades of Ice", 1}, SP{"Tiger Strike", 1}, SP{"Cobra Strike", 1}, SP{"Royal Strike", 1},
		),
		Left: mustSkill(c, "Dragon Talon"), Right: mustSkill(c, "Shadow Warrior"),
		Hotkeys: hotkeys(c, "Dragon Talon", "Dragon Tail", "Royal Strike", "Shadow Warrior", "Fade", "Quickness"),
		Gold:    250000,
		Items: append(worn("Harlequin Crest", "Highlord's Wrath", "Skin of the Vipermagi", "Jadetalon",
			"Firelizard's Talons", "Bul Katho's Wedding Band", "Raven Frost", "Verdugo's Hearty Cord", "Sandstorm Trek",
			"Dracul's Grasp"), pack(fighterBelt)...),
	}
}
