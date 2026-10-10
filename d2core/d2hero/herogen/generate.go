package herogen

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// Quest rewards of a hero that has finished all three difficulties, which the
// level 94 samples have: Lam Esen's Tome (5 stat points), the Potion of Life
// (20 life; the stored maximum includes it, see d2statlist.Class.BaseMax) and
// the skill points of Den of Evil, Radament and Izual in each difficulty.
const (
	LamEsenStatPoints = 5
	PotionOfLifeLife  = 20
	QuestSkillPoints  = 12
	StatPointsPerLvl  = 5
	SkillPointsPerLvl = 1
)

// firstSkillID is the skills.txt id of the first skill of each class; the 30
// skills of a class are consecutive and the save keeps their points in order.
var firstSkillID = [...]int{
	d2s.Amazon: 6, d2s.Sorceress: 36, d2s.Necromancer: 66, d2s.Paladin: 96, d2s.Barbarian: 126, d2s.Druid: 221,
	d2s.Assassin: 251,
}

// FirstSkillID returns the skills.txt id of the first of the class' 30 skills.
func FirstSkillID(c d2s.Class) int { return firstSkillID[c] }

// Spec describes a hero.
type Spec struct {
	Name  string
	Class d2s.Class
	Level int
	// Strength, Dexterity, Vitality and Energy are the base attributes (class
	// start plus the points spent, no item bonuses). The points left from the
	// level-up and quest grants are stored as unspent stat points.
	Strength, Dexterity, Vitality, Energy int
	// Skills maps a skills.txt id to its points.
	Skills map[int]int
	// Left and Right are the skills assigned to the mouse buttons (skills.txt
	// ids; 0 is the plain attack); Hotkeys are skills for F1.., a negative
	// entry is an empty key and a value above 1000 is skill+1000 on the left hand.
	Left, Right int
	Hotkeys     []int
	Gold        uint64
	Items       []ItemSpec
	// Created is the creation time stamp (fixed, so the file is reproducible).
	Created time.Time
}

// Errors of Generate.
var (
	ErrSpec = errors.New("herogen: invalid hero")
)

// Template is the part of an existing save the generated hero shares: the
// quest, waypoint and NPC state, the active difficulty, the map seed and the
// mercenary. Passing the sample Sorceress makes the Barbarian play the very
// same game (same maps, same hired mercenary, same quest flags).
type Template struct {
	Quests     [3][96]byte
	Waypoints  d2s.Waypoints
	NPC        [50]byte
	Difficulty [3]byte
	MapSeed    uint32
	Mercenary  d2s.Mercenary
	// Raw keeps the quest and waypoint sections as read (the bytes the
	// structured fields do not cover).
	QuestsRaw    [0x12A]byte
	WaypointsRaw [0x50]byte
}

// TemplateFrom copies the shared state out of a parsed save.
func TemplateFrom(c *d2s.Character) (*Template, error) {
	if c == nil || c.Header == nil || c.Body == nil {
		return nil, fmt.Errorf("%w: template needs a save with a body", ErrSpec)
	}

	return &Template{
		Quests: c.Body.Quests, Waypoints: c.Body.Waypoints, NPC: c.Body.NPC,
		Difficulty: c.Header.Difficulty, MapSeed: c.Header.MapSeed, Mercenary: c.Header.Mercenary,
		QuestsRaw: c.Body.QuestsRaw, WaypointsRaw: c.Body.WaypointsRaw,
	}, nil
}

// Hero is a generated hero: the character and its file.
type Hero struct {
	Character *d2s.Character
	Data      []byte
	// Totals are life, mana, stamina and combat numbers with the worn items.
	Totals d2statlist.Totals
}

// AllowedStatPoints is the number of points a hero of the level can have
// spent on attributes after all quests.
func AllowedStatPoints(level int) int { return (level-1)*StatPointsPerLvl + LamEsenStatPoints }

// AllowedSkillPoints is the number of skill points of a hero of the level
// after all quests.
func AllowedSkillPoints(level int) int { return (level-1)*SkillPointsPerLvl + QuestSkillPoints }

