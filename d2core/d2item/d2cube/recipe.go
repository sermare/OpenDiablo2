package d2cube

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// QualityTempered is the quality of tempered items (recipes "tmp").
const QualityTempered d2drop.Quality = 9

// Input is one input column of a recipe: an item (code, type, "any" or a
// unique's name) with the modifiers the item must satisfy.
type Input struct {
	Raw    string
	Code   string // a base item code ...
	Type   string // ... or an item type (a base item of the type or a descendant)
	Any    bool   // "any"
	Unique string // a unique item by name ("The Stone of Jordan")
	Qty    int    // how many (default 1)

	Quality   d2drop.Quality // 0 = any quality
	Ethereal  triState       // eth / noe
	Socketed  triState       // sock / nos
	Tier      Tier           // bas / exc / eli, TierNone = any
	Upgrade   bool           // upg: the item has a higher tier to upgrade to
	IsGemTier bool           // the token was a gem tier type (gem0 ... gem4)
}

type triState int

const (
	any3 triState = iota
	yes3
	no3
)

// Mod is a property line of a recipe output: property code, chance in percent
// (100 when blank), parameter and value range.
type Mod struct {
	Code   string
	Chance int
	Param  string
	Min    int
	Max    int
}

// Output is one of the three results (output, output b, output c) of a recipe.
type Output struct {
	Raw string

	Code    string // a base item code, or ...
	Type    string // ... an item type (a random base item of it)
	Special string // named results: "Cow Portal", "Pandemonium Portal", ...

	UseType bool // an item of the same base as the main input, re-rolled
	UseItem bool // the main input itself, modified

	Quality  d2drop.Quality // 0 = keep (useitem) or normal
	Ethereal bool
	Qty      int
	Sockets  int // sock=N: exactly N sockets (capped by the base item)
	Prefix   []int
	Suffix   []int
	Repair   bool // rep
	Recharge bool // rch
	Unsocket bool // uns
	Upgrade  bool // mod: keep the item's modifiers while changing its tier
	Tier     Tier // exc / eli: the tier mod upgrades to
	Mods     []Mod

	Level, PLevel, ILevel int
}

// Recipe is a row of CubeMain.txt.
type Recipe struct {
	Row         int // 0-based row number in the table (the stable id)
	Description string
	Enabled     bool
	Ladder      bool
	MinDiff     int
	Version     int
	Op          int
	Param       int
	Value       int
	Class       []string // class codes, empty = all
	NumInputs   int
	Inputs      []Input
	Outputs     []Output
}

// Table is the parsed CubeMain.txt.
type Table struct {
	Recipes []*Recipe
	// Skipped lists rows that could not be understood (row, reason). The
	// recipe is dropped: an unparseable row never matches.
	Skipped []string
}

var qualityTokens = map[string]d2drop.Quality{
	"low": d2drop.QualityLow, "nor": d2drop.QualityNormal, "hiq": d2drop.QualitySuperior,
	"mag": d2drop.QualityMagic, "set": d2drop.QualitySet, "rar": d2drop.QualityRare,
	"uni": d2drop.QualityUnique, "crf": d2drop.QualityCrafted, "tmp": QualityTempered,
}

// ParseTable reads CubeMain.txt. cat resolves whether a token is an item code
// or an item type.
func ParseTable(raw []byte, cat *Catalog) *Table {
	t := parseTable(raw)
	out := &Table{}

	for i, r := range t.rows {
		desc := t.s(r, "description")
		if desc == "" && t.s(r, "output") == "" {
			continue
		}

		rec := &Recipe{
			Row: i, Description: desc,
			Enabled: t.b(r, "enabled"), Ladder: t.b(r, "ladder"),
			MinDiff: t.n(r, "min diff"), Version: t.n(r, "version"),
			Op: t.n(r, "op"), Param: t.n(r, "param"), Value: t.n(r, "value"),
			NumInputs: t.n(r, "numinputs"),
		}

		for _, c := range strings.Split(t.s(r, "class"), ",") {
			if c = strings.TrimSpace(c); c != "" {
				rec.Class = append(rec.Class, c)
			}
		}

		var err error

		for k := 1; k <= 7; k++ {
			raw := t.s(r, "input "+strconv.Itoa(k))
			if raw == "" {
				continue
			}

			var in Input

			if in, err = parseInput(raw, cat); err != nil {
				break
			}

			rec.Inputs = append(rec.Inputs, in)
		}

		if err == nil {
			for _, pre := range []string{"", "b ", "c "} {
				if t.s(r, pre+"output") == "" {
					continue
				}

				var o Output

				if o, err = parseOutput(t, r, pre, cat); err != nil {
					break
				}

				rec.Outputs = append(rec.Outputs, o)
			}
		}

		if err == nil && (len(rec.Inputs) == 0 || len(rec.Outputs) == 0) {
			err = fmt.Errorf("no inputs or no outputs")
		}

		if err != nil {
			out.Skipped = append(out.Skipped, fmt.Sprintf("row %d %q: %v", i, desc, err))
			continue
		}

		out.Recipes = append(out.Recipes, rec)
	}

	return out
}

