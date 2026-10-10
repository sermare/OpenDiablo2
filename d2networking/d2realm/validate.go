package d2realm

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// ErrInvalidCharacter wraps every rejection of an uploaded character.
var ErrInvalidCharacter = errors.New("d2realm: character rejected")

// Bounds used by the open-realm validator. They are conservative upper limits
// derived from the game rules; a character that exceeds one cannot have been
// produced by playing (UNVERIFIED against the original server, which trusts
// the client in open realms).
const (
	maxLevel = 99
	// highest starting attribute total of any class (Barbarian 30+20+25+10)
	// plus the 5 points per level and the +5 of the Lam Esen's Tome quest.
	maxBaseAttributes = 85 + 5
	pointsPerLevel    = 5
	// 1 skill point per level after the first, plus Den of Evil (1), Radament
	// (1) and Izual (2) in each of the three difficulties.
	maxQuestSkillPoints = 12
	maxStashGold        = 2500000
	goldPerLevel        = 10000
	maxExperience       = 3520485254
	maxItems            = 2000
	maxSockets          = 6
	maxQuantity         = 511
	maxStatValue        = 1023 // 10 bit attribute fields
)

// Validator checks uploaded .d2s files. Tables may be nil, in which case the
// item list cannot be decoded and items are not checked (the body is still
// parsed with the default core stat widths).
type Validator struct {
	Tables *d2s.ItemTables
}

// Validate parses and range-checks a character file. The returned character
// is only meaningful when err is nil. All errors wrap ErrInvalidCharacter.
func (v *Validator) Validate(data []byte) (*d2s.Character, error) {
	c, err := v.parse(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCharacter, err)
	}

	if err := v.check(c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCharacter, err)
	}

	return c, nil
}

func (v *Validator) parse(data []byte) (*d2s.Character, error) {
	if len(data) > maxUpload {
		return nil, errors.New("file too large")
	}

	if v.Tables != nil {
		return d2s.Parse(data, v.Tables)
	}

	h, err := d2s.ParseHeader(data)
	if err != nil {
		return nil, err
	}

	c := &d2s.Character{Header: h}
	if h.HasBody() {
		if c.Body, err = d2s.ParseBody(data, nil); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (v *Validator) check(c *d2s.Character) error {
	h := c.Header

	if err := d2s.ValidateName(h.Name); err != nil {
		return err
	}

	if h.Class > d2s.Assassin {
		return d2s.ErrInvalidClass
	}

	if !h.IsExpansion() && h.Class >= d2s.Druid {
		return errors.New("expansion class in a classic character")
	}

	if h.Level < 1 || h.Level > maxLevel {
		return fmt.Errorf("level %d out of range", h.Level)
	}

	if _, act, ok := h.ActiveDifficulty(); ok && act > 4 {
		return fmt.Errorf("act %d out of range", act)
	}

	if c.Body == nil { // brand new character
		if h.Level != 1 {
			return fmt.Errorf("new character with level %d", h.Level)
		}

		return nil
	}

	if err := checkAttributes(h, c.Body); err != nil {
		return err
	}

	return v.checkItems(c)
}

func checkAttributes(h *d2s.Header, b *d2s.Body) error {
	a := b.Attributes

	if a.Level != uint64(h.Level) {
		return fmt.Errorf("stat level %d differs from header level %d", a.Level, h.Level)
	}

	level := uint64(h.Level)

	for name, val := range map[string]uint64{"strength": a.Strength, "energy": a.Energy,
		"dexterity": a.Dexterity, "vitality": a.Vitality} {
		if val > maxStatValue {
			return fmt.Errorf("%s %d out of range", name, val)
		}
	}

	attrs := a.Strength + a.Energy + a.Dexterity + a.Vitality + a.UnusedStats
	if limit := maxBaseAttributes + pointsPerLevel*(level-1); attrs > limit {
		return fmt.Errorf("attribute points %d exceed the %d a level %d character can have", attrs, limit, level)
	}

	var spent uint64
	for _, p := range b.SkillPoints {
		spent += uint64(p)
	}

	if limit := level - 1 + maxQuestSkillPoints; spent+a.UnusedSkillPoints > limit {
		return fmt.Errorf("skill points %d exceed the %d a level %d character can have",
			spent+a.UnusedSkillPoints, limit, level)
	}

	if a.Gold > level*goldPerLevel {
		return fmt.Errorf("gold %d exceeds the %d a level %d character can carry", a.Gold, level*goldPerLevel, level)
	}

	if a.StashedGold > maxStashGold {
		return fmt.Errorf("stashed gold %d out of range", a.StashedGold)
	}

	if a.Experience > maxExperience {
		return fmt.Errorf("experience %d out of range", a.Experience)
	}

	if level == 1 && a.Experience > 1<<20 { // level 2 starts at 500
		return fmt.Errorf("experience %d too high for level 1", a.Experience)
	}

	return nil
}

func (v *Validator) checkItems(c *d2s.Character) error {
	n := 0

	lists := [][]d2s.Item{c.Items, c.Corpse, c.MercItems}
	for _, list := range lists {
		for i := range list {
			if err := v.checkItem(&list[i], &n); err != nil {
				return err
			}
		}
	}

	if c.Golem != nil {
		return v.checkItem(c.Golem, &n)
	}

	return nil
}

func (v *Validator) checkItem(it *d2s.Item, count *int) error {
	if *count++; *count > maxItems {
		return errors.New("too many items")
	}

	if it.Ear {
		return nil
	}

	if v.Tables != nil && v.Tables.ItemKindOf(it.Code) == 0 {
		return fmt.Errorf("unknown item code %q", it.Code)
	}

	switch {
	case it.Level > maxLevel:
		return fmt.Errorf("item %q level %d out of range", it.Code, it.Level)
	case !it.Simple && (it.Quality < d2s.QualityLow || it.Quality > d2s.QualityCrafted):
		return fmt.Errorf("item %q quality %d invalid", it.Code, it.Quality)
	case it.TotalSockets > maxSockets || it.SocketCount > maxSockets:
		return fmt.Errorf("item %q has %d sockets", it.Code, it.TotalSockets)
	case it.Quantity > maxQuantity:
		return fmt.Errorf("item %q quantity %d out of range", it.Code, it.Quantity)
	}

	if len(it.Children) > int(it.TotalSockets) && len(it.Children) > maxSockets {
		return fmt.Errorf("item %q has %d socketed items", it.Code, len(it.Children))
	}

	for i := range it.Children {
		if err := v.checkItem(&it.Children[i], count); err != nil {
			return err
		}
	}

	return nil
}

// DifficultyAllowed reports whether a character may play a difficulty. Policy
// (OURS, unverified against the original): a character may play every
// difficulty up to the one its save is currently in. Characters whose save
// has no active difficulty (new characters) may only play Normal.
func DifficultyAllowed(h *d2s.Header, difficulty byte) bool {
	d, _, ok := h.ActiveDifficulty()
	if !ok {
		return difficulty == Normal
	}

	return int(difficulty) <= d
}