// Generate builds the hero. tmpl may be nil: the quests then start empty,
// every waypoint of every difficulty is active, hell is the active difficulty
// and there is no mercenary.
func (t *Tables) Generate(spec Spec, tmpl *Template) (*Hero, error) {
	cls, ok := t.Classes[spec.Class.String()]
	if !ok {
		return nil, fmt.Errorf("%w: no CharStats row for %v", ErrSpec, spec.Class)
	}

	exp := t.Exp[spec.Class.String()]
	if exp == nil || spec.Level < 1 || spec.Level > exp.MaxLevel {
		return nil, fmt.Errorf("%w: level %d", ErrSpec, spec.Level)
	}

	spent := spec.Strength + spec.Dexterity + spec.Vitality + spec.Energy -
		(cls.InitStr + cls.InitDex + cls.InitVit + cls.InitEne)
	if spent < 0 || spec.Strength < cls.InitStr || spec.Dexterity < cls.InitDex || spec.Vitality < cls.InitVit ||
		spec.Energy < cls.InitEne {
		return nil, fmt.Errorf("%w: attributes below the class start", ErrSpec)
	}

	unusedStats := AllowedStatPoints(spec.Level) - spent
	if unusedStats < 0 {
		return nil, fmt.Errorf("%w: %d stat points spent, a level %d hero has %d", ErrSpec, spent, spec.Level,
			AllowedStatPoints(spec.Level))
	}

	skillPoints, err := skillArray(spec)
	if err != nil {
		return nil, err
	}

	unusedSkills := AllowedSkillPoints(spec.Level)
	for _, p := range skillPoints {
		unusedSkills -= int(p)
	}

	if unusedSkills < 0 {
		return nil, fmt.Errorf("%w: more skill points than a level %d hero has (%d)", ErrSpec, spec.Level,
			AllowedSkillPoints(spec.Level))
	}

	created := spec.Created
	if created.IsZero() {
		created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	raw, err := d2s.NewCharacter(spec.Name, spec.Class, d2s.NewCharacterFlags{Expansion: true, Created: created},
		d2s.DefaultAppearance(spec.Class))
	if err != nil {
		return nil, err
	}

	c, err := d2s.Parse(raw, nil)
	if err != nil {
		return nil, err
	}

	c.Promote()
	c.Header.Level = uint8(spec.Level)
	c.Header.Difficulty = [3]byte{0, 0, 0x80} // hell, Act I: like the sample hero

	b := c.Body
	for d := range b.Waypoints {
		b.Waypoints[d] = 1<<39 - 1
	}

	if tmpl != nil {
		b.Quests, b.Waypoints, b.NPC = tmpl.Quests, tmpl.Waypoints, tmpl.NPC
		b.QuestsRaw, b.WaypointsRaw = tmpl.QuestsRaw, tmpl.WaypointsRaw
		c.Header.Difficulty, c.Header.MapSeed, c.Header.Mercenary = tmpl.Difficulty, tmpl.MapSeed, tmpl.Mercenary
	}

	b.SkillPoints = skillPoints
	c.Header.SetSkillBlock(skillBlock(spec))

	life, mana, stamina := cls.BaseMax(spec.Level, spec.Vitality, spec.Energy)
	life += PotionOfLifeLife

	set := func(id int, v uint64) { b.SetStat(id, v) }
	set(d2s.StatStrength, uint64(spec.Strength))
	set(d2s.StatEnergy, uint64(spec.Energy))
	set(d2s.StatDexterity, uint64(spec.Dexterity))
	set(d2s.StatVitality, uint64(spec.Vitality))
	set(d2s.StatUnusedStats, uint64(unusedStats))
	set(d2s.StatUnusedSkills, uint64(unusedSkills))
	set(d2s.StatCurrentHP, uint64(life)<<8)
	set(d2s.StatMaxHP, uint64(life)<<8)
	set(d2s.StatCurrentMana, uint64(mana)<<8)
	set(d2s.StatMaxMana, uint64(mana)<<8)
	set(d2s.StatCurrentStam, uint64(stamina)<<8)
	set(d2s.StatMaxStamina, uint64(stamina)<<8)
	set(d2s.StatLevel, uint64(spec.Level))
	// a third of the way into the level, so a kill never levels up and a death penalty has something to take
	set(d2s.StatExperience, uint64(exp.Threshold[spec.Level-1]+(exp.NextLevelExp(spec.Level)-exp.Threshold[spec.Level-1])/3))
	set(d2s.StatGold, spec.Gold)
	set(d2s.StatStashedGold, 0)

	for i, is := range spec.Items {
		it, err := t.buildItem(is, i)
		if err != nil {
			return nil, err
		}

		c.Items = append(c.Items, it)
	}

	if err := checkPlacement(c.Items); err != nil {
		return nil, err
	}

	// the stored maxima are the values without items; the current values of a save written at full
	// health are the totals with the worn items (verified on the real level 94 Sorceress)
	h := d2statlist.Hero{
		Class: cls, Level: spec.Level, Str: spec.Strength, Dex: spec.Dexterity, Vit: spec.Vitality, Ene: spec.Energy,
		BaseLife: life, BaseMana: mana, BaseStam: stamina, Difficulty: 2,
	}
	tot := d2statlist.Compute(h, statItems(c.Items, t.Bases), nil)

	set(d2s.StatCurrentHP, uint64(tot.MaxLife)<<8)
	set(d2s.StatCurrentMana, uint64(tot.MaxMana)<<8)
	set(d2s.StatCurrentStam, uint64(tot.MaxStamina)<<8)

	data, err := d2s.Write(c, t.Save)
	if err != nil {
		return nil, err
	}

	return &Hero{Character: c, Data: data, Totals: tot}, nil
}

// skillArray lays the skills out in the order of the class' 30 skills.
func skillArray(spec Spec) (out [30]byte, err error) {
	first := FirstSkillID(spec.Class)

	for id, pts := range spec.Skills {
		if id < first || id >= first+30 {
			return out, fmt.Errorf("%w: skill %d is not a %v skill", ErrSpec, id, spec.Class)
		}

		if pts < 0 || pts > 20 {
			return out, fmt.Errorf("%w: skill %d with %d points", ErrSpec, id, pts)
		}

		out[id-first] = byte(pts)
	}

	return out, nil
}

func skillBlock(spec Spec) d2s.SkillBlock {
	none := d2s.SkillWord{Skill: 0xFFFF}

	var b d2s.SkillBlock

	for i := range b.Hotkeys {
		b.Hotkeys[i] = none

		if i < len(spec.Hotkeys) && spec.Hotkeys[i] >= 0 {
			v := spec.Hotkeys[i]
			if v >= 1000 {
				b.Hotkeys[i].Skill = uint16(v-1000) | 0x8000
			} else {
				b.Hotkeys[i].Skill = uint16(v)
			}
		}
	}

	b.Left, b.Right = d2s.SkillWord{Skill: uint16(spec.Left)}, d2s.SkillWord{Skill: uint16(spec.Right)}
	b.LeftSwap, b.RightSwap = b.Left, b.Right

	return b
}

// checkPlacement refuses two items in one slot or cell.
func checkPlacement(items []d2s.Item) error {
	seen := map[string]string{}

	for i := range items {
		it := &items[i]

		var key string

		switch it.Location {
		case d2s.LocationEquipped:
			key = fmt.Sprintf("worn slot %d", it.Equipped)
		case d2s.LocationBelt:
			key = fmt.Sprintf("belt cell %d", it.X)
		default:
			// the grid cells of larger items are not tracked: the preset keeps a free cell between them
			key = fmt.Sprintf("page %d cell %d,%d", it.Page, it.X, it.Y)
		}

		if prev, dup := seen[key]; dup {
			return fmt.Errorf("%w: %s holds both %s and %s", ErrSpec, key, prev, strings.TrimSpace(it.Code))
		}

		seen[key] = strings.TrimSpace(it.Code)
	}

	return nil
}

// statItems converts the items that act on the hero (worn items and the
// charms of the inventory page) into stat list items, like d2hero's
// StatItemsFromD2S (a test compares the two).
func statItems(items []d2s.Item, bases d2statlist.Bases) []d2statlist.Item {
	var out []d2statlist.Item

	for i := range items {
		it := &items[i]
		code := strings.TrimSpace(it.Code)
		charm := len(code) == 3 && strings.HasPrefix(code, "cm") && it.Location == d2s.LocationStored &&
			it.Page == PageInventory

		if it.Simple || (it.Location != d2s.LocationEquipped && !charm) {
			continue
		}

		si := d2statlist.Item{
			Code: code, Charm: charm, Ethereal: it.Ethereal, Defense: it.Defense, Runeword: it.Runeword,
			SetID: int(it.SetID), Broken: it.MaxDurability > 0 && it.Durability == 0,
		}

		for _, p := range it.Properties {
			si.Props = append(si.Props, d2statlist.Prop{ID: p.ID, Param: int(p.Param), Value: p.Value})
		}

		if !charm {
			si.Slot = int(it.Equipped)
		}

		if b, ok := bases[code]; ok {
			si.Weapon, si.BaseBlock = b.Weapon, b.BaseBlock
		}

		out = append(out, si)
	}

	return out
}
