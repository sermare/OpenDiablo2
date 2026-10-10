package d2monster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// Oracle for the act bosses and key uniques: the real 1.14b monstats.txt
// (patch_d2) rows against the AI table and the think functions' slot use.
// The expected table below was extracted from that file once and is the pin:
// a change of the engine's reader or of the AI slot conventions that breaks it
// must be deliberate. Skipped unless D2_TABLES points at the extracted tables.

type bossSkill struct {
	name, mode string
	level      int
}

type bossRow struct {
	id     string
	hc     int // hcIdx column = exe class id (divider row not counted)
	ai     string
	threat int
	del    [3]int
	dist   [3]int
	level  [3]int
	skills []bossSkill
	aip    [3][8]int
	res    [3][6]int // ResDm, ResMa, ResFi, ResLi, ResCo, ResPo per difficulty
}

var bossRows = []bossRow{
	{id: "andariel", hc: 156, ai: "Andariel", threat: 14, del: [3]int{15, 11, 9}, dist: [3]int{0, 0, 0}, level: [3]int{12, 49, 75},
		skills: []bossSkill{{"AndrialSpray", "seq_andarielspray", 1}, {"AndyPoisonBolt", "A1", 1}},
		aip:    [3][8]int{{30, 10, 30, 50, 0, 0, 0, 0}, {35, 8, 32, 55, 0, 0, 0, 0}, {35, 6, 34, 60, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, -50, 50, 50, 80}, {0, 0, -50, 50, 50, 50}, {66, 0, -50, 66, 66, 66}}},
	{id: "duriel", hc: 211, ai: "Duriel", threat: 0, del: [3]int{15, 15, 15}, dist: [3]int{0, 0, 0}, level: [3]int{22, 55, 88},
		skills: []bossSkill{{"Charge", "seq_durielcharge", 5}, {"Jab", "seq_durieljab", 15}, {"Smite", "seq_durielsmite", 7}, {"Holy Freeze", "NU", 1}},
		aip:    [3][8]int{{5, 33, 50, 0, 0, 0, 0, 0}, {5, 33, 50, 0, 0, 0, 0, 0}, {6, 33, 50, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 20, 20, 50, 20}, {0, 0, 50, 50, 75, 50}, {50, 33, 75, 75, 95, 75}}},
	{id: "mephisto", hc: 242, ai: "Mephisto", threat: 14, del: [3]int{15, 10, 6}, dist: [3]int{0, 40, 46}, level: [3]int{26, 59, 87},
		skills: []bossSkill{{"PrimeLightning", "A2", 6}, {"PrimeBolt", "A2", 6}, {"PrimePoisonNova", "A2", 7}, {"MephistoMissile", "A2", 5}, {"MephFrostNova", "A2", 1}, {"Blizzard", "A2", 5}},
		aip:    [3][8]int{{15, 25, 25, 0, 0, 0, 0, 0}, {20, 33, 33, 0, 0, 0, 0, 0}, {25, 33, 33, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 33, 33, 25, 50}, {0, 0, 50, 50, 25, 50}, {20, 50, 75, 75, 75, 75}}},
	{id: "diablo", hc: 243, ai: "Diablo", threat: 14, del: [3]int{15, 13, 12}, dist: [3]int{0, 0, 0}, level: [3]int{40, 62, 94},
		skills: []bossSkill{{"DiabLight", "SC", 3}, {"DiabCold", "S2", 4}, {"DiabFire", "S1", 5}, {"DiabWall", "S3", 3}, {"DiabRun", "seq_diablorun", 5}, {"PrimeFirewall", "S3", 8}, {"DiabPrison", "S3", 1}},
		aip:    [3][8]int{{0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 33, 33, 33, 50}, {0, 0, 50, 50, 50, 50}, {45, 0, 50, 50, 50, 50}}},
	{id: "baalthrone", hc: 543, ai: "BaalThrone", threat: 0, del: [3]int{15, 15, 15}, dist: [3]int{0, 0, 0}, level: [3]int{60, 70, 90},
		skills: []bossSkill{{"Decrepify", "S3", 5}, {"Baal Corpse Explode", "S3", 1}, {"Defense Curse", "S3", 3}, {"Blood Mana", "S3", 3}},
		aip:    [3][8]int{{25, 0, 0, 0, 0, 0, 0, 0}, {25, 0, 0, 0, 0, 0, 0, 0}, {25, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 33, 33, 33, 50}, {0, 0, 50, 50, 50, 50}, {50, 0, 50, 50, 50, 50}}},
	{id: "baalcrab", hc: 544, ai: "BaalCrab", threat: 14, del: [3]int{15, 13, 12}, dist: [3]int{0, 0, 0}, level: [3]int{60, 75, 99},
		skills: []bossSkill{{"Baal Nova", "S3", 1}, {"Baal Inferno", "seq_baalinferno", 1}, {"Baal Tentacle", "S2", 1}, {"Baal Cold Missiles", "A1", 1}, {"Baal Teleport", "A1", 1}, {"Defense Curse", "S3", 3}, {"Blood Mana", "S3", 3}},
		aip:    [3][8]int{{0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 33, 33, 33, 50}, {0, 0, 50, 50, 50, 50}, {50, 0, 50, 50, 50, 50}}},
	{id: "baalclone", hc: 570, ai: "BaalCrabClone", threat: 14, del: [3]int{15, 14, 12}, dist: [3]int{0, 0, 0}, level: [3]int{60, 69, 95},
		skills: []bossSkill{{"Baal Nova", "S3", 1}, {"Baal Inferno", "seq_baalinferno", 1}, {"Baal Tentacle", "S2", 1}, {"Baal Cold Missiles", "A1", 1}, {"Baal Clone Teleport", "A1", 1}, {"Defense Curse", "S3", 3}, {"Blood Mana", "S3", 3}},
		aip:    [3][8]int{{0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 15, 15, 15, 33}, {0, 0, 25, 25, 25, 25}, {50, 0, 25, 25, 25, 25}}},
	{id: "baaltaunt", hc: 545, ai: "BaalTaunt", threat: 0, del: [3]int{15, 15, 15}, dist: [3]int{0, 0, 0}, level: [3]int{0, 33, 66},
		skills: []bossSkill{{"Baal Taunt", "A1", 1}},
		aip:    [3][8]int{{3, 10, 20, 0, 0, 0, 0, 0}, {3, 10, 20, 0, 0, 0, 0, 0}, {3, 10, 20, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0}}},
	{id: "baalcrabstairs", hc: 559, ai: "BaalToStairs", threat: 0, del: [3]int{15, 15, 15}, dist: [3]int{0, 0, 0}, level: [3]int{60, 70, 96},
		skills: []bossSkill{},
		aip:    [3][8]int{{4, 0, 0, 0, 0, 0, 0, 0}, {4, 0, 0, 0, 0, 0, 0, 0}, {4, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 33, 33, 33, 50}, {0, 0, 50, 50, 50, 50}, {0, 0, 50, 50, 50, 50}}},
	{id: "baaltentacle1", hc: 562, ai: "BaalTentacle", threat: 10, del: [3]int{15, 15, 15}, dist: [3]int{0, 0, 0}, level: [3]int{34, 67, 87},
		skills: []bossSkill{},
		aip:    [3][8]int{{70, 24, 10, 0, 0, 0, 0, 0}, {70, 24, 10, 0, 0, 0, 0, 0}, {70, 24, 10, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0}, {25, 0, 25, 0, 110, 0}}},
	{id: "baaltentacle5", hc: 566, ai: "BaalTentacle", threat: 10, del: [3]int{15, 15, 15}, dist: [3]int{0, 0, 0}, level: [3]int{42, 67, 87},
		skills: []bossSkill{},
		aip:    [3][8]int{{90, 16, 10, 0, 0, 0, 0, 0}, {90, 16, 10, 0, 0, 0, 0, 0}, {90, 16, 10, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0}, {25, 0, 25, 0, 130, 0}}},
	{id: "baalminion1", hc: 571, ai: "BaalMinion", threat: 10, del: [3]int{15, 14, 13}, dist: [3]int{0, 0, 0}, level: [3]int{55, 68, 92},
		skills: []bossSkill{{"Smite", "A2", 1}},
		aip:    [3][8]int{{90, 85, 50, 17, 0, 0, 0, 0}, {90, 85, 50, 17, 0, 0, 0, 0}, {90, 85, 50, 17, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 50, 0, 50, 95}, {0, 0, 95, 0, 50, 95}, {50, 0, 120, 33, 50, 95}}},
	{id: "bloodraven", hc: 267, ai: "BloodRaven", threat: 13, del: [3]int{15, 13, 10}, dist: [3]int{0, 0, 0}, level: [3]int{10, 43, 88},
		skills: []bossSkill{{"Nest", "seq_bloodravencast", 1}, {"Quick Strike", "seq_brquickstrike", 1}},
		aip:    [3][8]int{{0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 50, 50, 50, 50, 50}, {25, 50, 50, 50, 50, 50}, {50, 50, 50, 50, 50, 50}}},
	{id: "summoner", hc: 250, ai: "Summoner", threat: 14, del: [3]int{15, 13, 10}, dist: [3]int{0, 0, 0}, level: [3]int{18, 55, 80},
		skills: []bossSkill{{"Glacial Spike", "SC", 6}, {"Frost Nova", "SC", 5}, {"Fire Ball", "SC", 5}, {"VampireFirewall", "SC", 7}, {"Weaken", "SC", 4}},
		aip:    [3][8]int{{85, 5, 63, 40, 120, 33, 5, 40}, {93, 5, 63, 40, 100, 20, 8, 40}, {98, 5, 63, 40, 80, 10, 11, 40}},
		res:    [3][6]int{{0, 0, 50, 50, 50, 0}, {0, 0, 50, 50, 50, 0}, {0, 0, 75, 75, 75, 0}}},
	{id: "izual", hc: 256, ai: "Izual", threat: 14, del: [3]int{15, 12, 8}, dist: [3]int{0, 0, 0}, level: [3]int{29, 60, 86},
		skills: []bossSkill{{"Frost Nova", "SC", 8}},
		aip:    [3][8]int{{45, 50, 66, 0, 20, 3, 0, 0}, {50, 50, 66, 75, 5, 4, 0, 0}, {50, 50, 66, 100, 0, 4, 0, 0}},
		res:    [3][6]int{{30, 30, 30, 30, 75, 30}, {30, 30, 30, 30, 75, 30}, {30, 30, 30, 30, 75, 30}}},
	{id: "griswold", hc: 365, ai: "Griswold", threat: 11, del: [3]int{15, 13, 10}, dist: [3]int{0, 0, 0}, level: [3]int{5, 39, 84},
		skills: []bossSkill{},
		aip:    [3][8]int{{0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 0, 0, 0, 50}, {0, 0, 0, 0, 0, 50}, {50, 0, 0, 50, 0, 120}}},
	{id: "radament", hc: 229, ai: "GreaterMummy", threat: 14, del: [3]int{15, 14, 13}, dist: [3]int{0, 0, 0}, level: [3]int{16, 49, 83},
		skills: []bossSkill{{"Resurrect2", "seq_mummyres", 1}, {"Bestow", "seq_mummyres", 1}, {"UnHolyBolt", "seq_mummyres", 1}},
		aip:    [3][8]int{{85, 45, 55, 85, 24, 0, 0, 0}, {85, 50, 55, 90, 27, 0, 0, 0}, {85, 55, 55, 95, 31, 0, 0, 0}},
		res:    [3][6]int{{0, 0, 0, 0, 40, 50}, {0, 0, 0, 0, 60, 60}, {50, 0, 0, 0, 60, 80}}},
}

