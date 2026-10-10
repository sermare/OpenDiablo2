package d2records

import "testing"

// TestRealMissileSpecialsVerified2 pins the table facts the second missile
// verification relies on (patch_d2 missiles.txt).
func TestRealMissileSpecialsVerified2(t *testing.T) {
	tbl := loadRealRecords(t).MissileTable()

	for _, tc := range []struct {
		name         string
		doFunc, hit  int
		sHitPar1     int
		collide      int
		subMissile1  string
		hitSubMissle string
	}{
		{"fireball", 1, 1, 4, 3, "", ""},                // radius from the table, not the skill
		{"meteorcenter", 1, 14, 0, 0, "", "meteorfire"}, // sHitPar1 empty: aurarangecalc
		{"poisonjav", 2, 0, 0, 3, "poisonjavcloud", ""}, // SrvDoFunc 2 trail
		{"meteorfire", 5, 0, 0, 3, "", ""},              // SrvDoFunc 5 animation only
		{"firewallmaker", 6, 0, 0, 8, "firewall", ""},   // SrvDoFunc 6 trail
		{"guidedarrow", 7, 10, 0, 3, "", ""},
	} {
		sp := tbl.ByName(tc.name)
		if sp == nil {
			t.Errorf("%s missing", tc.name)

			continue
		}

		if sp.SrvDoFunc != tc.doFunc || sp.SrvHitFunc != tc.hit || sp.SHitPar[0] != tc.sHitPar1 ||
			sp.CollideType != tc.collide || sp.SubMissile[0] != tc.subMissile1 || sp.HitSubMissile[0] != tc.hitSubMissle {
			t.Errorf("%s: do=%d hit=%d par1=%d collide=%d sub=%q hitsub=%q", tc.name, sp.SrvDoFunc, sp.SrvHitFunc,
				sp.SHitPar[0], sp.CollideType, sp.SubMissile[0], sp.HitSubMissile[0])
		}
	}
}
