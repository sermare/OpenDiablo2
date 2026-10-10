package d2combat

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type e2eCase struct {
	AStats, DStats                                        [][2]int
	ALevel, AKind, DKind                                  int
	DLife, DMax                                           int
	Demon, Undead, Cls2, ASpecial, DSpecial, DBoss, ADead int
	ANoHeal, A2F, D85, D83, DFlag2, DFlagC                int
	Active, Mode, StrB, DexB, Mast, Crit, Diff, Expn, Pen int
	DivL, DivM, Drain, Scale, Skov, Flags0, Conv, CPct    int
	InitPhys, DMana, DStam                                int
	ALife, AMax                                           int
	Seed                                                  [2]uint32
	Built, Resolved, Applied                              []int32
	FDLife, FDMana, FDStam, FALife, FAMana                int
	States                                                [][]interface{}
	Ev                                                    []int
	Seed2                                                 [2]uint32
	RowBoss, Blunt, MonDmg, DDead                         int
	PhysRec                                               [][]int64
}

func statGetter(p [][2]int) func(int) int32 {
	m := map[int]int32{}
	for _, kv := range p {
		m[kv[0]] = int32(kv[1])
	}

	return func(s int) int32 { return m[s] }
}

func damageWords(d Damage) []int32 {
	w := make([]int32, 28)
	w[0], w[1] = int32(d.Flags), int32(d.Result)

	for i := 2; i <= 24; i++ {
		if i >= 20 && i < 24 || i == 24 || (i >= 2 && i <= 19) {
			w[i] = *d.word(i)
		}
	}

	w[25] = d.ForcedClass

	return w
}

