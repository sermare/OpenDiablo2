package d2summon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// monstatsFixture holds the real patch_d2 rows of a few minions (the columns
// the loader reads; every other column is left out).
const monstatsFixture = "Id\thcIdx\tAI\tVelocity\tRun\tminHP\tmaxHP\tAC\tA1MinD\tA1MaxD\tA1TH\t" +
	"MinHP(N)\tMaxHP(N)\tAC(N)\tA1MinD(N)\tA1MaxD(N)\tA1TH(N)\tMinHP(H)\tMaxHP(H)\tAC(H)\tA1MinD(H)\tA1MaxD(H)\tA1TH(H)\t" +
	"ResFi\tResFi(N)\tResFi(H)\tSkill1\n" +
	"necroskeleton\t363\tNecroPet\t13\t15\t21\t21\t5\t1\t2\t5\t30\t30\t5\t1\t2\t4\t42\t42\t6\t1\t2\t6\t0\t0\t0\t\n" +
	"claygolem\t289\tNecroPet\t8\t8\t100\t100\t100\t2\t5\t40\t175\t175\t100\t2\t6\t66\t275\t275\t100\t3\t7\t92\t0\t0\t0\t\n" +
	"firegolem\t292\tNecroPet\t10\t10\t313\t313\t200\t10\t27\t120\t313\t313\t200\t10\t27\t120\t313\t313\t200\t10\t27\t120\t0\t0\t0\t\n" +
	"druidhawk\t419\tRaven\t10\t20\t20\t32\t25\t0\t0\t0\t20\t32\t25\t0\t0\t0\t20\t32\t25\t0\t0\t0\t0\t0\t0\tRaven\n"

func loadFixture(t *testing.T) *Templates {
	t.Helper()

	tp, err := LoadTemplates([]byte(monstatsFixture))
	if err != nil {
		t.Fatal(err)
	}

	return tp
}

func must(t *testing.T, tp *Templates, id string) *Template {
	t.Helper()

	m, ok := tp.ByID(id)
	if !ok {
		t.Fatalf("no template %s", id)
	}

	return m
}

func TestLoadTemplates(t *testing.T) {
	tp := loadFixture(t)

	sk := must(t, tp, "NecroSkeleton") // case insensitive
	if sk.Class != 363 || sk.AI != "NecroPet" || sk.Diff[Normal].MaxHP != 21 || sk.Diff[Nightmare].MinHP != 30 ||
		sk.Diff[Hell].MinHP != 42 || sk.Diff[Hell].AC != 6 || sk.Run != 15 {
		t.Errorf("skeleton %+v", sk)
	}

	if c, ok := tp.ByClass(419); !ok || c.ID != "druidhawk" || c.Skill1 != "Raven" {
		t.Errorf("by class: %+v", c)
	}
}

func TestComputeSkeleton(t *testing.T) {
	sk := must(t, loadFixture(t), "necroskeleton")

	// Raise Skeleton level 7, Skeleton Mastery 3 (par1 = 8 life per level, in
	// 8.8): calc1 = 50*(7-3) = 200 percent life, damagepercent 7*(7-3) = 28,
	// tohit (7+3)*15 = 150, armorclass (7+3)*? = 40, maxhp 3*8*256.
	m := Mods{
		HPPct: 200,
		Aura: []d2skill.StatMod{{Stat: "damagepercent", Value: 28}, {Stat: "tohit", Value: 150},
			{Stat: "armorclass", Value: 40}},
		Passive: []d2skill.StatMod{{Stat: "maxhp", Value: 3 * 8 * 256}},
	}

	for _, tc := range []struct {
		name        string
		diff        Difficulty
		hp, def, ar int
		dmin, dmax  int
	}{
		// 21*300/100 + 24 = 87 ; defense 5+40 ; AR 5+150 ; damage 1,2 * 128% = 1,2
		{"normal", Normal, 87, 45, 155, 1, 2},
		// 30*3 + 24
		{"nightmare", Nightmare, 114, 45, 154, 1, 2},
		// 42*3 + 24
		{"hell", Hell, 150, 46, 156, 1, 2},
	} {
		s := Compute(sk, tc.diff, m, nil)
		if s.MaxHP != tc.hp || s.Defense != tc.def || s.AR != tc.ar || s.DmgMin != tc.dmin || s.DmgMax != tc.dmax {
			t.Errorf("%s: %+v", tc.name, s)
		}
	}

	// a level 1 skeleton is the bare template
	s := Compute(sk, Normal, Mods{}, nil)
	if s.MaxHP != 21 || s.Defense != 5 || s.AR != 5 || s.Walk != 13 {
		t.Errorf("bare skeleton %+v", s)
	}

	// damage percent scales the hits: a 1-2 hit +300% is 4-8
	s = Compute(sk, Normal, Mods{Aura: []d2skill.StatMod{{Stat: "damagepercent", Value: 300}}}, nil)
	if s.DmgMin != 4 || s.DmgMax != 8 {
		t.Errorf("damage %d-%d", s.DmgMin, s.DmgMax)
	}
}

