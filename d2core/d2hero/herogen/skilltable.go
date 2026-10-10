package herogen

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SkillRow is the part of a skills.txt row the generator checks a build against.
type SkillRow struct {
	ID       int
	Name     string
	Class    string // charclass: ama, sor, nec, pal, bar, dru, ass
	ReqLevel int
	MaxLevel int
	Reqs     []string // reqskill1..3, by name
	Passive  bool     // has a passivestate
	Range    string   // h2h, rng, none, both
	DoFunc   int      // srvdofunc
	Missile  string   // srvmissile
}

// loadSkills reads skills.txt (patch_d2 first). It returns no rows and no error
// when the file is missing: the skill checks are then skipped.
func loadSkills(dir string) (map[int]SkillRow, error) {
	var raw []byte

	for _, p := range []string{
		filepath.Join("skills", "patch_d2", "skills.txt"), "skills.txt", filepath.Join("skills", "d2exp", "skills.txt"),
	} {
		b, err := os.ReadFile(filepath.Join(dir, p))
		if err == nil {
			raw = b
			break
		}
	}

	if raw == nil {
		return nil, nil
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	col := map[string]int{}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	for _, need := range []string{"skill", "Id", "charclass", "reqlevel", "maxlvl", "reqskill1", "range"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("herogen: skills.txt has no column %q", need)
		}
	}

	cell := func(f []string, name string) string {
		if i, ok := col[name]; ok && i < len(f) {
			return f[i]
		}

		return ""
	}

	out := map[int]SkillRow{}

	for _, l := range lines[1:] {
		f := strings.Split(l, "\t")

		id, err := strconv.Atoi(cell(f, "Id"))
		if err != nil || cell(f, "skill") == "" {
			continue
		}

		r := SkillRow{
			ID: id, Name: cell(f, "skill"), Class: cell(f, "charclass"), Range: cell(f, "range"),
			Missile: cell(f, "srvmissile"), Passive: cell(f, "passivestate") != "",
		}
		r.ReqLevel, _ = strconv.Atoi(cell(f, "reqlevel"))
		r.MaxLevel, _ = strconv.Atoi(cell(f, "maxlvl"))
		r.DoFunc, _ = strconv.Atoi(cell(f, "srvdofunc"))

		for _, k := range []string{"reqskill1", "reqskill2", "reqskill3"} {
			if v := cell(f, k); v != "" {
				r.Reqs = append(r.Reqs, v)
			}
		}

		out[id] = r
	}

	return out, nil
}

// SkillByName finds the skill of that name among the skills of a class code (ama, sor, ...).
func (t *Tables) SkillByName(class, name string) (SkillRow, bool) {
	for _, r := range t.Skills {
		if r.Class == class && r.Name == name {
			return r, true
		}
	}

	return SkillRow{}, false
}

// classTokens maps the class to the skills.txt charclass code.
var classTokens = [...]string{"ama", "sor", "nec", "pal", "bar", "dru", "ass"}

// ValidateSkills checks a build against skills.txt: every skill belongs to the
// class, is within its maximum level, is unlocked at the hero's level and has
// points in each skill it requires. It does nothing without a skills table.
func (t *Tables) ValidateSkills(spec Spec) error {
	if t.Skills == nil {
		return nil
	}

	token := classTokens[spec.Class]

	for id, pts := range spec.Skills {
		if pts == 0 {
			continue
		}

		r, ok := t.Skills[id]

		switch {
		case !ok:
			return fmt.Errorf("%w: skill %d is not in skills.txt", ErrSpec, id)
		case r.Class != token:
			return fmt.Errorf("%w: %s (%d) is a %q skill, not %q", ErrSpec, r.Name, id, r.Class, token)
		case pts > r.MaxLevel:
			return fmt.Errorf("%w: %s has %d points, the maximum is %d", ErrSpec, r.Name, pts, r.MaxLevel)
		case r.ReqLevel > spec.Level:
			return fmt.Errorf("%w: %s needs level %d, the hero is %d", ErrSpec, r.Name, r.ReqLevel, spec.Level)
		}

		for _, req := range r.Reqs {
			rr, ok := t.SkillByName(token, req)
			if !ok {
				return fmt.Errorf("%w: %s requires %q, which is not in skills.txt", ErrSpec, r.Name, req)
			}

			if spec.Skills[rr.ID] == 0 {
				return fmt.Errorf("%w: %s requires %s, which has no points", ErrSpec, r.Name, req)
			}
		}
	}

	return nil
}
