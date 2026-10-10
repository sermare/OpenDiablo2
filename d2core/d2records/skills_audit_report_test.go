package d2records

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

func init() {
	// stand-ins in the engine (STAND-IN comments in d2common/d2missile and
	// d2common/d2skill/pipeline.go)
	standIns[55] = "hit13 splash radius Param1+Param2*(lvl-1), area hit of fn 1"
	standIns[64] = "do15 bolt spiral + hit29 16-way nova ring"
}

func playerRows(rows []auditRow) []auditRow {
	var out []auditRow

	for _, a := range rows {
		if a.Player {
			out = append(out, a)
		}
	}

	return out
}

// Every referenced missile, state, overlay, skilldesc, summon monster,
// prerequisite and synergy skill of all 357 rows exists in the tables.
func TestAuditReferencesResolve(t *testing.T) {
	rows, _ := buildAudit(t)

	if len(rows) != 357 || len(playerRows(rows)) != 210 {
		t.Fatalf("%d rows, %d player rows, want 357 / 210", len(rows), len(playerRows(rows)))
	}

	for _, a := range rows {
		if len(a.Unresolved) > 0 {
			t.Errorf("skill %d %s: unresolved %v", a.ID, a.Name, a.Unresolved)
		}
	}
}

// Every castable player skill runs Start and Do at levels 1 and 20 and
// produces a missile, an effect or a melee strike. Double Throw needs a
// throwing weapon, which the audit unit does not hold.
func TestAuditPlayerSkillsProduceAnEffect(t *testing.T) {
	rows, _ := buildAudit(t)
	needsWeapon := map[int]bool{140: true}

	for _, a := range playerRows(rows) {
		if a.Passive {
			continue
		}

		for lvl, c := range map[int]castOutcome{1: a.Cast1, 20: a.Cast20} {
			switch {
			case c.Panic != "":
				t.Errorf("skill %d %s L%d panics: %s", a.ID, a.Name, lvl, c.Panic)
			case needsWeapon[a.ID]:
			case !c.StartOK || !c.DoOK:
				t.Errorf("skill %d %s L%d refused: %q", a.ID, a.Name, lvl, c.Reason)
			case !c.produced():
				t.Errorf("skill %d %s L%d produced nothing", a.ID, a.Name, lvl)
			}
		}
	}
}

// Passives carry stats (or a state) at level 20; Shape Shifting is the one
// passive that is only a tree prerequisite.
func TestAuditPassivesCarryStats(t *testing.T) {
	rows, _ := buildAudit(t)
	prereqOnly := map[int]bool{224: true}

	for _, a := range playerRows(rows) {
		if a.Passive && a.StatN == 0 && len(a.States) == 0 && !prereqOnly[a.ID] {
			t.Errorf("passive %d %s has no stats and no state", a.ID, a.Name)
		}
	}
}

