package d2herostats

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2animspeed"

// Faster hit recovery / cast rate / block recovery / attack speed.
//
// Effective percent: the game does not apply the item percent linearly; it
// diminishes it (1.14, VERIFIED in the exe stat table, see d2animspeed):
//
//	eff = floor(120*p / (120+p))
//	frames = ceil(256*A / floor(S*(100+eff)/100))
//
// A is the action's base frame count and S the animation speed (256 normally).
// Hit and block recovery start from 50 percent instead of 100 (VERIFIED), which
// is why their tables differ from the cast formula above.
//
// STATUS: the Tables below are DERIVED at start-up from the d2animspeed rule and
// the AnimData frame counts/speeds of the class records (small embedded
// constants read from the expansion animdata.d2; no game file is committed).
// They replace the hand-copied community tables. Rows whose inputs are not
// fully established carry a non-empty Entry.Unverified. The IAS function is
// the community formula, unverified.

// Kind of speed modifier.
type Kind string

// Speed modifier kinds.
const (
	FHR Kind = "FHR" // faster hit recovery
	FCR Kind = "FCR" // faster cast rate (generic spells)
	FBR Kind = "FBR" // faster block recovery
	// IAS is the item attack speed stat (attack animation breakpoints).
	IAS Kind = "IAS"
)

// EffectivePct is the diminished percent: floor(120*p/(120+p)).
func EffectivePct(p int) int {
	if p <= 0 {
		return 0
	}

	return 120 * p / (120 + p)
}

// Frames returns ceil(256*a/floor(s*(100+eff(p))/100)). a is the base frame
// count, s the animation speed (256 normal). It is the cast-rate form.
func Frames(a, s, p int) int {
	spd := s * (100 + EffectivePct(p)) / 100
	if spd < 1 {
		spd = 1
	}

	return (256*a + spd - 1) / spd
}

// Breakpoints lists the smallest percent that reaches each successive frame
// count, starting with 0, and stops after `entries` entries (the games tables
// end where the animation hits its minimum).
func Breakpoints(a, s, entries int) []int {
	out := make([]int, 0, entries)
	cur := -1

	for p := 0; p < 100000 && len(out) < entries; p++ {
		if f := Frames(a, s, p); f != cur {
			out = append(out, p)
			cur = f
		}
	}

	return out
}

// Model is the cast-rate formula: base frames A at animation speed S.
type Model struct {
	A, S, Entries int
}

// Table returns the breakpoints the model produces.
func (m Model) Table() []int { return Breakpoints(m.A, m.S, m.Entries) }

// Anim is the AnimData record an Entry is derived from: frames per direction
// and base speed (8.8), VERIFIED numbers from the expansion animdata.d2.
type Anim struct {
	Record string // AnimData record name, e.g. "DZGH1HS"; empty when unknown
	Frames int
	Speed  int
}

// Entry is one class/kind breakpoint table. Table is DERIVED from Anim with
// the d2animspeed rule. Model is non-nil for FCR entries, where the older
// cast formula (Frames above) gives the same table (checked by a test).
type Entry struct {
	Class string
	Kind  Kind
	Note  string // animation the table applies to
	Table []int
	Model *Model
	Anim  Anim
	// Unverified is empty when the row follows from VERIFIED rules and
	// AnimData numbers only; otherwise it says what is not established.
	Unverified string
}

func action(k Kind) d2animspeed.Action {
	switch k {
	case FHR:
		return d2animspeed.ActionHit
	case FBR:
		return d2animspeed.ActionBlock
	case FCR:
		return d2animspeed.ActionCast
	default:
		return d2animspeed.ActionAttack
	}
}

// DeriveBreakpoints computes the breakpoint table of a kind for an animation of the given
// AnimData frames and speed. It is the reusable derivation: the engine can call
// it for any animation record.
func DeriveBreakpoints(k Kind, frames, speed int) []int {
	return d2animspeed.DeriveTable(action(k), frames, speed, d2animspeed.MaxTableStat)
}

