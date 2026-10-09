package d2hero

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// EnvD2SWriteback names a folder (OD2_D2S_WRITEBACK=<dir>) that heroes
// imported from a .d2s are saved into, as <dir>/<name>.d2s. Without it the
// export goes next to the .od2 save, so the player's original is never
// overwritten unless they opt in (by pointing this at its folder).
const EnvD2SWriteback = "OD2_D2S_WRITEBACK"

// D2SExport describes a finished export, for logging.
type D2SExport struct {
	Path     string
	Warnings []string
	// Summary is what a re-parse of the written file shows.
	Summary string
}

// D2SPath returns where SaveD2S writes the hero.
func D2SPath(state *HeroState) string {
	name := state.HeroName + ".d2s"

	if dir := os.Getenv(EnvD2SWriteback); dir != "" {
		return filepath.Join(dir, name)
	}

	return filepath.Join(filepath.Dir(state.FilePath), name)
}

// SaveD2S exports an imported hero to a real .d2s file (see D2SPath). Heroes
// without an imported original return ErrNoOriginal.
func (f *HeroStateFactory) SaveD2S(state *HeroState) (*D2SExport, error) {
	if state == nil || len(state.D2SBase) == 0 {
		return nil, ErrNoOriginal
	}

	if f.d2sTables == nil {
		tables, err := f.loadD2SItemTables()
		if err != nil {
			return nil, fmt.Errorf("item tables: %w", err)
		}

		f.d2sTables = tables
	}

	res := &D2SExport{Path: D2SPath(state)}

	data, warnings, err := ExportD2SWithOptions(state, state.D2SBase, f.d2sTables, ExportOptions{
		SkillIDs:   f.classSkillIDs(state.HeroType),
		LastPlayed: time.Now(),
	})
	res.Warnings = warnings

	if err != nil {
		return res, err
	}

	if err := os.MkdirAll(filepath.Dir(res.Path), mkdirPermission); err != nil {
		return res, err
	}

	// write beside the target and rename, so a crash never leaves half a save
	tmp := res.Path + ".tmp"
	if err := os.WriteFile(tmp, data, writefilePermission); err != nil {
		return res, err
	}

	if err := os.Rename(tmp, res.Path); err != nil {
		return res, err
	}

	res.Summary = SummarizeD2S(data, f.d2sTables)

	return res, nil
}

// SummarizeD2S re-parses a .d2s and describes it in one line.
func SummarizeD2S(data []byte, tables *d2s.ItemTables) string {
	c, err := d2s.Parse(data, tables)
	if err != nil {
		return "re-parse failed: " + err.Error()
	}

	if c.Body == nil {
		return fmt.Sprintf("name=%s class=%v level=%d status=0x%X expansion=%v hardcore=%v ladder=%v died=%v "+
			"(new character, no body) checksum=ok", c.Header.Name, c.Header.Class, c.Header.Level, c.Header.Status,
			c.Header.IsExpansion(), c.Header.IsHardcore(), c.Header.IsLadder(), c.Header.IsDead())
	}

	a := c.Body.Attributes
	diff, act, _ := c.Header.ActiveDifficulty()

	var eq []string

	for i := range c.Items {
		if c.Items[i].Location == d2s.LocationEquipped {
			eq = append(eq, strings.TrimSpace(c.Items[i].Code))
		}
	}

	merc := ""
	if m := c.Header.Mercenary; m.ID != 0 {
		merc = fmt.Sprintf("merc=type%d/name%d/exp%d/dead=%v ", m.Type, m.NameID, m.Experience, m.Dead)
	}

	bar := SkillBarFromBlock(c.Header.SkillBlock())
	spent := 0

	for _, p := range c.Body.SkillPoints {
		spent += int(p)
	}

	skills := fmt.Sprintf("skills=left%d/right%d swap=%d/%d hotkeys=%v spent=%d unused=%d ", bar.Left.Skill, bar.Right.Skill,
		bar.LeftSwap.Skill, bar.RightSwap.Skill, hotkeyIDs(bar), spent, a.UnusedSkillPoints)

	return fmt.Sprintf("name=%s class=%v level=%d exp=%d gold=%d str=%d dex=%d vit=%d ene=%d hp=%d mana=%d "+
		"difficulty=%d act=%d seed=%d items=%d equipped=%v status=0x%X hardcore=%v died=%v corpse=%d "+
		"%swaypoints=%#x %sact1quests=%s checksum=ok",
		c.Header.Name, c.Header.Class, a.Level, a.Experience, a.Gold, a.Strength, a.Dexterity, a.Vitality,
		a.Energy, a.MaxHP, a.MaxMana, diff, act+1, c.Header.MapSeed, len(c.Items), eq, c.Header.Status,
		c.Header.IsHardcore(), c.Header.IsDead(), len(c.Corpse), skills, c.Body.Waypoints[diff], merc, act1QuestSlots(c.Body, diff))
}

// act1QuestSlots lists the non-zero quest slots 0..7 (Act 1 quests and the
// Act 1 finished word) of a difficulty as slot:0xBITS, e.g. "[1:0x2001 2:0x0001]".
func act1QuestSlots(b *d2s.Body, difficulty int) string {
	rec := b.QuestRecord(difficulty)
	if rec == nil {
		return "[]"
	}

	var parts []string

	for slot := 0; slot <= d2s.QuestSlotAct1Finished; slot++ {
		if v := rec.Slot(slot); v != 0 {
			parts = append(parts, fmt.Sprintf("%d:0x%04x", slot, v))
		}
	}

	return "[" + strings.Join(parts, " ") + "]"
}