func loadBossMonstats(t *testing.T) ([]byte, *TxtProfiles) {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	tp, err := LoadTxtProfiles(buf)
	if err != nil {
		t.Fatal(err)
	}

	return buf, tp
}

func TestBossOracleProfiles(t *testing.T) {
	_, tp := loadBossMonstats(t)

	for _, r := range bossRows {
		class, ok := tp.ByID(r.id)
		if !ok {
			t.Errorf("%s: not in monstats", r.id)

			continue
		}

		if class != r.hc {
			t.Errorf("%s: class %d want %d", r.id, class, r.hc)
		}

		for d := Normal; d <= Hell; d++ {
			p, _ := tp.Profile(class, d)

			if p.AI != r.ai || p.Threat != r.threat || p.AIDel != r.del[d] || p.AIDist != r.dist[d] {
				t.Errorf("%s diff %d: ai=%q threat=%d aidel=%d aidist=%d", r.id, d, p.AI, p.Threat, p.AIDel, p.AIDist)
			}

			for n := 1; n <= 8; n++ {
				if p.AIP[n] != r.aip[d][n-1] {
					t.Errorf("%s diff %d: aip%d=%d want %d", r.id, d, n, p.AIP[n], r.aip[d][n-1])
				}
			}

			used := 0

			for i, s := range p.Skills {
				if !s.Used() {
					continue
				}

				used++

				if i >= len(r.skills) {
					t.Errorf("%s: unexpected Skill%d %q", r.id, i+1, s.Name)

					continue
				}

				want := r.skills[i]
				m, _ := ParseMode(want.mode)

				if s.Name != want.name || s.Level != want.level || s.Mode != m {
					t.Errorf("%s: Skill%d = %+v want %+v", r.id, i+1, s, want)
				}
			}

			if used != len(r.skills) {
				t.Errorf("%s: %d skills, want %d", r.id, used, len(r.skills))
			}
		}

		// every boss AI is implemented, not the idle stand-in
		if _, ok := Lookup(r.ai); !ok {
			t.Errorf("%s: AI %s not registered", r.id, r.ai)
		}
	}
}