// NextBreakpoint returns the next stat value above pct at which the entry's
// animation gets shorter, and how many ticks the animation takes now and at
// that breakpoint. ok is false at the last breakpoint. It is meant for the
// character panel ("next breakpoint"); it is not wired into any UI yet.
func (e Entry) NextBreakpoint(pct int) (next, ticksNow, ticksNext int, ok bool) {
	next, ok = d2animspeed.NextBreakpoint(e.Table, pct)
	if !ok {
		return 0, 0, 0, false
	}

	act := action(e.Kind)

	return next,
		d2animspeed.ActionTicks(act, e.Anim.Frames, e.Anim.Speed, pct),
		d2animspeed.ActionTicks(act, e.Anim.Frames, e.Anim.Speed, next),
		true
}

type spec struct {
	class string
	kind  Kind
	note  string
	anim  Anim
	unver string
}

const (
	unverLightning = "frame count 19 is the community number (equals records SOA11HT/DZA11HS, not SOSC); which record the exe plays for lightning is unverified"

	// Settled in wave 31 (VERIFIED in the exe, notes in w31-mech2.md), so the rows carry no flag any more:
	//   - Druid forms: states.txt gives state 139 (wolf) gfxtype 1 gfxclass 430 and state 140 (bear) gfxclass 431;
	//     monstats rows 430 / 431 are Wolf (token 40) and Bear (token TG). UNIT_GetShapeshiftAdjustedMode
	//     (0x624830 region) turns the player into a monster of that class for the animation rules, so hit and
	//     block recovery run the monster mode formulas on the 40 / TG records. The werebear row
	//     0,5,10,16,24,37,54,86,152,360 follows from TGGHHTH (5 frames at speed 184); it is NOT the Necromancer
	//     row (7 frames at 256), which was an old recollection.
	//   - Paladin block: COMPOSIT_ResolveWeaponClassCode (0x650480) never looks at the shield, the record comes
	//     from the weapon class; PABL1HS and PABLHTH both have 3 frames at 256, so "with a shield" is the 3 frame
	//     row. The old 8 frame (Sorceress) row matches no record.
	//   - Attack: stat 0x44 starts at 100 (0x56e5d0), a weapon adds -speed (0x555b10), a skill adds its own term.
	unverForm   = ""
	unverBear   = ""
	unverShield = ""
	unverIAS    = ""
)

