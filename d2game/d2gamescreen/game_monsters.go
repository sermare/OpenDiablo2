package d2gamescreen

import (
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapgen"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

const (
	monsterSpawnOffset     = 12  // subtiles between the hero and a debug-spawned monster
	heroMeleeReach         = 7   // subtiles; the hero swings when this close
	heroRepathSeconds      = 0.4 // how often a chasing hero updates its destination
	monsterTestDelay       = 3.0 // seconds after the hero exists before AUTOMONSTER spawns
	monsterTestDefaultSecs = 30.0
	monsterTestRing        = 12 // subtiles from the hero where AUTOMONSTER places monsters
	monsterTestSettle      = 2.0
)

// monsterTest is the state of the OD2_AUTOMONSTER scenario.
type monsterTest struct {
	ref      string
	count    int
	pack     bool // spawn a natural group (MinGrp/MaxGrp, minions) instead of count copies
	duration float64
	elapsed  float64
	spawned  bool
	allDead  float64
}

// monsterDirector returns the monster director, creating it once the local
// hero exists.
func (v *Game) monsterDirector() *d2monsters.Director {
	if v.monsters != nil {
		return v.monsters
	}

	if v.localPlayer == nil {
		return nil
	}

	opt := d2monsters.Options{
		Seed:       uint32(v.gameClient.MapEngine.Seed()),
		Difficulty: d2monster.Normal,
		Expansion:  true, // the game data is Lord of Destruction (MonLvl*Ex columns)
		// the scenario spawns monsters next to a hero who may still be in town
		IgnoreTown: os.Getenv("OD2_AUTOMONSTER") != "" || os.Getenv("OD2_AUTOMERC") != "",
		OnSound:    v.onMonsterSound,
	}

	if diff, err := strconv.Atoi(os.Getenv("OD2_AUTOMONSTER_DIFF")); err == nil && diff >= 0 && diff <= 2 {
		opt.Difficulty = d2monster.Difficulty(diff)
	}

	v.monsters = d2monsters.NewDirector(v.asset, v.gameClient.MapEngine, v.playerList, v.logLevel, opt)

	return v.monsters
}

func (v *Game) playerList() []*d2mapentity.Player {
	list := make([]*d2mapentity.Player, 0, len(v.gameClient.Players))
	for _, p := range v.gameClient.Players {
		list = append(list, p)
	}

	return list
}

// advanceMonsters runs the monster simulation, the hero's melee attacks and
// the OD2_AUTOMONSTER scenario.
func (v *Game) advanceMonsters(elapsed float64) {
	d := v.monsterDirector()
	if d == nil {
		return
	}

	d.Advance(elapsed)
	v.advanceMerc(elapsed)
	v.logLevelStatus(elapsed)
	v.advanceHeroAttack(elapsed)
	v.advanceMonsterTest(elapsed)
}

// OnPlayerAttack makes the hero walk up to a monster and fight it.
func (v *Game) OnPlayerAttack(m *d2mapentity.Monster) {
	v.attackTarget = m
	v.attackRepathAcc = heroRepathSeconds // path immediately

	v.Infof("attacking %q", m.Label())

	if v.gameControls != nil {
		v.gameControls.NPCMenu.Close()
	}

	v.npcTarget = nil
}

// advanceHeroAttack keeps swinging at the attack target while it lives and
// chases it when it is out of reach.
func (v *Game) advanceHeroAttack(elapsed float64) {
	m := v.attackTarget
	if m == nil || v.localPlayer == nil {
		return
	}

	if !m.Alive() || v.localPlayer.Stats.Health <= 0 {
		v.attackTarget = nil
		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())
	mx, my := m.SubtilePos()

	// the same edge distance the monsters use, so a monster that can hit the
	// hero can also be hit back
	if d2monster.EdgeDistance(hx-mx, hy-my, 1) <= heroMeleeReach {
		if v.localPlayer.IsCasting() {
			return
		}

		v.localPlayer.StopMoving()
		v.localPlayer.SetDirection(v.localPlayer.Position.DirectionTo(m.Position.Vector))

		hero, target := v.localPlayer, m
		v.playHeroSwing()
		hero.StartAttack(func() { v.monsters.HeroStrike(hero, target) })

		return
	}

	v.attackRepathAcc += elapsed
	if v.attackRepathAcc >= heroRepathSeconds {
		v.attackRepathAcc = 0

		x, y := m.GetPositionF()
		v.OnPlayerMove(x, y)
	}
}

// advanceMonsterTest implements OD2_AUTOMONSTER=<monster id or name>[,count]:
// it spawns the monsters around the hero, lets the hero fight whatever is in
// reach, logs spawn / aggro / attack / hit / death / drop lines (the director
// does that) and prints a summary after OD2_AUTOMONSTER_SECONDS (default 30)
// or shortly after the last monster died. OD2_AUTOEXIT quits afterwards.
func (v *Game) advanceMonsterTest(elapsed float64) {
	ref := os.Getenv("OD2_AUTOMONSTER")
	if ref == "" || v.localPlayer == nil {
		return
	}

	t := v.monsterTest
	if t == nil {
		t = &monsterTest{ref: ref, count: 1, duration: monsterTestDefaultSecs}

		if parts := strings.SplitN(ref, ",", 2); len(parts) == 2 {
			t.ref = parts[0]

			if strings.EqualFold(strings.TrimSpace(parts[1]), "pack") {
				t.pack = true
			} else if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				t.count = n
			}
		}

		if secs, err := strconv.ParseFloat(os.Getenv("OD2_AUTOMONSTER_SECONDS"), 64); err == nil && secs > 0 {
			t.duration = secs
		}

		v.monsterTest = t
	}

	t.elapsed += elapsed
	if t.elapsed < monsterTestDelay {
		return
	}

	if !t.spawned {
		t.spawned = true
		v.spawnMonsterTest(t)

		return
	}

	switch {
	case os.Getenv("OD2_AUTOCAST") != "":
		v.autoCast(elapsed) // the hero casts a skill instead of swinging
	case os.Getenv("OD2_AUTOMONSTER_PASSIVE") == "":
		v.autoFight() // OD2_AUTOMONSTER_PASSIVE=1 leaves the hero idle to watch monsters attack
	}

	alive := 0

	for _, m := range v.monsters.Monsters() {
		if m.Alive() {
			alive++
		}
	}

	if alive == 0 && v.monsters.Counters.Spawned > 0 {
		t.allDead += elapsed
	}

	if t.elapsed < monsterTestDelay+t.duration && t.allDead < monsterTestSettle {
		return
	}

	c := v.monsters.Counters
	v.Infof("AUTOMONSTER summary spawned=%d aggro=%d attacks=%d attack_hits=%d hero_swings=%d hero_hits=%d "+
		"deaths=%d drops=%d hero_deaths=%d hero_hp=%d/%d %s", c.Spawned, c.Aggro, c.Attacks, c.AttackHits,
		c.HeroSwings, c.HeroHits, c.Deaths, c.Drops, c.HeroDeaths,
		v.localPlayer.Stats.Health, v.localPlayer.Stats.MaxHealth, v.soundSummary())
	v.Infof("AUTOMONSTER world packs=%d shots=%d shot_hits=%d blocked_steps=%d max_stack=%d hit_recoveries=%d",
		c.Packs, c.Shots, c.ShotHits, c.BlockedSteps, c.MaxStack, c.HitRecoveries)

	v.logCastSummary()

	t.elapsed = math.Inf(-1) // print once
	t.allDead = 0
	t.duration = math.MaxFloat64

	v.autoTestExit()
}

