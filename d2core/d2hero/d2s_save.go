package d2hero

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// EnvD2SWriteback names a folder (OD2_D2S_WRITEBACK=<dir>) that heroes are
// saved into, as <dir>/<name>.d2s, instead of their real .d2s. Tests and the
// automation use it so they never touch a player's saves.
const EnvD2SWriteback = "OD2_D2S_WRITEBACK"

// BackupSuffix is appended to a .d2s name for the copy of the original that
// SaveD2S makes before it first replaces the file ("Name.d2s.bak").
const BackupSuffix = ".bak"

// D2SExport describes a finished export, for logging.
type D2SExport struct {
	Path     string
	Warnings []string
	// Summary is what a re-parse of the written file shows.
	Summary string
	// NewItems has one line per item made in the game (picked up, bought) that
	// the hero holds, saying whether it is in the re-parsed file and as what.
	NewItems []string
}

// NewItemsInD2S re-parses an exported .d2s and looks up every container item of
// the hero that was made in the game (no imported original).
func NewItemsInD2S(state *HeroState, data []byte, tables *d2s.ItemTables) []string {
	if state.Containers == nil {
		return nil
	}

	c, err := d2s.Parse(data, tables)
	if err != nil {
		return []string{"re-parse failed: " + err.Error()}
	}

	var out []string

	for i := range state.Containers.Items {
		s := &state.Containers.Items[i]
		if s.D2S != nil {
			continue
		}

		line := fmt.Sprintf("code=%s page=%d x=%d y=%d", s.Code, s.Page, s.X, s.Y)
		found := false

		for j := range c.Items {
			it := &c.Items[j]
			if trimCode(it.Code) == s.Code && int(it.X) == s.X && int(it.Y) == s.Y && it.Location != d2s.LocationEquipped {
				line += fmt.Sprintf(" found=true quality=%d ilvl=%d props=%d simple=%v ethereal=%v", it.Quality, it.Level,
					len(it.Properties), it.Simple, it.Ethereal)
				found = true

				break
			}
		}

		if !found {
			line += " found=false"
		}

		out = append(out, line)
	}

	return out
}

// D2SPath returns where SaveD2S writes the hero: OD2_D2S_WRITEBACK when set,
// else the .d2s the hero was imported from or created as, else (heroes that
// have neither) next to the .od2 save.
func D2SPath(state *HeroState) string {
	name := state.HeroName + ".d2s"

	if dir := os.Getenv(EnvD2SWriteback); dir != "" {
		return filepath.Join(dir, name)
	}

	if state.Imported != nil && state.Imported.Source != "" {
		return state.Imported.Source
	}

	return filepath.Join(filepath.Dir(state.FilePath), name)
}

// newD2SPath is where the .d2s of a character that is being created goes:
// OD2_D2S_WRITEBACK, else the player's first save folder (the one the
// character list is read from), else next to the .od2 save.
func newD2SPath(state *HeroState) string {
	if os.Getenv(EnvD2SWriteback) == "" {
		if dirs := ImportDirs(); len(dirs) > 0 {
			return filepath.Join(dirs[0], state.HeroName+".d2s")
		}
	}

	return D2SPath(state)
}

// writeFileAtomic writes data to path through a temporary file in the same
// folder (synced, then renamed over the target), so a crash or a full disk
// never leaves half a save, and a reader (another game process loading the
// same hero: a host and a joiner, parallel test runs) sees the old or the new
// file, never an empty or half-written one. The temp name is unique, so two
// writers of the same path do not interleave their data.
func writeFileAtomic(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(filepath.Clean(path)), "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}

	tmp := file.Name()

	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}

	if cerr := file.Close(); err == nil {
		err = cerr
	}

	if err == nil {
		err = os.Chmod(tmp, writefilePermission)
	}

	if err == nil {
		err = renameReplace(tmp, path)
	}

	if err != nil {
		_ = os.Remove(tmp)
	}

	return err
}

