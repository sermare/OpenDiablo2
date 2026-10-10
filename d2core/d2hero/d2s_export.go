package d2hero

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// lastPlayedOffset is where a .d2s header keeps the time it was last saved
// (unix seconds; seen at this offset in 1.14b saves, unverified for others).
const lastPlayedOffset = 0x30

// maxGold is the largest value the 25-bit gold stat can hold.
const maxGold = 1<<25 - 1

// ErrNoOriginal is returned when a hero has no imported .d2s to start from.
var ErrNoOriginal = errors.New("d2s: hero has no original .d2s to export onto")

// ExportOptions tune ExportD2SWithOptions. The zero value is valid.
type ExportOptions struct {
	// SkillIDs are the ids of the class' skills in ascending order, which is
	// the order of the 30 skill slots of a .d2s. When nil, skill points are
	// left as they were in the original.
	SkillIDs []int
	// LastPlayed is written to the header when non-zero.
	LastPlayed time.Time
	// Affixes maps rolled item names to save ids; with it (and the hero's
	// containers) the items of the containers are written: moved, removed and
	// made in the game (MergeContainerItems).
	Affixes *AffixIDs
	// Known says whether the engine has a record for a base item code.
	Known func(code string) bool
	// Warn receives a message for everything the engine state could not be
	// written faithfully. May be nil.
	Warn func(msg string)
}

// ExportD2S writes the hero back into a .d2s file. It starts from the original
// bytes the hero was imported from, so everything the engine does not model
// (unknown header bytes, items, mercenary, corpse, golem, unmodelled quest
// bits and NPC flags) is kept, and overwrites only the fields the engine owns
// and has changed. With an unchanged hero the output equals the original.
func ExportD2S(state *HeroState, original []byte, tables *d2s.ItemTables) ([]byte, error) {
	out, _, err := ExportD2SWithOptions(state, original, tables, ExportOptions{})

	return out, err
}

// ExportD2SWithOptions is ExportD2S with options; it also returns the warnings.
func ExportD2SWithOptions(state *HeroState, original []byte, tables *d2s.ItemTables,
	opts ExportOptions) ([]byte, []string, error) {
	if state == nil {
		return nil, nil, errors.New("d2s: nil hero state")
	}

	if len(original) == 0 {
		return nil, nil, ErrNoOriginal
	}

	c, err := d2s.Parse(original, tables)
	if err != nil {
		return nil, nil, fmt.Errorf("original: %w", err)
	}

	var warnings []string

	// a character that was never saved by the game has only a header: give it
	// the body sections the game writes on its first save
	if c.Body == nil && c.Header.IsNewCharacter() && state.Stats != nil {
		c.Promote()
	}

	warn := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		warnings = append(warnings, msg)

		if opts.Warn != nil {
			opts.Warn(msg)
		}
	}

	if c.Body != nil && state.Stats != nil {
		exportAttributes(c, state, warn)
		exportSkills(c.Body, state, opts.SkillIDs)
		origQuests := questRecordsOf(c.Body)
		exportProgress(c.Body, state)
		exportProgression(c.Header, origQuests, state)
		// a character without any item in its file (a new one) gets no starting
		// items written: item bits are never fabricated, so there is nothing to compare
		if len(c.Items) > 0 {
			exportEquipment(c, state, warn)
			checkEquipment(c, state, warn)
		}
	}

	if opts.Affixes != nil && state.Containers != nil {
		// the counts are reported by NewItemsInD2S; only the items that could not be written warn
		MergeContainerItems(c, state.Containers, tables, opts.Affixes, opts.Known, warn)
	}

	exportWorld(c.Header, state)

	if state.SkillBar != nil {
		c.Header.SetSkillBlock(state.SkillBar.Block())
	}

	exportMerc(c, state)
	exportMercItems(c, state, warn)
	exportDeath(c, state)

	if !opts.LastPlayed.IsZero() {
		binary.LittleEndian.PutUint32(c.Header.Raw[lastPlayedOffset:], uint32(opts.LastPlayed.Unix()))
	}

	out, err := d2s.Write(c, tables)
	if err != nil {
		return nil, warnings, err
	}

	// never hand out a file the parser itself would reject
	if _, err := d2s.Parse(out, tables); err != nil {
		return nil, warnings, fmt.Errorf("exported file does not parse: %w", err)
	}

	return out, warnings, nil
}