// Levels and resists are read by the engine (d2monsters), not by the AI; they
// are pinned here from the same file with the txt reader.
func TestBossOracleResistsAndLevels(t *testing.T) {
	buf, _ := loadBossMonstats(t)
	d := d2txt.LoadDataDictionary(buf)
	byID := map[string]int{}

	for i := range bossRows {
		byID[bossRows[i].id] = i
	}

	sfx := [3]string{"", "(N)", "(H)"}
	cols := [6]string{"ResDm", "ResMa", "ResFi", "ResLi", "ResCo", "ResPo"}
	seen := 0

	for d.Next() {
		i, ok := byID[d.String("Id")]
		if !ok {
			continue
		}

		seen++
		r := bossRows[i]

		for diff := 0; diff < 3; diff++ {
			if got := d.Number("Level" + sfx[diff]); got != r.level[diff] {
				t.Errorf("%s: Level%s=%d want %d", r.id, sfx[diff], got, r.level[diff])
			}

			for k, c := range cols {
				got := d.Number(c + sfx[diff])
				if got != r.res[diff][k] {
					t.Errorf("%s: %s%s=%d want %d", r.id, c, sfx[diff], got, r.res[diff][k])
				}

				// immunity rule: 100 or more takes everything (the exe clamps the resist to 100 at 0x579c90)
				if want := got >= 100; want != (d2combat.ApplyResist(1000, got) == 0) {
					t.Errorf("%s: %s%s=%d immunity mismatch", r.id, c, sfx[diff], got)
				}
			}
		}
	}

	if seen != len(bossRows) {
		t.Errorf("found %d of %d rows", seen, len(bossRows))
	}
}