// backupOriginal copies an existing file to path+BackupSuffix unless a backup
// is there already, so the first backup stays the untouched original.
func backupOriginal(path string) error {
	bak := path + BackupSuffix

	if _, err := os.Stat(bak); err == nil {
		return nil
	}

	orig, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	return writeFileAtomic(bak, orig)
}

// SaveD2S exports an imported hero to a real .d2s file (see D2SPath). Heroes
// without an imported original return ErrNoOriginal.
func (f *HeroStateFactory) SaveD2S(state *HeroState) (*D2SExport, error) {
	if state == nil || len(state.D2SBase) == 0 {
		return nil, ErrNoOriginal
	}

	// a hero that did not load completely would be written with parts missing
	if state.LoadProblem != "" {
		return &D2SExport{Path: D2SPath(state)}, fmt.Errorf("%w: %s", ErrIncompleteLoad, state.LoadProblem)
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
		Affixes:    f.affixIDs(),
		Known:      func(code string) bool { return f.asset.Records.Item.All[code] != nil },
		LastPlayed: time.Now(),
	})
	res.Warnings = warnings

	if err != nil {
		return res, err
	}

	// the one writer: validated, locked, last-writer checked, backed up, atomic
	sum, err := CommitD2S(res.Path, data, state.knownSum(res.Path))
	if err != nil {
		return res, err
	}

	state.d2sDiskPath, state.d2sDiskSum = res.Path, sum

	res.Summary = SummarizeD2S(data, f.d2sTables)
	res.NewItems = NewItemsInD2S(state, data, f.d2sTables)

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

	// the ears a hero carries (a hardcore PvP kill leaves one): name/level/class
	var ears []string

	for i := range c.Items {
		if e := c.Items[i].EarInfo; c.Items[i].Ear && e != nil {
			ears = append(ears, fmt.Sprintf("%s/L%d/c%d", e.Name, e.Level, e.Class))
		}
	}

	if len(ears) > 0 {
		merc += fmt.Sprintf("ears=%v ", ears)
	}

	bar := SkillBarFromBlock(c.Header.SkillBlock())
	spent := 0

	for _, p := range c.Body.SkillPoints {
		spent += int(p)
	}

	skills := fmt.Sprintf("skills=left%d/right%d swap=%d/%d hotkeys=%v spent=%d unused=%d ", bar.Left.Skill, bar.Right.Skill,
		bar.LeftSwap.Skill, bar.RightSwap.Skill, hotkeyIDs(bar), spent, a.UnusedSkillPoints)

	skills += fmt.Sprintf("stashgold=%d belt=%d %s %s", a.StashedGold, countAt(c.Items, d2s.LocationBelt), itemFingerprint(c),
		allDifficultiesSum(c.Body))

	return fmt.Sprintf("name=%s class=%v level=%d exp=%d gold=%d str=%d dex=%d vit=%d ene=%d hp=%d mana=%d "+
		"difficulty=%d act=%d seed=%d items=%d equipped=%v status=0x%X hardcore=%v died=%v corpse=%d "+
		"%swaypoints=%#x %sact1quests=%s laterquests=%s checksum=ok",
		c.Header.Name, c.Header.Class, a.Level, a.Experience, a.Gold, a.Strength, a.Dexterity, a.Vitality,
		a.Energy, a.MaxHP, a.MaxMana, diff, act+1, c.Header.MapSeed, len(c.Items), eq, c.Header.Status,
		c.Header.IsHardcore(), c.Header.IsDead(), len(c.Corpse), skills, c.Body.Waypoints[diff], merc, act1QuestSlots(c.Body, diff), laterQuestSlots(c.Body, diff))
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