// Loader fidelity and mana arithmetic against the raw cells, all 357 rows.
func TestAuditLoaderAndManaMatchTable(t *testing.T) {
	rm := loadRealRecords(t)
	reg := rm.SkillTable()
	raw := readRaw(t, "skills/patch_d2/skills.txt")
	num := func(r map[string]string, k string) int { n, _ := strconv.Atoi(r[k]); return n }

	n := 0

	for _, r := range raw.rows {
		if r["skill"] == "" || r["id"] == "" {
			continue
		}

		id := num(r, "id")
		sk := reg.ByID(id)
		rec := rm.Skill.Details[id]

		for _, c := range []struct {
			col      string
			got, exp int
		}{
			{"srvstfunc", sk.SrvStFunc, num(r, "srvstfunc")}, {"srvdofunc", sk.SrvDoFunc, num(r, "srvdofunc")},
			{"reqlevel", rec.Reqlevel, num(r, "reqlevel")}, {"maxlvl", rec.Maxlvl, num(r, "maxlvl")},
			{"mana", sk.Mana, num(r, "mana")}, {"lvlmana", sk.LvlMana, num(r, "lvlmana")},
			{"minmana", sk.MinMana, num(r, "minmana")}, {"manashift", sk.ManaShift, num(r, "manashift")},
		} {
			if c.got != c.exp {
				t.Errorf("skill %d %s %s: loaded %d, table %d", id, sk.Name, c.col, c.got, c.exp)
			}
		}

		for _, lvl := range []int{1, 2, 10, 20, 30} {
			want := d2combat.ManaCost(int16(num(r, "mana")), int16(num(r, "lvlmana")), int16(num(r, "minmana")),
				int16(num(r, "manashift")), lvl)

			if got := sk.ManaCost(lvl); got != want {
				t.Errorf("skill %d %s mana L%d: %d want %d", id, sk.Name, lvl, got, want)
			}
		}

		if sk.SrvMissile != r["srvmissile"] || sk.SrvMissileA != r["srvmissilea"] || sk.AuraState != r["aurastate"] {
			t.Errorf("skill %d %s: missile/aurastate columns differ", id, sk.Name)
		}

		n++
	}

	if n != 357 {
		t.Errorf("%d rows", n)
	}
}

// knownMissileGaps pins, per player skill, the missile movement / hit
// functions in the skill's missile closure that the sim does not model (see
// docs/skills-coverage.md). Fixing one means deleting its entry here.
var knownMissileGaps = map[int][]string{
	64:  {"frozenorbnova:do16"},
	78:  {"bonewallmaker:do13"},
	121: {"fistoftheheavensdelay:hit22"},
	130: {"howl:hit17"},
	138: {"shout:hit18"},
	146: {"battlecry:hit21"},
	149: {"battleorders:hit18"},
	155: {"battlecommand:hit18"},
	238: {"rabiescontagion:hit53", "rabiesplague:do30"},
	257: {"blade creeper:do20", "blade creeper:hit37"},
	280: {"royalstrikechaosice:do35"},
}

func TestAuditMissileGapsArePinned(t *testing.T) {
	rows, _ := buildAudit(t)

	for _, a := range playerRows(rows) {
		var got []string

		for _, g := range a.ClosureGaps {
			got = append(got, g)
		}

		want := knownMissileGaps[a.ID]
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("skill %d %s: unmodelled missile functions %v, pinned %v", a.ID, a.Name, got, want)
		}
	}
}

// ---- report ----

func fx(v int) string { // 8.8 as a number with one decimal
	return fmt.Sprintf("%.1f", float64(v)/256)
}