// specs lists the animations of the seven classes plus the 1.14 per-form
// variants. There is no Necromancer vampire form in 1.14 (no animdata
// records), so no row exists for it.
var specs = []spec{
	{"Amazon", FHR, "", Anim{"AMGHHTH", 6, 256}, ""},
	{"Sorceress", FHR, "", Anim{"SOGHHTH", 8, 256}, ""},
	{"Necromancer", FHR, "", Anim{"NEGHHTH", 7, 256}, ""},
	{"Paladin", FHR, "", Anim{"PAGHHTH", 5, 256}, ""},
	{"Barbarian", FHR, "", Anim{"BAGHHTH", 5, 256}, ""},
	// Druid human: the 1HS weapon class record runs at speed 248, which gives
	// the old (previously unexplained) community row exactly; the other weapon
	// classes and bare hands run at 256 and give the Necromancer row.
	{"Druid", FHR, "human form, one-handed swing weapon", Anim{"DZGH1HS", 7, 248}, ""},
	{"Druid", FHR, "human form, other weapon classes", Anim{"DZGHHTH", 7, 256}, ""},
	{"Druid", FHR, "werewolf", Anim{"40GHHTH", 4, 256}, unverForm},
	{"Druid", FHR, "werebear", Anim{"TGGHHTH", 5, 184}, unverBear},
	{"Assassin", FHR, "", Anim{"AIGHHTH", 5, 256}, ""},

	{"Amazon", FCR, "", Anim{"AMSCHTH", 20, 256}, ""},
	{"Sorceress", FCR, "", Anim{"SOSCHTH", 14, 256}, ""},
	{"Necromancer", FCR, "", Anim{"NESCHTH", 16, 256}, ""},
	{"Paladin", FCR, "", Anim{"PASCHTH", 16, 256}, ""},
	{"Barbarian", FCR, "", Anim{"BASCHTH", 14, 256}, ""},
	{"Druid", FCR, "", Anim{"DZSCHTH", 15, 208}, ""},
	{"Assassin", FCR, "", Anim{"AISCHTH", 17, 256}, ""},
	{"Sorceress", FCR, "lightning/chain lightning", Anim{"", 19, 256}, unverLightning},

	{"Amazon", FBR, "", Anim{"AMBLHTH", 3, 256}, ""},
	{"Sorceress", FBR, "", Anim{"SOBLHTH", 5, 256}, ""},
	{"Necromancer", FBR, "", Anim{"NEBLHTH", 6, 256}, ""},
	{"Paladin", FBR, "with a shield", Anim{"PABL1HS", 3, 256}, unverShield},
	{"Paladin", FBR, "without a shield", Anim{"PABLHTH", 3, 256}, ""},
	{"Barbarian", FBR, "", Anim{"BABLHTH", 4, 256}, ""},
	{"Druid", FBR, "human form", Anim{"DZBLHTH", 6, 256}, ""},
	{"Druid", FBR, "werewolf", Anim{"40BLHTH", 5, 256}, unverForm},
	{"Druid", FBR, "werebear", Anim{"TGBLHTH", 5, 200}, unverForm},
	{"Assassin", FBR, "", Anim{"AIBLHTH", 3, 256}, ""},

	{"Assassin", IAS, "martial arts, bare hands", Anim{"AIA1HTH", 11, 256}, unverIAS},
	{"Assassin", IAS, "claws (hand-to-hand weapon class)", Anim{"AIA1HT1", 11, 208}, unverIAS},
}

func buildTables() []Entry {
	out := make([]Entry, 0, len(specs))

	for _, s := range specs {
		e := Entry{Class: s.class, Kind: s.kind, Note: s.note, Anim: s.anim, Unverified: s.unver}
		e.Table = DeriveBreakpoints(s.kind, s.anim.Frames, s.anim.Speed)

		if s.kind == FCR {
			e.Model = &Model{s.anim.Frames, s.anim.Speed, len(e.Table)}
		}

		out = append(out, e)
	}

	return out
}

// Tables are the breakpoint tables per class, derived from the d2animspeed
// rule and the class animation frame counts. The first table of a class and
// kind is the default animation (Lookup).
var Tables = buildTables()

// Lookup returns the first table for the class and kind (the default
// animation); ok is false when there is none.
func Lookup(class string, kind Kind) (Entry, bool) {
	for _, e := range Tables {
		if e.Class == class && e.Kind == kind {
			return e, true
		}
	}

	return Entry{}, false
}

// Reached returns the index of the highest breakpoint of the table that the
// given percent reaches (0 for none), i.e. how many frames were saved.
func Reached(table []int, pct int) int {
	idx := 0

	for i, bp := range table {
		if pct >= bp {
			idx = i
		}
	}

	return idx
}

// AttackFrames is the length in game frames (25 per second) of an attack animation of a baseFrames long record at
// animation speed 256, with item IAS ias, weapon speed modifier wsm (weapons.txt "speed", negative = fast) and the
// skill's own speed term skillIAS. It follows the exe (PATH_UpdateUnitVelocityAndAnimRate 0x624150 and
// EVENT_SchedulePathStepEvents 0x551a20, VERIFIED, see d2animspeed.AttackTicks): the percent is
// 100 - wsm + skillIAS + floor(120*ias/(120+ias)) clamped to 15..175, the rate floor(256*percent/100), and the
// result ceil(256*baseFrames/rate), at least 2. The community tables list this number minus one; the old function
// subtracted it here, the engine does not.
func AttackFrames(baseFrames, ias, wsm, skillIAS int) int {
	return d2animspeed.AttackTicks(d2animspeed.Mode{Frames: baseFrames, Speed: 256}, 0,
		d2animspeed.AttackStat44(wsm, skillIAS), ias)
}