// laterQuestSlots lists the non-zero quest slots 8..40 (Acts 2-5) of a
// difficulty as slot:0xBITS, like act1QuestSlots.
func laterQuestSlots(b *d2s.Body, difficulty int) string {
	rec := b.QuestRecord(difficulty)
	if rec == nil {
		return "[]"
	}

	var parts []string

	for slot := d2s.QuestSlotAct1Finished + 1; slot <= 40; slot++ {
		if v := rec.Slot(slot); v != 0 {
			parts = append(parts, fmt.Sprintf("%d:0x%04x", slot, v))
		}
	}

	return "[" + strings.Join(parts, " ") + "]"
}

// renameReplace renames tmp over path. On Windows the replace fails with "access denied" for a moment while another
// process (a second game, an antivirus scan, the file indexer) holds the target open, so it retries a few times there.
// Other systems rename once.
func renameReplace(tmp, path string) error {
	err := os.Rename(tmp, path)
	if runtime.GOOS != "windows" {
		return err
	}

	for i := 0; err != nil && i < 10; i++ {
		time.Sleep(20 * time.Millisecond)

		err = os.Rename(tmp, path)
	}

	return err
}

func countAt(items []d2s.Item, location uint8) int {
	n := 0

	for i := range items {
		if items[i].Location == location {
			n++
		}
	}

	return n
}

// itemFingerprint is "itemsum=<hex>" over everything the file says about the items of the hero, the corpse and the
// mercenary (code, place, quality, level, durability, quantity, sockets, properties, children): two saves with the same
// fingerprint hold the same items in the same places. The order of the items does not matter.
func itemFingerprint(c *d2s.Character) string {
	var lines []string

	var add func(tag string, it *d2s.Item)

	add = func(tag string, it *d2s.Item) {
		props := make([]string, 0, len(it.Properties))
		for _, p := range it.Properties {
			props = append(props, fmt.Sprintf("%d/%d/%d", p.ID, p.Param, p.Value))
		}

		lines = append(lines, fmt.Sprintf("%s|%s|%x|%d|%d|%d|%d|%d|%d|q%d|l%d|d%d/%d|n%d|s%d|%v|f%x|%s", tag, it.Code, it.ID,
			it.Location, it.Equipped, it.Page, it.X, it.Y, it.Version, it.Quality, it.Level, it.Durability, it.MaxDurability,
			it.Quantity, it.TotalSockets, it.Ethereal, it.Flags, strings.Join(props, ",")))

		for i := range it.Children {
			add(tag+"+", &it.Children[i])
		}
	}

	for i := range c.Items {
		add("i", &c.Items[i])
	}

	for i := range c.Corpse {
		add("c", &c.Corpse[i])
	}

	for i := range c.MercItems {
		add("m", &c.MercItems[i])
	}

	sort.Strings(lines)

	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))

	return fmt.Sprintf("itemsum=%x", sum[:5])
}

// allDifficultiesSum is "alldiff=n/h/e" with, per difficulty, the number of set quest bits and the waypoint mask,
// and a short hash of the three quest records: a hero that played Normal, Nightmare and Hell must keep all three.
func allDifficultiesSum(b *d2s.Body) string {
	h := sha256.New()
	parts := make([]string, 0, 3)

	for d := 0; d < 3; d++ {
		bits := 0
		rec := b.QuestRecord(d)

		for slot := 0; slot < d2s.QuestSlots; slot++ {
			v := rec.Slot(slot)
			bits += bitsSet(v)
			fmt.Fprintf(h, "%d:%x,", slot, v)
		}

		fmt.Fprintf(h, "w%x;", b.Waypoints[d])
		parts = append(parts, fmt.Sprintf("q%d/w%d", bits, bitsSet64(b.Waypoints[d])))
	}

	return fmt.Sprintf("alldiff=%s/%x", strings.Join(parts, ","), h.Sum(nil)[:4])
}

func bitsSet(v uint16) int { return bitsSet64(uint64(v)) }

func bitsSet64(v uint64) int {
	n := 0

	for ; v != 0; v &= v - 1 {
		n++
	}

	return n
}
