package d2object

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Shrine codes of shrines.txt (the "Code" column; also the index of the
// effect function table at 0x6e2b48 in Game.exe, VERIFIED).
const (
	ShrineNone = iota
	ShrineRefill
	ShrineHealth
	ShrineMana
	ShrineHealthExchange
	ShrineManaExchange
	ShrineArmor
	ShrineCombat
	ShrineResistFire
	ShrineResistCold
	ShrineResistLightning
	ShrineResistPoison
	ShrineSkill
	ShrineManaRecharge
	ShrineStamina
	ShrineExperience
	ShrineEnirhs
	ShrinePortal
	ShrineGem
	ShrineStorm
	ShrineWarping
	ShrineExploding
	ShrinePoison
	numShrines
)

// Shrine is one row of shrines.txt.
type Shrine struct {
	Code          int
	Type          string // Recharge, Booster, Magic
	Name          string
	Effect        string // the description column
	Arg0, Arg1    int
	DurationFrame int // "Duration in frames", 0 for instant shrines
	ResetMinutes  int
	Rarity        int
	EffectClass   int // 1 magic, 2 health, 3 mana, 4 refill and boosters
	LevelMin      int
	// Out marks rows whose description says "(OUT)" (cut features; the
	// binary still has an effect function for them, whether they spawn is
	// UNVERIFIED).
	Out bool
	// Sound is the Sounds.txt handle played when the shrine is used
	// (ESOUND_SHRINE_*; which shrine plays which is UNVERIFIED but the names
	// are one per effect).
	Sound string
}

// DurationSeconds is the length of the shrine state in seconds.
func (s Shrine) DurationSeconds() float64 { return float64(s.DurationFrame) / FramesPerSecond }

// ResetFrames is the delay after which a used shrine can be used again:
// VERIFIED in the shrine operate function (minutes * 0x4b0 + 1 frames).
func (s Shrine) ResetFrames() int {
	if s.ResetMinutes <= 0 {
		return 0
	}

	return s.ResetMinutes*0x4b0 + 1
}

// ResetSeconds is ResetFrames at FramesPerSecond. 0x4b0 frames per "minute" is
// 48 s at 25 fps; whether the original frame rate of these timers is 25 or 20
// is UNVERIFIED.
func (s Shrine) ResetSeconds() float64 { return float64(s.ResetFrames()) / FramesPerSecond }

var shrineSounds = map[int]string{
	ShrineRefill: "shrine_refill", ShrineHealth: "shrine_recharge", ShrineMana: "shrine_recharge",
	ShrineHealthExchange: "shrine_exchange", ShrineManaExchange: "shrine_exchange",
	ShrineArmor: "shrine_armorboost", ShrineCombat: "shrine_combatboost",
	ShrineResistFire: "shrine_resistfire", ShrineResistCold: "shrine_resistcold",
	ShrineResistLightning: "shrine_resistlightning", ShrineResistPoison: "shrine_resistpoison",
	ShrineSkill: "shrine_skill", ShrineManaRecharge: "shrine_recharge", ShrineStamina: "shrine_recharge",
	ShrineExperience: "shrine_experience", ShrineEnirhs: "shrine_ofenirhs", ShrinePortal: "shrine_portal",
	ShrineGem: "shrine_gemupgrade", ShrineStorm: "shrine_storm", ShrineWarping: "shrine_warping",
	ShrineExploding: "shrine_exploding", ShrinePoison: "shrine_poison",
}

// DefaultShrines is shrines.txt of the 1.14b expansion data (d2exp.mpq),
// transcribed so the engine does not need to parse the table; the optional
// real-data test compares it with the file. Row 0 ("None") is kept so that
// the slice index is the shrine code.
var DefaultShrines = []Shrine{
	{Code: 0, Type: "None", Name: "None", Effect: "none", Arg0: 100, Arg1: 100, ResetMinutes: 2, Rarity: 1},
	{Code: 1, Type: "Recharge", Name: "Refill", Effect: "fills health and mana", Arg0: 100, Arg1: 100,
		ResetMinutes: 2, Rarity: 1, EffectClass: 4, LevelMin: 1},
	{Code: 2, Type: "Recharge", Name: "Health Boost", Effect: "doubles current health (even over maximum) (OUT)",
		Arg0: 200, Arg1: 400, ResetMinutes: 5, Rarity: 2, EffectClass: 2, LevelMin: 1, Out: true},
	{Code: 3, Type: "Recharge", Name: "Mana Boost", Effect: "doubles current mana (even over maximum) (OUT)",
		Arg0: 200, Arg1: 400, ResetMinutes: 5, Rarity: 2, EffectClass: 3, LevelMin: 1, Out: true},
	{Code: 4, Type: "Recharge", Name: "Health Exchange",
		Effect: "takes 1/2 health, gives 5 times that number to current mana (OUT)",
		Arg0:   50, Arg1: 500, Rarity: 3, EffectClass: 2, LevelMin: 2, Out: true},
	{Code: 5, Type: "Recharge", Name: "Mana Exchange",
		Effect: "takes 1/2 mana, gives 5 times that number to current health (OUT)",
		Arg0:   50, Arg1: 500, Rarity: 3, EffectClass: 3, LevelMin: 2, Out: true},
	{Code: 6, Type: "Booster", Name: "Armor Boost", Effect: "+100% AC", Arg0: 100, Arg1: 1,
		DurationFrame: 2400, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 5},
	{Code: 7, Type: "Booster", Name: "Combat Boost", Effect: "+200% to hit, +200% min and max damage",
		Arg0: 200, Arg1: 200, DurationFrame: 2400, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 8},
	{Code: 8, Type: "Booster", Name: "Resist Fire Boost", Effect: "+75% to resist fire", Arg0: 75,
		DurationFrame: 3600, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 5},
	{Code: 9, Type: "Booster", Name: "Resist Cold Boost", Effect: "+75% to resist cold", Arg0: 75,
		DurationFrame: 3600, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 26},
	{Code: 10, Type: "Booster", Name: "Resist Lightning Boost", Effect: "+75% to resist lightning", Arg0: 75,
		DurationFrame: 3600, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 32},
	{Code: 11, Type: "Booster", Name: "Resist Poison Boost", Effect: "+75% to resist poison, and poison duration = 0",
		Arg0: 75, DurationFrame: 3600, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 21},
	{Code: 12, Type: "Booster", Name: "Skill Boost", Effect: "+2 to all skill levels", Arg0: 2,
		DurationFrame: 2400, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 1},
	{Code: 13, Type: "Booster", Name: "Recharge Boost", Effect: "+400% mana recharge rate", Arg0: 400,
		DurationFrame: 2400, ResetMinutes: 5, Rarity: 2, EffectClass: 4, LevelMin: 1},
	{Code: 14, Type: "Booster", Name: "Stamina Boost", Effect: "Unlimited stamina", Arg0: 200,
		DurationFrame: 4800, ResetMinutes: 5, Rarity: 3, EffectClass: 4, LevelMin: 1},
	{Code: 15, Type: "Booster", Name: "Experience Boost", Effect: "50% more experience per kill",
		DurationFrame: 3600, Rarity: 3, EffectClass: 4, LevelMin: 1},
	{Code: 16, Type: "Magic", Name: "Shrine of Enirhs", Effect: "reverses character name (OUT)",
		Rarity: 3, EffectClass: 1, LevelMin: 1, Out: true},
	{Code: 17, Type: "Magic", Name: "Portal to unknown", Effect: "opens a portal to town",
		Rarity: 2, EffectClass: 1, LevelMin: 3},
	{Code: 18, Type: "Magic", Name: "Gem Upgrade",
		Effect: "upgrades a random gem (up to one before max level if available), or gives a random gem",
		Rarity: 3, EffectClass: 1, LevelMin: 4},
	{Code: 19, Type: "Magic", Name: "Storm Shrine",
		Effect: "all players and monsters lose 1/2 current HP, shoots fireballs",
		Arg0:   50, Arg1: 2000, Rarity: 2, EffectClass: 1, LevelMin: 1},
	{Code: 20, Type: "Magic", Name: "Warping Shrine", Effect: "nearest monster becomes unique",
		Rarity: 3, EffectClass: 1, LevelMin: 3},
	{Code: 21, Type: "Magic", Name: "Exploding Shrine",
		Effect: "drops 5-10 exploding potions & tosses out exploding potions in random directions/distances",
		Arg0:   5, Arg1: 10, Rarity: 3, EffectClass: 1, LevelMin: 1},
	{Code: 22, Type: "Magic", Name: "Poison Shrine",
		Effect: "drops 5-10 poison gas potions & creates ring of poison gas around shrine",
		Arg0:   5, Arg1: 10, Rarity: 3, EffectClass: 1, LevelMin: 1},
}

func init() {
	for i := range DefaultShrines {
		DefaultShrines[i].Sound = shrineSounds[DefaultShrines[i].Code]
	}
}

// ShrineByCode returns the row of a shrine code.
func ShrineByCode(rows []Shrine, code int) (Shrine, bool) {
	for _, r := range rows {
		if r.Code == code {
			return r, true
		}
	}

	return Shrine{}, false
}

// ParseShrines reads shrines.txt (tab separated, header row).
func ParseShrines(data []byte) ([]Shrine, error) {
	lines := strings.Split(strings.ReplaceAll(string(bytes.TrimSpace(data)), "\r", ""), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("shrines.txt: no rows")
	}

	col := map[string]int{}
	for i, h := range strings.Split(lines[0], "\t") {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for _, need := range []string{"code", "arg0", "arg1", "duration in frames", "reset time in minutes", "rarity", "effectclass", "levelmin"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("shrines.txt: missing column %q", need)
		}
	}

	out := []Shrine{}

	for _, ln := range lines[1:] {
		f := strings.Split(ln, "\t")
		get := func(name string) string {
			if i := col[name]; i < len(f) {
				return strings.Trim(strings.TrimSpace(f[i]), `"`)
			}

			return ""
		}
		num := func(name string) int { n, _ := strconv.Atoi(get(name)); return n }

		s := Shrine{
			Code: num("code"), Type: get("shrine type"), Name: get("shrine name"), Effect: get("effect"),
			Arg0: num("arg0"), Arg1: num("arg1"), DurationFrame: num("duration in frames"),
			ResetMinutes: num("reset time in minutes"), Rarity: num("rarity"), EffectClass: num("effectclass"),
			LevelMin: num("levelmin"),
		}
		s.Out = strings.Contains(s.Effect, "(OUT)")
		s.Sound = shrineSounds[s.Code]
		out = append(out, s)
	}

	return out, nil
}

// Roller is the small deterministic generator of the rolls (the original
// uses its LCG {seed, 0x29A}; this package only needs reproducibility).
type Roller uint64

func (s *Roller) next() uint64 {
	*s += 0x9e3779b97f4a7c15
	z := uint64(*s)
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb

	return z ^ (z >> 31)
}

// Roll returns a number in [0, n).
func (s *Roller) Roll(n int) int {
	if n <= 1 {
		return 0
	}

	return int(s.next() % uint64(n))
}

// NewRoller returns a deterministic generator for shrine and well rolls.
func NewRoller(seed uint32) *Roller { r := Roller(seed) + 1; return &r }

// effectClassesFor maps the objects.txt Parm0 of a shrine object (1, 2 or 3)
// to the shrines.txt EffectClass values it can show. UNVERIFIED hypothesis:
// the manashrine rows (249, 302) have Parm0 2 and the visual classes seem to
// be 1 magic, 2 recharge (health/mana), 3 booster/refill.
func effectClassesFor(parm0 int) []int {
	switch parm0 {
	case 1:
		return []int{1}
	case 2:
		return []int{2, 3}
	case 3:
		return []int{4}
	}

	return nil
}

// RollShrine picks the shrine type of a new shrine object: among the rows
// whose LevelMin does not exceed the area level (and whose EffectClass fits
// the object's Parm0), weighted by 4-Rarity (rarity 1 is the most common). The
// original's weighting is UNVERIFIED. Rows marked Out are skipped unless
// includeOut is set. It returns false when nothing qualifies.
func RollShrine(rows []Shrine, r *Roller, areaLevel, parm0 int, includeOut bool) (Shrine, bool) {
	classes := effectClassesFor(parm0)

	pick := func(filterClass bool) (Shrine, bool) {
		total := 0

		var cand []Shrine

		for _, s := range rows {
			if s.Code == ShrineNone || s.LevelMin > areaLevel || (s.Out && !includeOut) {
				continue
			}

			if filterClass && !containsInt(classes, s.EffectClass) {
				continue
			}

			cand = append(cand, s)
			total += weight(s)
		}

		if len(cand) == 0 {
			return Shrine{}, false
		}

		n := r.Roll(total)
		for _, s := range cand {
			if n < weight(s) {
				return s, true
			}

			n -= weight(s)
		}

		return cand[len(cand)-1], true
	}

	if len(classes) > 0 {
		if s, ok := pick(true); ok {
			return s, true
		}
	}

	return pick(false)
}

func weight(s Shrine) int {
	if w := 4 - s.Rarity; w > 0 {
		return w
	}

	return 1
}

func containsInt(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}

	return false
}

