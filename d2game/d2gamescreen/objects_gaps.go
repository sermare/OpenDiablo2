package d2gamescreen

import (
	"fmt"
	"hash/fnv"
	"math"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// The object gaps of the second audit (feat/objects-gaps): exploding barrels, locked and trapped containers,
// the world shrines and the well charges. The rules are in d2common/d2object/gaps.go with the addresses of
// Game.exe they follow; this file applies them to the hero, the monsters and the map.

// objRoller is a deterministic generator for one object and one purpose, so a given game rolls the same
// lock, trap and explosion for the same object (the exe draws from the level and unit seeds; the stream
// position is UNVERIFIED).
func (v *Game) objRoller(ob *d2mapentity.Object, salt string) *d2object.Roller {
	h := fnv.New32a()
	fmt.Fprintf(h, "%d:%s", v.objectSeed(ob), salt)

	return d2object.NewRoller(h.Sum32())
}

// levelMonLvl1 is the MonLvl1 column of the current level: the exe reads the classic normal column in the
// container init functions (LevelsTxt wMonLvl1 at 0x54da20 / 0x54db00).
func (v *Game) levelMonLvl1() int {
	if det := v.asset.Records.GetLevelDetails(v.currentLevel()); det != nil {
		return det.MonsterLevelNormal
	}

	return 1
}

// objSubtile is the subtile position of an object.
func objSubtile(ob *d2mapentity.Object) (x, y int) {
	fx, fy := ob.GetPositionF()

	return int(math.Floor(fx * subtilesInTile)), int(math.Floor(fy * subtilesInTile))
}

// heroSubtile is the subtile position of the hero.
func (v *Game) heroSubXY() (x, y int) {
	fx, fy := v.heroTilePos()

	return int(math.Floor(fx * subtilesInTile)), int(math.Floor(fy * subtilesInTile))
}

// chestInitOf rolls, once per object, what OBJINIT_RollVariantWithHighFlag would have stored at creation:
// whether the container is locked and which spawn handler ("trap") it carries.
func (v *Game) chestInitOf(ob *d2mapentity.Object) d2object.ChestInit {
	in := v.objectInstance(ob)
	if in.chestInit != nil {
		return *in.chestInit
	}

	rec := ob.Record()
	ci := d2object.RollChestInit(rec.InitFn, rec.Lockable, v.levelMonLvl1(), v.objRoller(ob, "init"))
	in.chestInit = &ci

	if ci.Locked || ci.Handler != 0 {
		v.Infof("OBJECT container id=%d name=%q initfn=%d lockable=%v monlvl1=%d locked=%v spawn_handler=%d (trap %d%%, lock %d%%)",
			rec.Index, ob.Label(), rec.InitFn, rec.Lockable, v.levelMonLvl1(), ci.Locked, ci.Handler,
			d2object.TrapChancePct(v.levelMonLvl1()), d2object.LockChancePct(v.levelMonLvl1()))
	}

	return ci
}

// containerPlan is what operating a container does before its treasure is rolled.
type containerPlan struct {
	skip    bool // nothing further happens (locked without a key, exploded, already open)
	rolls   int  // treasure class rolls
	locked  bool
	handler int
	monster bool // a barrel releases a level monster
}

// planContainer applies the operate functions of the loot containers (OperateFn 7, 5, 3, 4...) up to the
// treasure roll. Containers of other functions keep one roll.
func (v *Game) planContainer(ob *d2mapentity.Object) containerPlan {
	rec := ob.Record()
	if ob.IsOpened() {
		return containerPlan{skip: true}
	}

	switch rec.OperateFn {
	case 7: // exploding barrel: damage around it, no treasure
		if opened, _ := ob.Open(); opened {
			v.playSoundAt(d2object.SoundFor(rec.OperateFn, rec.Name), ob.GetPosition(), "object")
			v.explodeObject(ob, 0)
		}

		return containerPlan{skip: true}
	case 30:
		if rec.SubClass&d2object.SubTrap == 0 {
			v.explodingChest(ob)
		}
	}

	ci := v.chestInitOf(ob)
	plan := containerPlan{rolls: 1, locked: ci.Locked, handler: ci.Handler}

	if ci.Locked && !v.takeKey(ob) {
		return containerPlan{skip: true}
	}

	rng := v.objRoller(ob, "drop")

	switch rec.OperateFn {
	case 3: // urns and baskets (0x5845d0)
		plan.rolls = 0
		if d2object.SmallContainerDrops(rng) {
			plan.rolls = 1
		}
	case 5: // barrels (0x5847b0)
		plan.rolls = 0
		if d2object.SmallContainerDrops(rng) {
			plan.rolls = 1
		}

		plan.monster = d2object.BarrelSpawnsMonster(rng)
	case 4: // chests (0x583e70)
		plan.rolls = d2object.ContainerDropRolls(ci.Locked, rng)
	}

	return plan
}

// takeKey consumes a key for a locked container (0x583e70 -> INV_ConsumeOneKey 0x55cf90). Without a key the
// container stays shut.
func (v *Game) takeKey(ob *d2mapentity.Object) bool {
	rec := ob.Record()
	if v.gameControls == nil {
		return false
	}

	src := v.gameControls.KeySource()
	if src == nil {
		v.Infof("OBJECT locked id=%d name=%q: a key is needed (the container stays shut)", rec.Index, ob.Label())
		return false
	}

	left := v.gameControls.ConsumeKey(src)
	v.Infof("OBJECT unlocked id=%d name=%q with a key (keys left in the stack %d)", rec.Index, ob.Label(), left)

	return true
}

// afterContainer runs what follows a container opening: the barrel's monster and the spawn handler (the trap)
// that fires SpawnDelayFrames later.
func (v *Game) afterContainer(ob *d2mapentity.Object, plan containerPlan) {
	if plan.monster {
		v.spawnLevelMonsters(ob, 1, "barrel")
	}

	if plan.handler == 0 {
		return
	}

	h := d2object.HandlerFor(plan.handler)
	at := v.objects.clock + float64(d2object.SpawnDelayFrames)/d2object.FramesPerSecond
	v.objects.spawns = append(v.objects.spawns, pendingSpawn{at: at, ob: ob, handler: plan.handler})
	v.Infof("OBJECT spawn handler %d (%s) of id=%d scheduled in %.2fs", plan.handler, spawnKindName(h), ob.Record().Index,
		at-v.objects.clock)
}

func spawnKindName(h d2object.SpawnHandler) string {
	switch h.Kind {
	case d2object.SpawnTrap:
		return "trap monster " + strconv.Itoa(h.Class)
	case d2object.SpawnFire:
		return "fire pair"
	case d2object.SpawnMonster:
		return "level monsters"
	}

	return "none"
}

// pendingSpawn is a container spawn handler waiting for its delay.
type pendingSpawn struct {
	at      float64
	ob      *d2mapentity.Object
	handler int
}

// runSpawns fires the due spawn handlers (OBJECT_ServerRunSpawnHandler's event 4, 0x580410).
func (v *Game) runSpawns() {
	keep := v.objects.spawns[:0]

	for _, s := range v.objects.spawns {
		if v.objects.clock < s.at {
			keep = append(keep, s)
			continue
		}

		if v.gameClient.MapEngine.Entities()[s.ob.ID()] == nil {
			continue // the level changed
		}

		v.fireSpawnHandler(s.ob, s.handler)
	}

	v.objects.spawns = keep
}

func (v *Game) fireSpawnHandler(ob *d2mapentity.Object, handler int) {
	h := d2object.HandlerFor(handler)
	rec := ob.Record()

	switch h.Kind {
	case d2object.SpawnTrap:
		md := v.monsterDirector()
		if md == nil {
			return
		}

		stat := md.FindStat(strconv.Itoa(h.Class))
		if stat == nil {
			v.Warningf("OBJECT spawn handler %d: no monstats row %d", handler, h.Class)
			return
		}

		sx, sy := objSubtile(ob)

		m, err := md.SpawnNear(stat, sx, sy, 1)
		if err != nil {
			v.Warningf("OBJECT trap of id=%d: %v", rec.Index, err)
			return
		}

		v.Infof("OBJECT trap id=%d handler=%d monster=%d %q spawned at subtile (%d,%d)", rec.Index, handler, h.Class,
			m.Label(), sx, sy)
	case d2object.SpawnMonster:
		n := d2object.LevelMonsterCount(v.objRoller(ob, "ambush"))
		v.spawnLevelMonsters(ob, n, "container")
	default:
		v.Infof("OBJECT spawn handler %d (%s) of id=%d: the two fire objects are not placed (UNVERIFIED decor)", handler,
			spawnKindName(h), rec.Index)
	}
}

// spawnLevelMonsters creates n monsters of the level's types next to an object (0x5801d0).
func (v *Game) spawnLevelMonsters(ob *d2mapentity.Object, n int, why string) {
	md := v.monsterDirector()
	if md == nil {
		return
	}

	sx, sy := objSubtile(ob)

	for i := 0; i < n; i++ {
		stat := md.LevelMonster(v.currentLevel())
		if stat == nil {
			v.Infof("OBJECT %s of id=%d: the level %d has no monster types", why, ob.Record().Index, v.currentLevel())
			return
		}

		m, err := md.SpawnNear(stat, sx, sy, 1)
		if err != nil {
			v.Warningf("OBJECT %s monster: %v", why, err)
			return
		}

		v.Infof("OBJECT %s id=%d released monster %q at subtile (%d,%d)", why, ob.Record().Index, m.Label(), sx, sy)
	}
}

// explodeObject is OBJECT_ServerDamageUnitsAroundObject (0x5820c0): the hero and every living monster within
// ExplosionRadius subtiles of the barrel may be hurt (hit chance and damage of 0x5de7f0), and every other
// unexploded barrel closer than ExplosionChainDistance explodes too. Walls between the barrel and a unit
// (PATH_CanTraceLineIgnoringBothUnits 0x804) are not tested (UNVERIFIED); town levels are safe (ROOM_IsTownRoom).
func (v *Game) explodeObject(ob *d2mapentity.Object, depth int) {
	rec := ob.Record()
	rng := v.objRoller(ob, "boom")
	bx, by := objSubtile(ob)
	hurt, tried := 0, 0

	town := d2level.IsTown(v.currentLevel())

	if p := v.localPlayer; p != nil && p.Stats != nil && p.Stats.Health > 0 && !town {
		hx, hy := v.heroSubXY()
		if d2object.WithinRadius(hx-bx, hy-by, d2object.ExplosionRadius) {
			tried++

			def := 0
			if p.Stats.Totals != nil {
				def = p.Stats.Totals.Defense
			}

			out := d2object.RollExplosion(rng, d2object.ExplosionTarget{Level: p.Stats.Level, Dex: p.Stats.Dexterity,
				Defense: def, Life8: p.Stats.Health << 8}, rec.Damage)
			v.Infof("OBJECT explosion id=%d target=hero chance=%d roll=%d hit=%v dmg8=%d", rec.Index, out.Chance, out.Roll,
				out.Hit, out.Damage8)

			if out.Hit {
				if md := v.monsterDirector(); md != nil {
					hurt += md.HurtHero(p, d2object.Fixed8Points(out.Damage8), "phys", "barrel")
				}
			}
		}
	}

	if md := v.monsterDirector(); md != nil && !town {
		for _, m := range md.Monsters() {
			if !m.Alive() {
				continue
			}

			mx, my := m.SubtilePos()
			if !d2object.WithinRadius(mx-bx, my-by, d2object.ExplosionRadius) {
				continue
			}

			tried++
			out := d2object.RollExplosion(rng, d2object.ExplosionTarget{Level: m.Vitals.Level, Defense: m.Vitals.Defense,
				Life8: m.Vitals.HP << 8}, rec.Damage)
			v.Infof("OBJECT explosion id=%d target=%q chance=%d roll=%d hit=%v dmg8=%d", rec.Index, m.Label(), out.Chance,
				out.Roll, out.Hit, out.Damage8)

			if out.Hit {
				hurt += md.HurtMonster(m, d2object.Fixed8Points(out.Damage8), v.localPlayer, "barrel")
			}
		}
	}

	v.Infof("OBJECT explosion id=%d name=%q at subtile (%d,%d) radius=%d targets=%d damage_total=%d chain_depth=%d", rec.Index,
		ob.Label(), bx, by, d2object.ExplosionRadius, tried, hurt, depth)

	// chain: other unexploded barrels closer than 3 (UNIT_GetDistanceToUnit < 3)
	sizeA := rec.SizeX
	if sizeA < 1 {
		sizeA = 1
	}

	for _, e := range v.gameClient.MapEngine.Entities() {
		other, ok := e.(*d2mapentity.Object)
		if !ok || other == ob || other.IsOpened() || other.Record().Index != d2object.ExplodingBarrelID {
			continue
		}

		ox, oy := objSubtile(other)

		sizeB := other.Record().SizeX
		if sizeB < 1 {
			sizeB = 1
		}

		if d2object.UnitDistance(ox-bx, oy-by, sizeA, sizeB) >= d2object.ExplosionChainDistance {
			continue
		}

		if opened, _ := other.Open(); opened {
			v.Infof("OBJECT explosion chain id=%d -> barrel at subtile (%d,%d)", rec.Index, ox, oy)
			v.playSoundAt(d2object.SoundFor(other.Record().OperateFn, other.Record().Name), other.GetPosition(), "object")
			v.explodeObject(other, depth+1)
		}
	}
}

// explodingChest is OBJECT_OperateDamageThenSetModeOne (0x57fb90, OperateFn 30): the activator is hit by two
// packets, physical and then fire (MONAI_ApplyDamageToNonTownTarget with kinds 0 and 1). The hero is the
// only activator; the chest's loot is unchanged (UNVERIFIED whether the chest also drops).
func (v *Game) explodingChest(ob *d2mapentity.Object) {
	p := v.localPlayer
	if p == nil || p.Stats == nil || d2level.IsTown(v.currentLevel()) {
		return
	}

	md := v.monsterDirector()
	if md == nil {
		return
	}

	rec := ob.Record()
	rng := v.objRoller(ob, "boom")

	for i, kind := range []string{"phys", "fire"} {
		def := 0
		if p.Stats.Totals != nil {
			def = p.Stats.Totals.Defense
		}

		out := d2object.RollExplosion(rng, d2object.ExplosionTarget{Level: p.Stats.Level, Dex: p.Stats.Dexterity, Defense: def,
			Life8: p.Stats.Health << 8}, rec.Damage)
		v.Infof("OBJECT trap chest id=%d packet=%d kind=%s chance=%d roll=%d hit=%v dmg8=%d", rec.Index, i, kind, out.Chance,
			out.Roll, out.Hit, out.Damage8)

		if out.Hit {
			md.HurtHero(p, d2object.Fixed8Points(out.Damage8), kind, "exploding chest")
		}
	}
}

// ---------------------------------------------------------------------------------------------------------
// Shrines
// ---------------------------------------------------------------------------------------------------------

// stormShrine is the Storm Shrine (0x580cb0): every living monster and the hero lose Arg0 percent of their
// current life, and 16 fireballs are created around the shrine. The fireballs are NOT launched: the object
// code has no hook into the missile world yet, so only the request list is logged.
func (v *Game) stormShrine(s d2object.Shrine) string {
	lost, units := 0, 0

	if p := v.localPlayer; p != nil && p.Stats != nil && p.Stats.Health > 0 {
		loss := d2object.StormLifeLoss(p.Stats.Health, s.Arg0)
		p.Stats.Health -= loss
		lost += loss
		units++
	}

	if md := v.monsterDirector(); md != nil {
		for _, m := range md.Monsters() {
			if !m.Alive() {
				continue
			}

			if loss := d2object.StormLifeLoss(m.Vitals.HP, s.Arg0); loss > 0 {
				md.HurtMonster(m, loss, v.localPlayer, "storm shrine")

				lost += loss
			}

			units++
		}
	}

	shots := d2object.StormMissiles()
	lvl := d2object.MissileLevel(v.localPlayer.Stats.Level)

	return fmt.Sprintf("world=storm life_lost=%d%% units=%d total_life_lost=%d fireballs=%d level=%d (missiles not launched: no object-missile hook)",
		s.Arg0, units, lost, len(shots), lvl)
}

// potionShrine is the Exploding and the Poison Shrine (0x580ff0 / 0x581320): Arg0..Arg1-1 potions land around
// the hero and six missiles are fired around the shrine. The potions are dropped; the missiles are only
// listed (see stormShrine).
func (v *Game) potionShrine(ob *d2mapentity.Object, s d2object.Shrine, ps d2object.PotionShrine) string {
	rng := v.objRoller(ob, "potions")
	n := d2object.PotionCount(s.Arg0, s.Arg1, rng)
	px, py := v.heroTilePos()
	cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), n, false)
	dropped := 0

	for i := 0; i < n && i < len(cells); i++ {
		it, err := v.itemFactory().ItemFromCode(ps.ItemCode, d2drop.QualityNormal, v.areaLevel(), v.chestSeed()+uint32(i)+7000)
		if err != nil {
			v.Warningf("OBJECT potion shrine: %v", err)
			break
		}

		it.SetQuantity(1)

		if _, err := v.spawnGroundItem(it, cells[i]); err != nil {
			v.Warningf("OBJECT potion shrine: %v", err)
			break
		}

		dropped++
	}

	return fmt.Sprintf("world=%s potions=%d dropped=%d code=%s missiles=%d id=%d level=%d (missiles not launched)", s.Name, n, dropped,
		ps.ItemCode, len(ps.PotionMissiles()), ps.MissileID, d2object.MissileLevel(v.localPlayer.Stats.Level))
}