func TestWriteSkillsCoverage(t *testing.T) {
	path := os.Getenv("D2_WRITE_SKILLS_AUDIT")
	if path == "" {
		t.Skip("D2_WRITE_SKILLS_AUDIT not set")
	}

	rows, rm := buildAudit(t)
	mt := rm.MissileTable()

	var b strings.Builder

	players := playerRows(rows)
	count := map[string]int{}
	pass, cast, withGaps := 0, 0, 0

	for _, a := range players {
		count[a.DoState]++

		if a.Passive {
			pass++
		} else if a.Cast1.produced() && a.Cast20.produced() {
			cast++
		}

		if len(a.ClosureGaps) > 0 {
			withGaps++
		}
	}

	b.WriteString("# Skill engine coverage (all skills.txt rows)\n\n")
	b.WriteString("Generated by `D2_TABLES=$HOME/git/d2-tables D2_WRITE_SKILLS_AUDIT=docs/skills-coverage.md go test ./d2core/d2records -run TestWriteSkillsCoverage`.\n")
	b.WriteString("Tables: patch_d2 skills.txt (357 rows, 210 player), missiles.txt, States.txt, Overlay.txt, skilldesc.txt, monstats.txt. Numbers and ids only.\n\n")
	b.WriteString("Every row is cast through `d2skill.Pipeline` (Start + Do) at skill levels 1 and 20 with a hero (clvl 30, 5000 mana), one enemy 5 subtiles away and a zombie corpse at the aim point. Passives are evaluated through `PassiveStats`. The engine columns come from that run; the table columns from the loaded records. Pinned by `skills_audit_test.go` / `skills_audit_report_test.go`.\n\n")
	fmt.Fprintf(&b, "## Summary (210 player skills)\n\n| check | count |\n|---|---|\n")
	fmt.Fprintf(&b, "| rows in skills.txt | %d |\n| player skills | %d |\n| passives (stats at L20) | %d |\n", len(rows), len(players), pass)
	fmt.Fprintf(&b, "| castable skills that produce a missile, effect or strike at L1 and L20 | %d of %d |\n", cast, len(players)-pass)
	fmt.Fprintf(&b, "| with a registered do handler | %d |\n| generic srvmissile path | %d |\n| unresolved table references (missile, state, overlay, skilldesc, summon, prereq, synergy, sumskill) | 0 |\n", count["handler"], count["generic"])
	fmt.Fprintf(&b, "| skills whose missile closure has a movement/hit function the sim lacks | %d |\n\n", withGaps)

	b.WriteString("Columns: st/do = srvstfunc/srvdofunc; mana = mana cost at L1 / L20 (whole points); delay = cooldown frames L1 / L20; dmg20 = damage at L20 in points (elem = elemental min-max with synergies at zero, phys = min-max with a 10-20 weapon); elen = elemental length frames; missile = first srvmissile, vel L1->L20 and range; aura = aura length frames L20; pet = summon monster and petmax L20; syn = distinct synergy skill references in the calc columns; req = reqlevel / maxlvl / prerequisites; cast = result at L1/L20 (m missiles, e effect kinds, s strike); gaps = missile functions in the closure the sim does not model .\n\n")

	classes := []string{"ama", "sor", "nec", "pal", "bar", "dru", "ass"}
	cname := map[string]string{"ama": "Amazon", "sor": "Sorceress", "nec": "Necromancer", "pal": "Paladin",
		"bar": "Barbarian", "dru": "Druid", "ass": "Assassin"}

	for _, cl := range classes {
		fmt.Fprintf(&b, "## %s\n\n", cname[cl])
		b.WriteString("| id | skill | st/do | impl | mana | delay | dmg20 | elen | missile (vel, rng) | aura | pet | syn | req | cast | gaps / stand-in |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")

		for _, a := range players {
			if a.Class != cl {
				continue
			}

			dmg := "-"
			if a.DmgKind != "" {
				dmg = fmt.Sprintf("%s %s-%s", strings.SplitN(a.DmgKind, ":", 2)[0], fx(a.DmgMin), fx(a.DmgMax))
			}

			ms := "-"
			if a.Missile != "" {
				ms = fmt.Sprintf("%s (%d->%d, %d)", a.Missile, a.MVel1, a.MVel20, a.MR)
			}

			pet := "-"
			if a.Summon != "" {
				pet = fmt.Sprintf("%s x%d", a.Summon, a.PetMax20)
			}

			cs := "passive"
			if !a.Passive {
				cs = outcome(a.Cast1) + "/" + outcome(a.Cast20)
			} else {
				cs = fmt.Sprintf("stats %d/%d", a.Stat1, a.StatN)
			}

			notes := strings.Join(a.ClosureGaps, " ")
			if a.StandIn != "" {
				notes = strings.TrimSpace("STAND-IN " + a.StandIn + " " + notes)
			}

			fmt.Fprintf(&b, "| %d | %s | %d/%d | %s | %s / %s | %d / %d | %s | %s | %s | %s | %s | %d | %d/%d %s | %s | %s |\n",
				a.ID, a.Name, a.St, a.Do, a.DoState, fx(a.Mana1), fx(a.Mana20), a.Delay1, a.Delay20, dmg,
				orDash(a.Elen20), ms, orDash(a.AuraLen20), pet, a.Syn, a.ReqLvl, a.MaxLvl, strings.Join(a.Req, "+"), cs, notes)
		}

		b.WriteString("\n")
	}

	// monster and other skills: do-function coverage
	b.WriteString("## Non-player rows (147): srvdofunc coverage\n\nMonster, item and scroll skills run through the monster AI / item code rather than the player pipeline; `handler` means a `doTable` entry exists.\n\n| srvdofunc | rows | handler | ids |\n|---|---|---|---|\n")

	by := map[int][]int{}

	for _, a := range rows {
		if !a.Player {
			by[a.Do] = append(by[a.Do], a.ID)
		}
	}

	var dos []int
	for d := range by {
		dos = append(dos, d)
	}

	sort.Ints(dos)

	haveDo, haveRows := 0, 0

	for _, d := range dos {
		ids := by[d]
		sk := rm.SkillTable().ByID(ids[0])
		h := "no"

		if d == 0 || d2skill.Implemented(&d2skill.Skill{SrvDoFunc: d, SrvMissile: sk.SrvMissile, SrvMissileA: sk.SrvMissileA}) {
			h = "yes"
			haveRows += len(ids)
			haveDo++
		}

		strs := make([]string, len(ids))
		for i, id := range ids {
			strs[i] = strconv.Itoa(id)
		}

		fmt.Fprintf(&b, "| %d | %d | %s | %s |\n", d, len(ids), h, strings.Join(strs, " "))
	}

	fmt.Fprintf(&b, "\n%d of %d distinct srvdofunc values (%d of 147 rows) have a handler or the generic missile path.\n\n", haveDo, len(dos), haveRows)

	// missile functions
	b.WriteString("## Missile movement and hit functions used by player skills\n\n| function | modelled | skills |\n|---|---|---|\n")

	uses := map[string][]int{}

	for _, a := range players {
		for _, g := range a.ClosureGaps {
			f := g[strings.LastIndex(g, ":")+1:]
			uses[f] = append(uses[f], a.ID)
		}
	}

	var fs []string
	for f := range uses {
		fs = append(fs, f)
	}

	sort.Slice(fs, func(i, j int) bool {
		a, _ := strconv.Atoi(fs[i][strings.IndexAny(fs[i], "0123456789"):])
		c, _ := strconv.Atoi(fs[j][strings.IndexAny(fs[j], "0123456789"):])

		if fs[i][:2] != fs[j][:2] {
			return fs[i] < fs[j]
		}

		return a < c
	})

	for _, f := range fs {
		ids := dedupe(uses[f])
		strs := make([]string, len(ids))

		for i, id := range ids {
			strs[i] = strconv.Itoa(id)
		}

		fmt.Fprintf(&b, "| %s | no | %s |\n", f, strings.Join(strs, " "))
	}

	_ = mt

	b.WriteString("\nModelled by the sim: movement " + intList(d2missile.ModelledDoFuncs) + "; hit " + intList(d2missile.ModelledHitFuncs) + ".\n")

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func outcome(c castOutcome) string {
	if c.Panic != "" {
		return "PANIC"
	}

	if !c.StartOK || !c.DoOK {
		return "refused:" + c.Reason
	}

	s := ""
	if c.Missiles > 0 {
		s += fmt.Sprintf("%dm", c.Missiles)
	}

	if len(c.Effects) > 0 {
		s += fmt.Sprintf("%de", len(c.Effects))
	}

	if c.Melee {
		s += "s"
	}

	if s == "" {
		return "none"
	}

	return s
}

func orDash(n int) string {
	if n == 0 {
		return "-"
	}

	return strconv.Itoa(n)
}

func dedupe(l []int) []int {
	seen := map[int]bool{}

	var out []int

	for _, v := range l {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}

	sort.Ints(out)

	return out
}

func intList(l []int) string {
	s := make([]string, len(l))
	for i, v := range l {
		s[i] = strconv.Itoa(v)
	}

	return strings.Join(s, " ")
}