func exportAttributes(c *d2s.Character, state *HeroState, warn func(string, ...interface{})) {
	b, s := c.Body, state.Stats
	a := &b.Attributes

	set := func(id int, cur uint64, want int, frac bool) {
		if want < 0 {
			want = 0
		}

		v := uint64(want)
		if cur == v {
			return
		}

		if frac {
			v <<= 8 // hit points, mana and stamina carry eight fractional bits
		}

		had := len(b.Stats)
		b.SetStat(id, v)

		if len(b.Stats) != had { // keep the stats in the id order the game writes them
			sort.SliceStable(b.Stats, func(i, j int) bool { return b.Stats[i].ID < b.Stats[j].ID })
		}
	}

	set(d2s.StatStrength, a.Strength, s.Strength, false)
	set(d2s.StatEnergy, a.Energy, s.Energy, false)
	set(d2s.StatDexterity, a.Dexterity, s.Dexterity, false)
	set(d2s.StatVitality, a.Vitality, s.Vitality, false)
	set(d2s.StatUnusedStats, a.UnusedStats, s.StatsPoints, false)
	set(d2s.StatUnusedSkills, a.UnusedSkillPoints, s.SkillPoints, false)
	set(d2s.StatLevel, a.Level, s.Level, false)
	set(d2s.StatExperience, a.Experience, s.Experience, false)

	// compared in whole points; the stored values carry the fraction
	set(d2s.StatCurrentHP, a.CurrentHP, s.Health, true)
	// a .d2s stores the maxima WITHOUT item bonuses (the engine's Max* are totals)
	set(d2s.StatMaxHP, a.MaxHP, storedMax(s.BaseMaxHealth, s.MaxHealth), true)
	set(d2s.StatCurrentMana, a.CurrentMana, s.Mana, true)
	set(d2s.StatMaxMana, a.MaxMana, storedMax(s.BaseMaxMana, s.MaxMana), true)
	set(d2s.StatMaxStamina, a.MaxStamina, storedMax(s.BaseMaxStamina, s.MaxStamina), true)
	// current stamina is not kept by the engine (it resets on entering the world)

	gold := state.Gold
	if gold > maxGold {
		warn("gold %d exceeds what a .d2s can hold, clamped to %d", gold, maxGold)

		gold = maxGold
	}

	set(d2s.StatGold, a.Gold, gold, false)

	if state.StashGold != nil {
		stash := *state.StashGold
		if stash > d2inventory.StashGoldLimit {
			warn("stash gold %d exceeds the stash cap, clamped to %d", stash, d2inventory.StashGoldLimit)

			stash = d2inventory.StashGoldLimit
		}

		set(d2s.StatStashedGold, a.StashedGold, stash, false)
	}

	if s.Level > 0 && s.Level < 256 {
		c.Header.Level = uint8(s.Level)
	}
}

func exportSkills(b *d2s.Body, state *HeroState, ids []int) {
	for slot, id := range ids {
		if slot >= len(b.SkillPoints) {
			break
		}

		skill := state.Skills[id]
		if skill == nil {
			continue // not imported (no description): keep the original points
		}

		points := skill.SkillPoints
		if skill.Shallow != nil {
			points = skill.Shallow.SkillPoints
		}

		if points > 0 && points < 256 {
			b.SkillPoints[slot] = byte(points)
		}
	}
}

func exportProgress(b *d2s.Body, state *HeroState) {
	p := state.Progress
	if p == nil {
		return
	}

	for d := range p.Quests {
		b.Quests[d] = p.Quests[d]
	}

	b.Waypoints = p.Waypoints
	b.NPC = p.NPC
}

// exportWorld writes the map seed and the active difficulty and act.
func exportWorld(h *d2s.Header, state *HeroState) {
	if state.MapSeed != 0 {
		h.MapSeed = state.MapSeed
	}

	diff, act, ok := h.ActiveDifficulty()
	if !ok || state.Difficulty < 0 || int(state.Difficulty) > 2 {
		return
	}

	// HeroState.Act is the act the hero is in (set from the header on import
	// and by act travel), so it is written back as is; a state without an act
	// (older hero files) keeps the act of the header.
	if state.Act >= 1 && state.Act <= 5 {
		act = state.Act - 1
	}

	if diff == int(state.Difficulty) && act == int(h.Difficulty[diff]&^0x80) {
		return
	}

	h.Difficulty = [3]byte{}
	h.Difficulty[state.Difficulty] = 0x80 | byte(act)
}