func parseInput(raw string, cat *Catalog) (Input, error) {
	in := Input{Raw: raw, Qty: 1}
	toks := strings.Split(raw, ",")

	first := strings.TrimSpace(toks[0])

	switch {
	case first == "any":
		in.Any = true
	case cat != nil && cat.bases[first] != nil:
		in.Code = first
	case cat != nil && cat.IsType(first):
		in.Type = first
		in.IsGemTier = len(first) == 4 && strings.HasPrefix(first, "gem") && first[3] >= '0' && first[3] <= '4'
	case strings.ContainsAny(first, " ") || (first != "" && first[0] >= 'A' && first[0] <= 'Z'):
		in.Unique = first
	default:
		return in, fmt.Errorf("unknown item %q", first)
	}

	for _, tk := range toks[1:] {
		tk = strings.TrimSpace(tk)

		switch {
		case qualityTokens[tk] != 0:
			in.Quality = qualityTokens[tk]
		case tk == "eth":
			in.Ethereal = yes3
		case tk == "noe":
			in.Ethereal = no3
		case tk == "sock":
			in.Socketed = yes3
		case tk == "nos":
			in.Socketed = no3
		case tk == "bas":
			in.Tier = TierBasic
		case tk == "exc":
			in.Tier = TierExceptional
		case tk == "eli":
			in.Tier = TierElite
		case tk == "upg":
			in.Upgrade = true
		case strings.HasPrefix(tk, "qty="):
			n, err := strconv.Atoi(tk[4:])
			if err != nil || n < 1 {
				return in, fmt.Errorf("bad qty %q", tk)
			}

			in.Qty = n
		default:
			return in, fmt.Errorf("unknown input modifier %q", tk)
		}
	}

	return in, nil
}

func parseOutput(t *table, r []string, pre string, cat *Catalog) (Output, error) {
	raw := t.s(r, pre+"output")
	o := Output{Raw: raw, Qty: 0}
	toks := strings.Split(raw, ",")
	first := strings.TrimSpace(toks[0])

	switch {
	case first == "usetype":
		o.UseType = true
	case first == "useitem":
		o.UseItem = true
	case cat != nil && cat.bases[first] != nil:
		o.Code = first
	case cat != nil && cat.IsType(first):
		o.Type = first
	case first != "":
		o.Special = first // "Cow Portal", "Pandemonium Portal", ...
	}

	for _, tk := range toks[1:] {
		tk = strings.TrimSpace(tk)

		switch {
		case qualityTokens[tk] != 0:
			o.Quality = qualityTokens[tk]
		case tk == "eth":
			o.Ethereal = true
		case tk == "rep":
			o.Repair = true
		case tk == "rch":
			o.Recharge = true
		case tk == "uns":
			o.Unsocket = true
		case tk == "mod":
			o.Upgrade = true
		case tk == "exc":
			o.Tier = TierExceptional
		case tk == "eli":
			o.Tier = TierElite
		case tk == "bas":
			o.Tier = TierBasic
		case tk == "sock":
			o.Sockets = -1
		case strings.HasPrefix(tk, "sock="):
			o.Sockets, _ = strconv.Atoi(tk[5:])
		case strings.HasPrefix(tk, "qty="):
			o.Qty, _ = strconv.Atoi(tk[4:])
		case strings.HasPrefix(tk, "pre="):
			n, err := strconv.Atoi(tk[4:])
			if err != nil {
				return o, fmt.Errorf("bad %q", tk)
			}

			o.Prefix = append(o.Prefix, n)
		case strings.HasPrefix(tk, "suf="):
			n, err := strconv.Atoi(tk[4:])
			if err != nil {
				return o, fmt.Errorf("bad %q", tk)
			}

			o.Suffix = append(o.Suffix, n)
		default:
			return o, fmt.Errorf("unknown output modifier %q", tk)
		}
	}

	o.Level, o.PLevel, o.ILevel = t.n(r, pre+"lvl"), t.n(r, pre+"plvl"), t.n(r, pre+"ilvl")

	for k := 1; k <= 5; k++ {
		p := pre + "mod " + strconv.Itoa(k)

		code := t.s(r, p)
		if code == "" {
			continue
		}

		m := Mod{Code: code, Chance: 100, Param: t.s(r, p+" param"),
			Min: t.n(r, p+" min"), Max: t.n(r, p+" max")}
		if c := t.s(r, p+" chance"); c != "" {
			m.Chance, _ = strconv.Atoi(c)
		}

		o.Mods = append(o.Mods, m)
	}

	return o, nil
}