// warpingShrine is OBJECT_OperateUpgradeNearbyMonsterToUnique (0x580f60): the nearest eligible monster becomes
// a unique with rolled modifiers (shrines.txt: "nearest monster becomes unique"; it is not a teleport).
func (v *Game) warpingShrine(ob *d2mapentity.Object) string {
	md := v.monsterDirector()
	if md == nil {
		return "world=warping no monsters"
	}

	bx, by := objSubtile(ob)

	var (
		best *d2mapentity.Monster
		bd   = math.MaxInt32
	)

	for _, m := range md.Monsters() {
		if !md.UniqueUpgradeCandidate(m) {
			continue
		}

		mx, my := m.SubtilePos()
		if d := (mx-bx)*(mx-bx) + (my-by)*(my-by); d < bd {
			best, bd = m, d
		}
	}

	if best == nil || !md.MakeUnique(best) {
		return "world=warping no eligible monster near the shrine"
	}

	return fmt.Sprintf("world=warping monster=%q dist2=%d mods=%v", best.Label(), bd, best.Modifiers)
}

// gemShrine is OBJECT_ShrineGenerateItemsFromInventoryCodes (0x580b50): the first loose gem of the inventory
// with a better gem yields one new gem of the better code next to the hero (the old gem stays); with none,
// a random chipped gem is created.
func (v *Game) gemShrine(ob *d2mapentity.Object) string {
	var (
		inv  []d2object.GemChoice
		gems = v.gameControls.InventoryGems()
	)

	for _, g := range gems {
		rec := g.CommonRecord()
		inv = append(inv, d2object.GemChoice{Code: g.GetItemCode(), BetterCode: rec.BetterGem})
	}

	code, up := d2object.GemUpgrade(inv, v.objRoller(ob, "gem"))

	px, py := v.heroTilePos()
	cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), 1, false)

	if len(cells) == 0 {
		return fmt.Sprintf("world=gem-upgrade code=%s upgraded=%v no free cell", code, up)
	}

	it, err := v.itemFactory().ItemFromCode(code, d2drop.QualityNormal, v.areaLevel(), v.chestSeed()+8000)
	if err != nil {
		return fmt.Sprintf("world=gem-upgrade code=%s failed: %v", code, err)
	}

	it.SetQuantity(1)

	if _, err := v.spawnGroundItem(it, cells[0]); err != nil {
		return fmt.Sprintf("world=gem-upgrade code=%s failed: %v", code, err)
	}

	return fmt.Sprintf("world=gem-upgrade gems_in_inventory=%d code=%s upgraded=%v", len(gems), code, up)
}