func TestComputeGolems(t *testing.T) {
	tp := loadFixture(t)

	// Clay Golem level 5 with Golem Mastery 2: calc1 = (100+35*4)*(100+ln12)/100-100
	// with ln12 = 35+... the evaluated value arrives as HPPct. 240 -> 100*340/100.
	clay := Compute(must(t, tp, "claygolem"), Normal, Mods{HPPct: 240,
		Passive: []d2skill.StatMod{{Stat: "velocitypercent", Value: 25}}}, nil)
	if clay.MaxHP != 340 || clay.Walk != 10 || clay.Run != 10 || clay.WalkPct != 25 {
		t.Errorf("clay %+v", clay)
	}

	// Fire Golem: absorbs fire, gets flat fire damage, resist 100 - dm12
	fire := Compute(must(t, tp, "firegolem"), Hell, Mods{Aura: []d2skill.StatMod{
		{Stat: "fireresist", Value: 120}, {Stat: "item_absorbfire_percent", Value: 30},
		{Stat: "firemindam", Value: 11}, {Stat: "firemaxdam", Value: 19}}}, nil)
	if fire.Res[ResFire] != ResistCap || fire.Absorb["item_absorbfire_percent"] != 30 || fire.FireMin != 11 ||
		fire.FireMax != 19 || fire.MaxHP != 313 {
		t.Errorf("fire golem %+v", fire)
	}
}

func TestComputeHPRoll(t *testing.T) {
	hawk := must(t, loadFixture(t), "druidhawk") // 20..32 life

	seen := map[int]bool{}
	r := d2rand.New(1234)

	for i := 0; i < 200; i++ {
		s := Compute(hawk, Normal, Mods{}, r)
		if s.MaxHP < 20 || s.MaxHP > 32 {
			t.Fatalf("hp %d outside 20..32", s.MaxHP)
		}

		seen[s.MaxHP] = true
	}

	if len(seen) < 8 {
		t.Errorf("only %d distinct rolls", len(seen))
	}

	if Compute(hawk, Normal, Mods{}, nil).MaxHP != 32 {
		t.Error("nil rng must take maxHP")
	}
}

func order(petType string, max, count int, kind string) *d2skill.SummonOrder {
	return &d2skill.SummonOrder{Key: petType, PetType: petType, Max: max, Count: count, Kind: kind}
}

// summon plans and applies an order, handing out sequential ids.
func summon(r *Roster, o *d2skill.SummonOrder, frame int, next *uint32) Plan {
	p := r.Plan(o)

	var ids []uint32

	for i := 0; i < p.Spawn; i++ {
		*next++
		ids = append(ids, *next)
	}

	r.Apply(o, p, ids, frame)

	return p
}