// TestOracleMeleePipeline runs random melee hits through BuildAttackerDamage,
// ResolveStruct, ApplyHit and the crushing blow / open wounds callbacks and
// compares the damage struct at every stage and the final life, mana and
// stamina of both units with Game.exe.
func TestOracleMeleePipeline(t *testing.T) {
	var g struct {
		Cases []json.RawMessage `json:"cases"`
	}

	loadJSON(t, "e2e_golden.json", &g)

	bad := 0
	stage := map[string]int{}

	for n, raw := range g.Cases {
		var c e2eCase

		var seed0 []uint32

		var seed2lo, seed2hi uint32

		var dc []interface{}
		vals := []interface{}{&c.AStats, &c.ALevel, &c.AKind, &c.DKind, &c.DStats, new(int), new(int), new([]int), &c.AMax, &c.ALife, &c.DLife, &c.DMax,
			&c.Demon, &c.Undead, &c.Cls2, &c.ASpecial, &c.DSpecial, &c.DBoss, &c.ADead, &c.ANoHeal, &c.A2F, &c.D85, &c.D83, &c.DFlag2, &c.DFlagC,
			&c.Active, &c.Mode, &c.StrB, &c.DexB, &c.Mast, &c.Crit, &c.Diff, &c.Expn, &c.Pen, &c.DivL, &c.DivM, &c.Drain, &c.Scale, &c.Skov, &c.Flags0,
			&c.Conv, &c.CPct, &c.InitPhys, &c.DMana, &c.DStam, &seed0, &c.Built, &c.Resolved, &c.Applied, &c.FDLife, &c.FDMana, &c.FDStam,
			&c.FALife, &c.FAMana, &c.States, &c.Ev, &seed2lo, &seed2hi, &c.RowBoss, &c.Blunt, &c.PhysRec, &c.MonDmg, &c.DDead}
		_ = dc

		if err := json.Unmarshal(raw, &[]interface{}{}); err != nil {
			t.Fatal(err)
		}

		var row []json.RawMessage
		if err := json.Unmarshal(raw, &row); err != nil {
			t.Fatal(err)
		}

		for i, v := range vals {
			if err := json.Unmarshal(row[i], v); err != nil {
				t.Fatalf("case %d field %d: %v", n, i, err)
			}
		}

		c.Seed = [2]uint32{seed0[0], seed0[1]}

		aget, dget := statGetter(c.AStats), statGetter(c.DStats)
		seed := &d2rand.Seed{Lo: c.Seed[0], Hi: c.Seed[1]}

		o := BuildAttackerDamage(seed, MeleeIn{
			Get: aget, AttackerKind: c.AKind, AttackerMerc: c.Cls2 == 2, Special: c.ASpecial == 1, DefenderKind: c.DKind,
			DefenderDemon: c.Demon == 1, DefenderUndead: c.Undead == 1, DefenderAC: dget(0x1f), Scale: uint8(c.Scale), SkillOverride: c.Skov == 1,
			Damage: Damage{Flags: uint32(c.Flags0), Physical: int32(c.InitPhys)}, ConvClass: byte(c.Conv), ConvPct: int32(c.CPct),
			Weapon:       &WeaponRollIn{HandMode: c.Mode, StrBonus: int16(c.StrB), DexBonus: int16(c.DexB), Mastery: int32(c.Mast)},
			ActiveWeapon: c.Active != 0, WeaponStrike: int32(c.Crit), UndeadBlunt: c.Blunt == 1 && c.Active != 0,
		})
		built := damageWords(o.Damage)

		if !equalWords(built, c.Built, 25) {
			stage["build"]++
			bad++

			if stage["build"] <= 3 {
				t.Errorf("case %d build: a=%d d=%d merc=%d demon=%d undead=%d s79=%d s7a=%d s19=%d s78=%d active=%d mode=%d strb=%d dexb=%d mast=%d crit=%d scale=%d skov=%d flags0=%#x conv=%d/%d init=%d\n got %v\nwant %v",
					n, c.AKind, c.DKind, c.Cls2, c.Demon, c.Undead, aget(0x79), aget(0x7a), aget(0x19), aget(0x78), c.Active, c.Mode, c.StrB, c.DexB, c.Mast, c.Crit, c.Scale, c.Skov, c.Flags0, c.Conv, c.CPct, c.InitPhys, built, c.Built)
			}

			continue
		}

		aPlain := c.AKind == 1 && c.ASpecial == 0
		dPlain := c.DKind == 1 && c.DSpecial == 0

		res := ResolveStruct(ResolveIn{
			Dmg: o.Damage, AGet: aget, DGet: dget, AttackerKind: c.AKind, DefenderKind: c.DKind, AttackerPlainMonster: aPlain, DefenderPlainMonster: dPlain,
			DefenderUndead: c.Undead == 1, DefenderDemon: c.Demon == 1, AttackerState2F: c.A2F == 1, DefenderState85: c.D85 == 1, DefenderState83: c.D83 == 1,
			Scale: ScaleIn{
				HaveBoth: true, AttackerKind: c.AKind, DefenderKind: c.DKind, AttackerSpecial: c.ASpecial == 1, DefenderSpecial: c.DSpecial == 1,
				DefenderBoss: c.DBoss == 1, RowPresent: true, RowBossPct: int32(c.RowBoss),
			},
			Expansion: c.Expn == 1, Difficulty: c.Diff, Penalty: c.Pen,
		})
		resolved := damageWords(res)

		if !equalWords(resolved, c.Resolved, 25) {
			stage["resolve"]++
			bad++

			if stage["resolve"] <= 3 {
				t.Errorf("case %d resolve (a=%d d=%d aPlain=%v dPlain=%v):\n built %v\n got %v\nwant %v", n, c.AKind, c.DKind, aPlain, dPlain, built, resolved, c.Resolved)
			}

			continue
		}

		res.Flags = 0x20
		res.Result = res.Result&0xffff0000 | res.Result&0xfff6 | 1

		ap := ApplyHit(seed, ApplyIn{
			Dmg: res, AttackerKind: c.AKind, AttackerSpecial: c.ASpecial == 1,
			Life: int32(c.ALife), MaxLife: int32(c.AMax), Mana: int32(aget(8)), MaxMana: int32(aget(9)), AttackerDead: c.ADead == 1, AttackerNoHeal: c.ANoHeal == 1,
			DefenderKind: c.DKind, DefLife: int32(c.DLife), DefMaxLife: int32(c.DMax), DefMana: dget(8), DefStamina: dget(0xa),
			DefenderIsMonster: c.DKind == 1, DefenderNoDamage: c.MonDmg == 0, DefDead: c.DDead == 1, DefenderDrain: int32(c.Drain & 255), DiffLifeDiv: int32(c.DivL), DiffManaDiv: int32(c.DivM),
		})
		applied := damageWords(ap.Dmg)

		if !equalWords(applied, c.Applied, 25) {
			stage["apply-struct"]++
			bad++

			if stage["apply-struct"] <= 3 {
				t.Errorf("case %d apply struct: a=%d sp=%d d=%d drain=%d dlife=%d dmax=%d dmana=%d dstam=%d alife=%d amax=%d divs=%d/%d\n in %v\n got %v\nwant %v", n, c.AKind, c.ASpecial, c.DKind, c.Drain, c.DLife, c.DMax, c.DMana, c.DStam, c.ALife, c.AMax, c.DivL, c.DivM, damageWords(res), applied, c.Applied)
			}

			continue
		}

		// item events on the attacker: open wounds, then crushing blow (order assumed)
		defLife := ap.DefLife
		states := ap.States
		events := append([]int(nil), ap.Events...)
		result := ap.Dmg.Result

		var wounds []StateCall

		if ch := aget(0x87); ch > 0 {
			ow := RollOpenWoundsExe(seed, OpenWoundsInput{Chance: int(ch), AttackerLevel: c.ALevel, DefenderKind: c.DKind, DefenderFlagC: c.DFlagC == 1, Event: 7})
			if ow.Hit {
				wounds = append(wounds, StateCall{"wounds", []int32{OpenWoundsLength, int32(-ow.Value)}})
			}
		}

		if ch := aget(0x88); ch > 0 {
			cb := RollCrushingBlowExe(seed, CrushingInput{
				Chance: int(ch), DefenderKind: c.DKind, Special: c.DSpecial == 1, Boss: c.DBoss == 1, DataFlag2: c.DFlag2 == 1,
				PlayerCount: int(dget(100)), Event: 7, Life: defLife, PhysResistRaw: int(dget(0x24)),
			})
			if cb.Hit {
				defLife = cb.Life
				if cb.Knock {
					events = append(events, 0x93)
				}

				if cb.Died {
					result |= ResultDied
				}
			}
		}

		_ = result

		got := fmt.Sprint(defLife, ap.DefMana, ap.DefStamina, ap.Life, ap.Mana, filterEvents(events), seed.Lo, seed.Hi)
		want := fmt.Sprint(c.FDLife, c.FDMana, c.FDStam, c.FALife, c.FAMana, c.Ev, seed2lo, seed2hi)

		var gotStates, wantStates []string

		for _, s := range states {
			gotStates = append(gotStates, fmt.Sprintf("%s%v", s.Name, s.Args))
		}

		for _, s := range wounds {
			gotStates = append(gotStates, fmt.Sprintf("%s%v", s.Name, s.Args))
		}

		for _, s := range c.States {
			wantStates = append(wantStates, stateString(s))
		}

		if got != want || fmt.Sprint(gotStates) != fmt.Sprint(wantStates) {
			stage["final"]++
			bad++

			if stage["final"] <= 5 {
				t.Errorf("case %d final (a=%d d=%d) dlife=%d dmana=%d dstam=%d alife=%d amana=%d applied=%v:\n got %s %v\nwant %s %v", n, c.AKind, c.DKind, c.DLife, c.DMana, c.DStam, c.ALife, aget(8), applied, got, gotStates, want, wantStates)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d pipeline cases differ: %v", bad, len(g.Cases), stage)
	}
}

func filterEvents(e []int) []int {
	if e == nil {
		return []int{}
	}

	return e
}

func stateString(s []interface{}) string {
	// golden state rows are [name, args...] with the name as the first element
	args := make([]int32, 0, len(s)-1)
	for _, a := range s[1:] {
		args = append(args, int32(a.(float64)))
	}

	return fmt.Sprintf("%s%v", s[0].(string), args)
}

func equalWords(a, b []int32, n int) bool {
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
