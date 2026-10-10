package d2mapgen

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2animdata"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2loader/asset/types"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
)

// Level-generation benchmarks and budget test on the real game data (no window needed: animations
// bind their renderer lazily). They read the install folder from D2_GAME_DIR and are skipped when
// it is unset, like the other real-data tests:
//
//	D2_GAME_DIR="$HOME/.wine-d2classic/drive_c/Program Files (x86)/Diablo II" \
//	  go test ./d2core/d2map/d2mapgen -run LevelBudget -bench Level -benchtime 3x
//
// OD2_REALMAPS is set by the helpers, as the providers need it.

var perfMPQs = []string{ //nolint:gochecknoglobals // fixed load order of the 1.14b + LoD install
	"patch_d2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq",
	"d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2video.mpq", "d2speech.mpq",
}

// perfLevels is one level of every kind in every act: outdoor with exact tiles, preset levels,
// and mazes.
var perfLevels = []struct { //nolint:gochecknoglobals // table
	id   int
	name string
}{
	{2, "act1-blood-moor-outdoor"}, {4, "act1-stony-field-outdoor"}, {9, "act1-den-maze"},
	{17, "act1-burial-grounds-preset"}, {28, "act1-cathedral-maze"}, {74, "act2-claw-viper-preset"},
	{41, "act2-rocky-waste-outdoor"}, {47, "act2-sewers-maze"}, {60, "act2-tomb-maze"},
	{76, "act3-spider-forest-outdoor"}, {85, "act3-spider-cavern-maze"},
	{104, "act4-plains-of-despair-preset"}, {111, "act5-frigid-highlands-outdoor"},
	{115, "act5-glacial-trail-maze"}, {129, "act5-worldstone-maze"},
}

func perfAssets(tb testing.TB) *d2asset.AssetManager {
	tb.Helper()

	dir := os.Getenv("D2_GAME_DIR")
	if dir == "" {
		tb.Skip("D2_GAME_DIR not set (Diablo II install folder)")
	}

	tb.Setenv("OD2_REALMAPS", "1")

	a, err := d2asset.NewAssetManager(d2util.LogLevelError)
	if err != nil {
		tb.Fatal(err)
	}

	for _, m := range perfMPQs {
		if err = a.AddSource(filepath.Join(dir, m), types.AssetSourceMPQ); err != nil {
			tb.Skipf("game data incomplete: %v", err)
		}
	}

	for _, p := range d2resource.DataDictionaries() {
		if err = a.LoadRecords(p); err != nil {
			tb.Fatalf("%s: %v", p, err)
		}
	}

	data, err := a.LoadFile(d2resource.AnimationData)
	if err != nil {
		tb.Fatal(err)
	}

	a.Records.Animation.Data, _ = d2animdata.Load(data)

	return a
}

func perfGenerator(tb testing.TB, a *d2asset.AssetManager) *MapGenerator {
	tb.Helper()

	e := d2mapengine.CreateMapEngine(d2util.LogLevelError, a)

	g, err := NewMapGenerator(a, d2util.LogLevelError, e)
	if err != nil {
		tb.Fatal(err)
	}

	return g
}

const perfSeed = 0x101d574a

// perfCPU is the CPU time of the process: unlike wall-clock time it is not inflated when other
// programs compete for the machine.
func perfCPU() time.Duration {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0
	}

	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

// TestLevelBudget generates every level in perfLevels twice (cold caches, then warm) and fails when a
// level takes more than a generous budget. The limits are several times the measured times so a
// loaded machine does not flake; they catch accidental O(n^2) or per-tile file loading.
func TestLevelBudget(t *testing.T) {
	a := perfAssets(t)
	g := perfGenerator(t, a)

	const coldBudget, warmBudget = 15 * time.Second, 5 * time.Second

	for _, lv := range perfLevels {
		lv := lv

		t.Run(fmt.Sprintf("%d-%s", lv.id, lv.name), func(t *testing.T) {
			for pass, budget := range []time.Duration{coldBudget, warmBudget} {
				t0, c0 := time.Now(), perfCPU()
				err := g.GenerateRealMaze(lv.id, perfSeed, 0)
				el, cpu := time.Since(t0), perfCPU()-c0

				if err != nil {
					t.Fatalf("level %d (%s): %v", lv.id, lv.name, err)
				}

				t.Logf("PERF level=%d %s pass=%d ms=%.1f cpu_ms=%.1f", lv.id, lv.name, pass,
					float64(el)/float64(time.Millisecond), float64(cpu)/float64(time.Millisecond))

				if el > budget {
					t.Errorf("level %d (%s) pass %d took %v (budget %v)", lv.id, lv.name, pass, el, budget)
				}
			}
		})
	}
}

// BenchmarkLevelGenerate is the whole level build (DRLG layout, DT1/DS1 loading from the caches,
// stamps, exact tiles, monsters) with warm asset caches.
func BenchmarkLevelGenerate(b *testing.B) {
	a := perfAssets(b)
	g := perfGenerator(b, a)

	for _, lv := range perfLevels {
		if err := g.GenerateRealMaze(lv.id, perfSeed, 0); err != nil { // fill the caches
			b.Fatalf("level %d: %v", lv.id, err)
		}

		b.Run(fmt.Sprintf("%d-%s", lv.id, lv.name), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if err := g.GenerateRealMaze(lv.id, perfSeed, 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDRLGMaze is the pure layout step of a maze level (no assets besides the tables).
func BenchmarkDRLGMaze(b *testing.B) {
	a := perfAssets(b)

	tb, err := LoadDRLGTables(a)
	if err != nil {
		b.Fatal(err)
	}

	base, _ := d2rand.DrlgBaseSeed(perfSeed)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: 28, Difficulty: d2drlg.Difficulty(0), BaseSeed: base}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadDRLGTables is the table parsing done for every level load.
func BenchmarkLoadDRLGTables(b *testing.B) {
	a := perfAssets(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := LoadDRLGTables(a); err != nil {
			b.Fatal(err)
		}
	}
}
