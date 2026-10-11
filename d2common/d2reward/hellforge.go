package d2reward

// The Hellforge reward shower (Game.exe 0x5b42d0, table at 0x73b624).
//
// After the forge is smashed its anvil drops items in pulses. Every pulse
// drops Count items (the quest completion count the exe keeps at quest data
// +0x14), each one a gem or skull chosen from the table of the current stage
// by the quest-state generator modulo 7. After a pulse that produced at least
// one item the stage falls by one; a pulse that produced nothing ends the
// shower silently (the exe schedules no further pulse). When the stage reaches
// zero the shower is over and, in expansion games, one rune is dropped from the
// list of the difficulty, chosen by a roll of 11. Otherwise the next pulse is
// scheduled HellforgeDelayFrames later.
//
// UNVERIFIED: where the stage and count fields are first set (the start stage
// 4 and the count are taken from the quest completion; the object-mode handler
// that begins the shower was not located). The unit animation mode 4 the exe
// selects when the +4 flag byte is set is a visual detail left out.

// HellforgeDelayFrames is the delay between two pulses (0x14 frames).
const HellforgeDelayFrames = 0x14

// HellforgeStartStage is the first stage of a shower (UNVERIFIED, see above).
const HellforgeStartStage = 4

// Gem and skull codes by quality (seven each, VERIFIED from the table bytes).
var (
	hellforgePerfect  = [7]string{"gpv", "gpr", "gpb", "gpy", "gpg", "gpw", "skz"}
	hellforgeFlawless = [7]string{"gzv", "glr", "glb", "gly", "glg", "glw", "skl"}
	hellforgeStandard = [7]string{"gsv", "gsr", "gsb", "gsy", "gsg", "gsw", "sku"}
)

// hellforgeRunes are the end runes per difficulty (normal, nightmare, hell).
var hellforgeRunes = [3][11]string{
	{"r01", "r02", "r03", "r04", "r05", "r06", "r07", "r08", "r09", "r10", "r11"},
	{"r12", "r13", "r14", "r15", "r16", "r17", "r18", "r19", "r20", "r21", "r22"},
	{"r15", "r16", "r17", "r18", "r19", "r20", "r21", "r22", "r23", "r24", "r25"},
}

// HellforgeTable returns the seven item codes the shower can drop at a stage,
// nil when the stage has none (0 and above 4).
func HellforgeTable(stage int) []string {
	switch stage {
	case 1:
		return hellforgeStandard[:]
	case 2, 3:
		return hellforgeFlawless[:]
	case 4:
		return hellforgePerfect[:]
	}

	return nil
}

// HellforgeRunes returns the 11 end runes of a difficulty (0 normal, 1
// nightmare, 2 hell; anything else counts as normal, like the exe).
func HellforgeRunes(difficulty int) []string {
	if difficulty < 0 || difficulty > 2 {
		difficulty = 0
	}

	return hellforgeRunes[difficulty][:]
}

// Shower is the state of one Hellforge shower.
type Shower struct {
	Active     bool
	Stage      int
	Count      int
	Expansion  bool
	Difficulty int
}

// NewShower starts a shower for a quest completion count.
func NewShower(count int, expansion bool, difficulty int) *Shower {
	return &Shower{Active: true, Stage: HellforgeStartStage, Count: count, Expansion: expansion, Difficulty: difficulty}
}

// PulseResult is what one pulse did.
type PulseResult struct {
	Dropped []string // items that were created, in order
	Rune    string   // the end rune ("" when none)
	Next    int      // frames until the next pulse, 0: no further pulse
}

// Pulse runs one pulse. step is the quest-state generator (one step per
// gem), roll(n) its bounded roll for the rune; drop creates an item and
// reports whether it succeeded.
func (s *Shower) Pulse(step func() uint32, roll func(n int32) uint32, drop func(code string) bool) PulseResult {
	var res PulseResult

	if !s.Active || s.Count <= 0 {
		return res
	}

	table := HellforgeTable(s.Stage)
	if table == nil {
		return res // the exe leaves without changing anything
	}

	for i := 0; i < s.Count; i++ {
		code := table[step()%7]
		if drop(code) {
			res.Dropped = append(res.Dropped, code)
		}
	}

	if len(res.Dropped) == 0 {
		return res
	}

	s.Stage--
	if s.Stage >= 1 {
		res.Next = HellforgeDelayFrames

		return res
	}

	s.Active = false

	if s.Expansion {
		code := HellforgeRunes(s.Difficulty)[roll(11)]
		if drop(code) {
			res.Rune = code
		}
	}

	return res
}