// engineSlotCodes maps a .d2s equipment slot to the code of the item the
// engine has there (only the slots ImportD2S fills).
func engineSlotCodes(state *HeroState) map[uint8]string {
	e := &state.Equipment
	codes := map[uint8]string{}

	if e.Head != nil {
		codes[d2sSlotHead] = e.Head.ItemCode
	}

	if e.Torso != nil {
		codes[d2sSlotTorso] = e.Torso.ItemCode
	}

	if e.Legs != nil {
		codes[d2sSlotFeet] = e.Legs.ItemCode
	}

	if e.RightArm != nil {
		codes[d2sSlotGloves] = e.RightArm.ItemCode
	}

	if e.RightHand != nil {
		codes[d2sSlotRightArm] = e.RightHand.GetItemCode()
	}

	if e.Shield != nil {
		codes[d2sSlotLeftArm] = e.Shield.ItemCode
	} else if e.LeftHand != nil {
		codes[d2sSlotLeftArm] = e.LeftHand.GetItemCode()
	}

	return codes
}

// checkEquipment compares the engine's equipment with the original's. Item
// bits are never fabricated: the original d2s items are always kept, and any
// difference is reported.
func checkEquipment(c *d2s.Character, state *HeroState, warn func(string, ...interface{})) {
	engine := engineSlotCodes(state)
	original := map[uint8]string{}

	for i := range c.Items {
		if it := &c.Items[i]; it.Location == d2s.LocationEquipped {
			original[it.Equipped] = strings.TrimSpace(it.Code)
		}
	}

	for slot, code := range engine {
		code = strings.TrimSpace(code)

		switch have, ok := original[slot]; {
		case !ok:
			warn("equipment slot %d: engine item %q has no .d2s counterpart, not written", slot, code)
		case have != code:
			warn("equipment slot %d: engine has %q but the .d2s has %q, keeping the .d2s item", slot, code, have)
		}
	}
}

// exportMerc writes the mercenary header fields (dead flag, id, name, type,
// experience). A hero without Merc state keeps whatever the original had. A
// merc hired over the original one drops the old merc's items (the original
// game discards them too).
func exportMerc(c *d2s.Character, state *HeroState) {
	m := state.Merc
	if m == nil || !c.Header.IsExpansion() {
		return
	}

	if m.Replaced && c.Header.Mercenary.ID != m.ID {
		c.MercItems = nil
	}

	c.Header.Mercenary = d2s.Mercenary{Dead: m.Dead, ID: m.ID, NameID: m.NameID, Type: m.Type, Experience: m.Experience}
}

// corpseHeaderX and corpseHeaderY are where the 12 corpse header bytes keep the
// corpse position. UNVERIFIED: the writer (0x5674f0) fills the second and
// third dword from two getters, the loader skips all 12 bytes.
const (
	corpseHeaderX = 4
	corpseHeaderY = 8
)

// exportDeath writes the died status and the corpse of a hero that died. The
// equipped items are moved from the item list to the corpse, as the game does
// (the corpse built by 0x57d6f0 receives the equipped items); inventory and
// stash stay. A corpse that came with the imported file is left alone.
func exportDeath(c *d2s.Character, state *HeroState) {
	d := state.Death
	if d == nil {
		return
	}

	if state.Hardcore {
		c.Header.Status |= d2s.StatusHardcore
	}

	if d.Died {
		c.Header.Status |= d2s.StatusDied
	} else {
		c.Header.Status &^= d2s.StatusDied
	}

	if d.Corpse == nil || c.Body == nil || c.HasCorpse {
		return
	}

	kept := c.Items[:0:0]

	for i := range c.Items {
		if c.Items[i].Location == d2s.LocationEquipped {
			c.Corpse = append(c.Corpse, c.Items[i])
			continue
		}

		kept = append(kept, c.Items[i])
	}

	c.Items = kept
	c.HasCorpse = true

	binary.LittleEndian.PutUint32(c.CorpseHeader[corpseHeaderX:], uint32(d.Corpse.X))
	binary.LittleEndian.PutUint32(c.CorpseHeader[corpseHeaderY:], uint32(d.Corpse.Y))
}

func questRecordsOf(b *d2s.Body) (q [3]d2s.QuestRecord) {
	for d := range q {
		q[d] = d2s.QuestRecord(b.Quests[d])
	}

	return q
}

// exportProgression raises the progression byte when the hero finished act
// bosses the original save had not recorded. It never lowers it, and an
// unchanged hero keeps the original byte.
func exportProgression(h *d2s.Header, orig [3]d2s.QuestRecord, state *HeroState) {
	now := d2difficulty.ProgressionOf(state.questRecords(), h.IsExpansion())
	if now <= d2difficulty.ProgressionOf(orig, h.IsExpansion()) {
		return
	}

	if now > d2difficulty.Progression(h.Status) {
		h.Status = d2difficulty.WithProgression(h.Status, now)
	}
}