func (v *Game) spawnMonsterTest(t *monsterTest) {
	if area, err := strconv.Atoi(os.Getenv("OD2_AUTOMONSTER_AREA")); err == nil && area > 0 {
		v.monsters.SetAreaLevel(v.monsters.AreaLevelOf(area))
	}

	if t.pack {
		v.spawnPackTest(t)

		return
	}

	stat := v.monsters.FindStat(t.ref)
	if stat == nil {
		v.Errorf("AUTOMONSTER: unknown monster %q", t.ref)
		v.autoTestExit()

		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	v.Infof("AUTOMONSTER start monster=%s count=%d hero=(%d,%d) hero_hp=%d/%d", stat.Key, t.count, hx, hy,
		v.localPlayer.Stats.Health, v.localPlayer.Stats.MaxHealth)

	var leader *d2mapentity.Monster

	// the scenario forms a pack: the first monster leads, the rest follow
	// its commands (only AIs with group commands, e.g. Fallen, make use of it)
	// OD2_AUTOMONSTER_FAR=<subtiles> puts every second monster on a far ring, so
	// the log shows the sounds of near and far fights (volume and pan per
	// sound, SOUNDAT lines) and the far ones closing in.
	far, _ := strconv.Atoi(os.Getenv("OD2_AUTOMONSTER_FAR"))

	for i := 0; i < t.count; i++ {
		angle := 2 * math.Pi * float64(i) / float64(t.count)
		ring := float64(monsterTestRing)

		if far > 0 && i%2 == 1 {
			ring = float64(far)
		}

		x := hx + int(math.Round(math.Cos(angle)*ring))
		y := hy + int(math.Round(math.Sin(angle)*ring))

		m, err := v.monsters.SpawnNear(stat, x, y, 2)
		if err != nil {
			v.Errorf("AUTOMONSTER: spawn failed: %v", err)
			continue
		}

		if leader == nil {
			leader = m
		} else {
			v.monsters.Group(leader, m)
		}
	}
}

// autoFight makes the hero attack the nearest living monster (scenario only).
func (v *Game) autoFight() {
	if v.attackTarget != nil && v.attackTarget.Alive() {
		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	var best *d2mapentity.Monster

	bestDist := 0

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if dist := d2monster.Distance(hx-mx, hy-my); best == nil || dist < bestDist {
			best, bestDist = m, dist
		}
	}

	if best != nil {
		v.OnPlayerAttack(best)
	}
}

// spawnPackTest implements OD2_AUTOMONSTER=<monster>,pack: one natural group
// (or the super unique of that name) is planned from the monstats / superuniques
// columns and placed in a cluster; the director logs "MONSTER pack ..." with the
// composition and the scenario prints the per-class counts.
func (v *Game) spawnPackTest(t *monsterTest) {
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())
	centre := d2path.Point{X: hx + monsterTestRing, Y: hy}

	var (
		res *d2monsters.PackResult
		err error
	)

	if stat := v.monsters.FindStat(t.ref); stat != nil {
		res, err = v.monsters.SpawnGroup(stat, centre)
	} else {
		res, err = v.monsters.SpawnSuperUnique(t.ref, centre)
	}

	if err != nil {
		v.Errorf("AUTOMONSTER: pack failed: %v", err)
		v.autoTestExit()

		return
	}

	counts := map[string]int{}
	for _, m := range res.Monsters {
		counts[m.Stat.Key]++
	}

	v.Infof("AUTOMONSTER pack start ref=%s leader=%s planned=%d spawned=%d followers=%d hero=(%d,%d) classes=%v",
		t.ref, res.Leader.Label(), len(res.Plan.Members), len(res.Monsters), len(res.Leader.Brain.Minions),
		hx, hy, counts)
}

// levelStatusSeconds is how often OD2_REALMAPS levels log the hero / monster
// status (LEVELSTATUS lines) for the autotests.
const levelStatusSeconds = 2.0

func (v *Game) logLevelStatus(elapsed float64) {
	if d2mapgen.RealLevel() == 0 || v.localPlayer == nil {
		return
	}

	v.levelStatusAcc += elapsed
	if v.levelStatusAcc < levelStatusSeconds {
		return
	}

	v.levelStatusAcc = 0

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())
	alive, nearest := 0, -1

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		alive++

		mx, my := m.SubtilePos()
		if d := d2monster.Distance(hx-mx, hy-my); nearest < 0 || d < nearest {
			nearest = d
		}
	}

	c := v.monsters.Counters
	v.Infof("LEVELSTATUS hero=(%d,%d) tile=(%d,%d) hp=%d/%d monsters_alive=%d nearest=%d spawned=%d aggro=%d attacks=%d",
		hx, hy, hx/5, hy/5, v.localPlayer.Stats.Health, v.localPlayer.Stats.MaxHealth, alive, nearest,
		c.Spawned, c.Aggro, c.Attacks)
}