// Facts the table cannot show but the think functions depend on: which slots
// each boss AI reads exist in the real table, and the aip fields that are
// empty there (Duriel never charges and never uses A2).
func TestBossOracleSlotUse(t *testing.T) {
	_, tp := loadBossMonstats(t)

	need := map[string]int{ // highest 1-based Skill slot the AI casts
		"andariel": 2, "duriel": 4, "mephisto": 6, "diablo": 7, "summoner": 5, "izual": 1,
		"baalthrone": 4, "baalcrab": 7, "baalclone": 7, "baaltaunt": 1, "baalminion1": 1, "bloodraven": 2,
	}

	for id, n := range need {
		class, _ := tp.ByID(id)

		for d := Normal; d <= Hell; d++ {
			p, _ := tp.Profile(class, d)

			for s := 0; s < n; s++ {
				if !p.Skills[s].Used() {
					t.Errorf("%s diff %d: Skill%d empty, the AI reads it", id, d, s+1)
				}
			}

			for s := n; s < NumSkills; s++ {
				if p.Skills[s].Used() {
					t.Errorf("%s diff %d: Skill%d = %q is set but unused by the AI", id, d, s+1, p.Skills[s].Name)
				}
			}
		}
	}

	dur, _ := tp.ByID("duriel")

	for d := Normal; d <= Hell; d++ {
		p, _ := tp.Profile(dur, d)

		if p.AIP[4] != 0 || p.AIP[5] != 0 {
			t.Errorf("duriel diff %d: aip4=%d aip5=%d, the AI expects them empty", d, p.AIP[4], p.AIP[5])
		}
	}

	// Diablo has no aip values at all: his weights are built in
	dia, _ := tp.ByID("diablo")

	for d := Normal; d <= Hell; d++ {
		p, _ := tp.Profile(dia, d)

		for n := 1; n <= 8; n++ {
			if p.AIP[n] != 0 {
				t.Errorf("diablo diff %d: aip%d=%d", d, n, p.AIP[n])
			}
		}
	}
}

// Target modes of the Baal AIs. The exe values (verify-monster-ai.md,
// VERIFIED at the AI table 0x739c08) are BaalThrone 2, BaalCrab 0, BaalTaunt
// 1, BaalToStairs 1, BaalTentacle 1, BaalCrabClone 0. feat/verify-monster-ai
// corrected the earlier modelling, so the code now matches the exe column.
func TestBaalTargetModesDivergence(t *testing.T) {
	exe := map[string]int{"BaalThrone": 2, "BaalCrab": 0, "BaalTaunt": 1, "BaalToStairs": 1, "BaalTentacle": 1, "BaalCrabClone": 0}

	for name, want := range exe {
		d, ok := Lookup(name)
		if !ok {
			t.Fatalf("%s missing", name)
		}

		if d.TargetMode != want {
			t.Errorf("%s: mode %d, exe %d", name, d.TargetMode, want)
		}
	}
}