func TestRosterSkeletonLimit(t *testing.T) {
	var r Roster

	var id uint32

	// Raise Skeleton level 7: petmax 2+7/3 = 4. The fifth cast replaces the
	// oldest skeleton.
	o := order("skeleton", 4, 1, "minion")

	for i := 0; i < 4; i++ {
		if p := summon(&r, o, i, &id); p.Spawn != 1 || len(p.Evict) != 0 {
			t.Fatalf("cast %d: %+v", i, p)
		}
	}

	p := summon(&r, o, 5, &id)
	if p.Spawn != 1 || len(p.Evict) != 1 || p.Evict[0] != 1 || r.Count("skeleton") != 4 {
		t.Errorf("fifth cast %+v count %d", p, r.Count("skeleton"))
	}

	// a different group does not count
	if p := summon(&r, order("skeletonmage", 4, 1, "minion"), 6, &id); len(p.Evict) != 0 || r.Total() != 5 {
		t.Errorf("mage %+v total %d", p, r.Total())
	}

	// the limit shrinks (respec to level 1: petmax 1): all but the new one go
	p = summon(&r, order("skeleton", 1, 1, "minion"), 7, &id)
	if len(p.Evict) != 4 || r.Count("skeleton") != 1 {
		t.Errorf("shrunk limit %+v count %d", p, r.Count("skeleton"))
	}
}

func TestRosterSharedGroups(t *testing.T) {
	var r Roster

	var id uint32

	// all golems share the pettype: a Fire Golem replaces a Clay Golem
	clay := &d2skill.SummonOrder{Key: "ClayGolem", PetType: "golem", Max: 1, Count: 1}
	fire := &d2skill.SummonOrder{Key: "FireGolem", PetType: "golem", Max: 1, Count: 1}
	summon(&r, clay, 0, &id)

	p := summon(&r, fire, 1, &id)
	if len(p.Evict) != 1 || r.Count("golem") != 1 || r.Pets()[0].Key != "FireGolem" {
		t.Errorf("golem swap %+v pets %+v", p, r.Pets())
	}

	// six sentries of three skills: the sixth kills the first
	for i := 0; i < 5; i++ {
		kinds := []string{"lightningsentry", "chargeboltsentry", "infernosentry"}
		summon(&r, &d2skill.SummonOrder{Key: kinds[i%3], PetType: "assassintrap", Max: 5, Count: 1, Kind: "trap"}, i, &id)
	}

	first := r.idsOf("assassintrap")[0]
	p = summon(&r, &d2skill.SummonOrder{Key: "deathsentry", PetType: "assassintrap", Max: 5, Count: 1, Kind: "trap"}, 9, &id)

	if len(p.Evict) != 1 || p.Evict[0] != first || r.Count("assassintrap") != 5 {
		t.Errorf("sentry %+v count %d", p, r.Count("assassintrap"))
	}

	// vines share one slot, the three totems another
	summon(&r, &d2skill.SummonOrder{Key: "plaguepoppy", PetType: "vine", Max: 1, Count: 1}, 10, &id)

	if p = summon(&r, &d2skill.SummonOrder{Key: "vinecreature", PetType: "vine", Max: 1, Count: 1}, 11, &id); len(p.Evict) != 1 {
		t.Errorf("vine swap %+v", p)
	}
}

func TestRosterRavenTopUp(t *testing.T) {
	var r Roster

	var id uint32

	// Raven level 4: petmax min(4,5) = 4, count 4 at once
	o := order("raven", 4, 4, "minion")
	if p := summon(&r, o, 0, &id); p.Spawn != 4 {
		t.Fatalf("first cast %+v", p)
	}

	// full: recasting creates nothing and evicts nothing
	if p := summon(&r, o, 1, &id); p.Spawn != 0 || len(p.Evict) != 0 || r.Count("raven") != 4 {
		t.Errorf("recast at full %+v", p)
	}

	// two died: only two are replaced
	r.Remove(1)
	r.Remove(2)

	if p := summon(&r, o, 2, &id); p.Spawn != 2 || r.Count("raven") != 4 {
		t.Errorf("top up %+v count %d", p, r.Count("raven"))
	}

	// the level went up to 5: one more
	if p := summon(&r, order("raven", 5, 5, "minion"), 3, &id); p.Spawn != 1 {
		t.Errorf("level up %+v", p)
	}
}

