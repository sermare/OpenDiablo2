package d2skill

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func countStatus(rows []CoverageRow, after bool) map[string]int {
	n := map[string]int{}

	for _, r := range rows {
		s := r.Before
		if after {
			s = r.After
		}

		n[s]++
	}

	return n
}

// TestCoverageNPSD checks the coverage table against the real skills.txt:
// every skill of the four classes is listed once with the right id and class,
// nothing marked castable is refused by the pipeline, and no skill is missing
// after this branch. With D2_WRITE_COVERAGE=<path> it writes the markdown.
func TestCoverageNPSD(t *testing.T) {
	reg := loadRealRegistry(t)
	class := map[string]string{"nec": "Necromancer", "pal": "Paladin", "sor": "Sorceress", "dru": "Druid"}
	seen := map[int]bool{}
	per := map[string]int{}

	for _, r := range CoverageNPSD {
		sk := reg.ByID(r.ID)
		if sk == nil || sk.Name != r.Name {
			t.Errorf("id %d: %q is not in skills.txt", r.ID, r.Name)
			continue
		}

		if class[sk.CharClass] != r.Class {
			t.Errorf("%s: class %s, table says %s", r.Name, class[sk.CharClass], r.Class)
		}

		if seen[r.ID] {
			t.Errorf("%s listed twice", r.Name)
		}

		seen[r.ID] = true
		per[r.Class]++

		if r.After == "M" {
			t.Errorf("%s is still missing", r.Name)
		}

		switch r.After {
		case "T":
			if !sk.Passive && sk.SrvDoFunc != 0 {
				t.Errorf("%s is marked passive but has do function %d", r.Name, sk.SrvDoFunc)
			}
		default:
			if !Implemented(sk) {
				t.Errorf("%s: status %s but the pipeline cannot cast it (do %d)", r.Name, r.After, sk.SrvDoFunc)
			}
		}
	}

	for _, id := range reg.IDs() {
		sk := reg.ByID(id)
		if c := class[sk.CharClass]; c != "" && !seen[id] {
			t.Errorf("%s (%s) is missing from the coverage table", sk.Name, c)
		}
	}

	for c, n := range per {
		if n != 30 {
			t.Errorf("%s lists %d skills, want 30", c, n)
		}
	}

	if path := os.Getenv("D2_WRITE_COVERAGE"); path != "" {
		if err := os.WriteFile(path, []byte(coverageMarkdown(CoverageNPSD)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func coverageMarkdown(rows []CoverageRow) string {
	var b strings.Builder

	before, after := countStatus(rows, false), countStatus(rows, true)
	cell := func(m map[string]int) string {
		return fmt.Sprintf("%d | %d | %d | %d | %d", m["F"], m["P"], m["S"], m["M"], m["T"])
	}

	b.WriteString("# Player skill coverage: Necromancer, Paladin, Sorceress, Druid\n\n")
	b.WriteString("Generated from `d2common/d2skill/coverage_npsd.go` (`D2_WRITE_COVERAGE=<file> go test ./d2common/d2skill -run CoverageNPSD`).\n")
	b.WriteString("Source of truth for the skill rows: the patch_d2 `skills.txt` (the export of the compiled `skills.bin`; `D2_TABLES=$HOME/git/d2-tables`).\n")
	b.WriteString("`do` is the srvdofunc (server do table 0x72fa48 in `skills-2.md`); `-` is the generic missile path.\n\n")
	b.WriteString("Status: **F** fully simulated, **P** partial (runs, a documented part is approximated or missing), **S** stub (only a state nothing reads), **M** missing (cast refused), **T** passive (no cast, feeds other skills).\n")
	b.WriteString("The statuses come from reading the handlers and the scenario logs (`scripts/verify.d/86-class-skills.sh`), not from diffing every skill against the exe. `VERIFIED` notes were read in the decompilation.\n\n")
	b.WriteString("| | F | P | S | M | T |\n|---|---|---|---|---|---|\n")
	b.WriteString("| before (start of feat/skills-npsd) | " + cell(before) + " |\n")
	b.WriteString("| after | " + cell(after) + " |\n\n")

	cur := ""

	for _, r := range rows {
		if r.Class != cur {
			cur = r.Class
			b.WriteString("\n## " + cur + "\n\n| id | skill | do | before | after | note |\n|---|---|---|---|---|---|\n")
		}

		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n", r.ID, r.Name, r.DoFunc, r.Before, r.After, r.Note))
	}

	b.WriteString("\n## Not in this list\n\n")
	b.WriteString("Fire Blast / Fire Trauma (and the other trap bombs) are Assassin skills, handled by the Amazon/Barbarian/Assassin branch.\n")

	return b.String()
}