// ShrineRetries is the number of rerolls the exe allows when a rolled shrine's LevelMin exceeds the
// level (VERIFIED in OBJECT_ChooseShrineType 0x54d840 / OBJECT_RollShrineForLevel 0x54d5c0: a counter
// starts at 8 and the loop runs while level < LevelMin and --counter > 0). After the last try the
// roll is kept even if the level is too low.
const ShrineRetries = 8

// RemapShrineCode applies the final fix-up of OBJECT_ChooseShrineType (VERIFIED, 0x54d840): the rolled
// code 5 (Mana Exchange) becomes 3 (Mana Boost), 4 (Health Exchange) becomes 2 (Health Boost) and 0x10
// (Enirhs) becomes 0x12 (Gem Upgrade). These are the "(OUT)" features of shrines.txt.
func RemapShrineCode(code int) int {
	switch code {
	case ShrineManaExchange:
		return ShrineMana
	case ShrineHealthExchange:
		return ShrineHealth
	case ShrineEnirhs:
		return ShrineGem
	}

	return code
}

// RollShrineUniform is the roll of a shrine object whose Parm0 is 0 (VERIFIED, 0x54d840): a uniform
// pick over the rows 1..len(rows)-1 (rarity is not used), rerolled up to ShrineRetries times while the
// row's LevelMin exceeds areaLevel, then RemapShrineCode. rows is indexed by shrine code.
func RollShrineUniform(rows []Shrine, r *Roller, areaLevel int) (Shrine, bool) {
	if len(rows) < 2 {
		return Shrine{}, false
	}

	code := 1

	for tries := ShrineRetries; ; {
		code = 1 + r.Roll(len(rows)-1)
		if areaLevel >= rows[code].LevelMin {
			break
		}

		if tries--; tries <= 0 {
			break
		}
	}

	return ShrineByCode(rows, RemapShrineCode(code))
}