func TestRosterExpiryAndClear(t *testing.T) {
	var r Roster

	var id uint32

	trap := order("assassintrap", 5, 1, "trap")
	trap.Frames = 100
	summon(&r, trap, 10, &id)
	summon(&r, order("skeleton", 4, 1, "minion"), 20, &id)

	if g := r.Expire(109); len(g) != 0 {
		t.Errorf("expired early: %v", g)
	}

	if g := r.Expire(110); len(g) != 1 || g[0] != 1 || r.Total() != 1 {
		t.Errorf("expiry: %v total %d", g, r.Total())
	}

	// uncounted pettype ("none"): bone wall pieces, no limit, no eviction
	wall := order(Unlimited, 64, 8, "wall")
	if p := summon(&r, wall, 30, &id); p.Spawn != 8 || len(p.Evict) != 0 {
		t.Errorf("wall %+v", p)
	}

	if p := summon(&r, wall, 31, &id); p.Spawn != 8 || r.Total() != 17 {
		t.Errorf("second wall %+v total %d", p, r.Total())
	}

	if ids := r.Clear(); len(ids) != 17 || r.Total() != 0 {
		t.Errorf("clear %d", len(ids))
	}

	if r.Remove(99) {
		t.Error("removed an unknown pet")
	}
}

// ---- real data ----

func realFile(t *testing.T, parts ...string) []byte {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	b, err := os.ReadFile(filepath.Join(append([]string{dir}, parts...)...))
	if err != nil {
		t.Skipf("table missing: %v", err)
	}

	return b
}

func TestRealMonstats(t *testing.T) {
	tp, err := LoadTemplates(realFile(t, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	// values read from patch_d2 monstats.txt
	sk := must(t, tp, "necroskeleton")
	if sk.Class != 363 || sk.Diff[Normal].MaxHP != 21 || sk.Diff[Nightmare].MaxHP != 30 || sk.Diff[Hell].MaxHP != 42 {
		t.Errorf("skeleton %+v", sk.Diff)
	}

	for id, ai := range map[string]string{"claygolem": "NecroPet", "druidhawk": "Raven", "plaguepoppy": "Vines",
		"vinecreature": "CycleOfLife", "bladecreeper": "BladeCreeper", "lightningsentry": "AssassinSentry"} {
		if m := must(t, tp, id); m.AI != ai {
			t.Errorf("%s AI = %q, want %q", id, m.AI, ai)
		}
	}
}

// Every summon of skills.txt resolves to a monstats row, and the limits of
// the three classes are the documented ones.
func TestRealSkillsSummons(t *testing.T) {
	tp, err := LoadTemplates(realFile(t, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	d := d2txt.LoadDataDictionary(realFile(t, "skills", "patch_d2", "skills.txt"))

	groups := map[string]map[string]bool{}
	fixedMax := map[string]string{}
	n := 0

	for d.Next() {
		key := d.String("summon")
		if key == "" {
			continue
		}

		n++

		if _, ok := tp.ByID(key); !ok {
			t.Errorf("skill %q summons %q, which is not in monstats", d.String("skill"), key)
		}

		pt := strings.ToLower(d.String("pettype"))
		if groups[pt] == nil {
			groups[pt] = map[string]bool{}
		}

		groups[pt][d.String("skill")] = true
		fixedMax[pt+"/"+d.String("skill")] = d.String("petmax")
	}

	if n < 30 {
		t.Fatalf("only %d summon skills", n)
	}

	// the shared groups of the notes
	for pt, want := range map[string][]string{
		"golem":        {"Clay Golem", "BloodGolem", "IronGolem", "FireGolem"},
		"assassintrap": {"Blade Sentinel", "Charged Bolt Sentry", "Wake of Fire Sentry", "Lightning Sentry", "Inferno Sentry", "Death Sentry"},
		"vine":         {"Plague Poppy", "Cycle of Life", "Vines"},
		"totem":        {"Oak Sage", "Heart of Wolverine", "Spirit of Barbs"},
	} {
		for _, sk := range want {
			if !groups[pt][sk] {
				t.Errorf("pettype %s lacks %s", pt, sk)
			}
		}
	}

	if fixedMax["golem/Clay Golem"] != "1" || fixedMax["assassintrap/Lightning Sentry"] != "5" {
		t.Errorf("petmax: %v", fixedMax)
	}
}
