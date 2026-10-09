package d2gamescreen

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

const (
	hirelingTablePath = "/data/global/excel/hireling.txt"
	mercTestDelay     = 3.0
	mercTestFightSecs = 30.0
	mercTestFollowGap = 60 // subtiles the hero walks away in the follow phase

	mercTestDefault = "skeleton1,4"
)

// sellerOffers is the offer table of one hire NPC in this game (per game and
// per vendor, not per player: hirelings.md).
type sellerOffers struct {
	rng   *d2rand.Seed
	table *d2hireling.OfferTable
}

// mercGame is the merc-related state of the game screen.
type mercGame struct {
	table      *d2hireling.Table
	tried      bool
	offers     map[int]*sellerOffers
	geared     bool                 // the give/take path of the controls is wired to the director
	spawnedFor *d2monsters.Director // the director the merc was spawned into: a new level has a new one, so the merc follows the hero
	test       *mercTest
}

// mercTest is the state of the OD2_AUTOMERC scenario.
type mercTest struct {
	ref     string
	count   int
	elapsed float64
	phase   int
	phaseT  float64
	startX  float64
	startY  float64
	logAcc  float64
	done    bool
}

// hirelingTable loads hireling.txt once (nil when the game data has none:
// a classic install has no mercenaries).
func (v *Game) hirelingTable() *d2hireling.Table {
	if v.merc.tried {
		return v.merc.table
	}

	v.merc.tried = true

	data, err := v.asset.LoadFile(hirelingTablePath)
	if err != nil {
		v.Infof("MERC no hireling.txt (%v): mercenaries are unavailable", err)
		return nil
	}

	t, err := d2hireling.Parse(strings.NewReader(string(data)))
	if err != nil || len(t.Rows) == 0 {
		v.Errorf("MERC hireling.txt: %v", err)
		return nil
	}

	t = t.ForVersion(d2hireling.ExpansionVersion) // the Lord of Destruction rows (V: lookups use expansion ? 100 : 0)
	v.merc.table = t
	v.Infof("MERC loaded %d hireling rows (expansion)", len(t.Rows))

	return t
}

// mercDifficulty is the 1-based difficulty used for offers.
func (v *Game) mercDifficulty() int {
	if v.monsters != nil {
		return int(v.monsters.Difficulty()) + 1
	}

	return 1
}

// advanceMerc keeps the merc alive in the game: it hands the table to the
// director, spawns the hero's merc from the save, mirrors its experience and
// dead flag back into the hero (so the next save keeps them) and runs the
// OD2_AUTOMERC scenario.
func (v *Game) advanceMerc(elapsed float64) {
	d := v.monsters
	if d == nil || v.localPlayer == nil {
		return
	}

	if d.Hirelings() == nil {
		if t := v.hirelingTable(); t != nil {
			d.SetHirelings(t)
		}
	}

	p := v.localPlayer

	// the OD2_AUTOMONSTER scenarios test the hero alone: the save's merc stays home
	soloTest := (os.Getenv("OD2_AUTOMONSTER") != "" || os.Getenv("OD2_AUTOAI") != "" || os.Getenv("OD2_AUTOBOSS") != "") && os.Getenv("OD2_AUTOMERC") == ""

	if v.merc.spawnedFor != d && p.Merc != nil && d.Hirelings() != nil && !soloTest {
		v.merc.spawnedFor = v.monsters

		save := saveOf(p.Merc)
		if v.gameControls != nil {
			save.Gear = v.gameControls.MercStatItems() // the saved 'jf' items count from the first frame
		}

		if _, err := d.SpawnMerc(p, save); err != nil {
			v.Errorf("MERC spawn: %v", err)
		}
	}

	v.wireMercGear(p)

	if info, ok := d.Merc(p); ok && p.Merc != nil {
		p.Merc.Experience = info.Save.Experience
		p.Merc.Dead = info.Save.Dead
	}

	v.advanceMercTest(elapsed)
}

// wireMercGear connects the give/take path of the game controls with the merc unit
// (once): the rules need the merc's class and table stats, and every change of the
// gear is applied to the unit. A dead merc takes nothing (UNVERIFIED: the exe's
// distance and state checks of the packet handler were not traced).
func (v *Game) wireMercGear(p *d2mapentity.Player) {
	if v.merc.geared || v.gameControls == nil {
		return
	}

	v.merc.geared = true

	v.gameControls.SetMercGearHost(d2player.MercGearHost{
		Ref: func() (d2hero.MercRef, bool) {
			info, ok := v.monsters.Merc(p)
			if !ok || info.Save.Dead || info.Rec == nil {
				return d2hero.MercRef{}, false
			}

			return d2hero.MercRef{Class: info.Rec.Class, Base: info.Base}, true
		},
		Changed: func() {
			if v.monsters.SetMercItems(p, v.gameControls.MercStatItems()) {
				if info, ok := v.monsters.Merc(p); ok {
					v.Infof("MERC gear applied defense=%d hp=%d dmg=%d-%d ar=%d resist=%v", info.Stats.Defense, info.Stats.MaxHP,
						info.Stats.DmgMin, info.Stats.DmgMax, info.Stats.AR, info.Gear.Resist)
				}
			}
		},
	})
}

func saveOf(m *d2hero.MercState) d2monsters.MercSave {
	return d2monsters.MercSave{Dead: m.Dead, ID: m.ID, NameID: m.NameID, Type: m.Type, Experience: m.Experience}
}

// ---- offers and hiring ----

// offerTable returns the (lazily built) offer table of a seller.
func (v *Game) offerTable(seller int) *sellerOffers {
	t := v.hirelingTable()
	if t == nil {
		return nil
	}

	if v.merc.offers == nil {
		v.merc.offers = map[int]*sellerOffers{}
	}

	if o := v.merc.offers[seller]; o != nil {
		return o
	}

	// the exe draws from the game seed's own generator; the game seed plus the
	// seller class is the deterministic stand-in here (UNVERIFIED)
	rng := d2rand.New(uint32(v.gameClient.MapEngine.Seed()) + uint32(seller))

	tab := t.NewOfferTable(rng, seller, v.mercDifficulty())
	if tab == nil {
		return nil
	}

	o := &sellerOffers{rng: rng, table: tab}
	v.merc.offers[seller] = o

	return o
}

func (v *Game) mercDisplayName(rec *d2hireling.Record, nameID int) string {
	key := d2hireling.NameKey(rec, nameID)
	if s := v.asset.TranslateString(key); s != "" && s != key {
		return s
	}

	return key
}

// offerLine is the text of one hire row.
func (v *Game) offerLine(o d2hireling.Offer) string {
	return fmt.Sprintf("%s  Lv %d  %s  %dg", v.mercDisplayName(o.Rec, o.NameID), o.Stats.Level, o.Rec.SubType, o.Cost)
}

// openHire shows the hire list of a seller in the NPC menu: one row per
// offered mercenary (name, level, kind, price), and a revive row when the
// hero's merc is dead.
func (v *Game) openHire(npc d2interface.MapEntity) {
	seller := v.npcClassID(npc)

	if seller == d2hireling.SellerTyrael && v.localPlayer != nil && v.monsters != nil {
		// Tyrael (0x16f) is on the revive allow-list of 0x577a10 but sells no mercs
		v.openRevive(npc)

		return
	}

	o := v.offerTable(seller)
	if o == nil || v.localPlayer == nil {
		v.Infof("NPC menu: Hire at %q has no offers (class %d)", npc.Label(), seller)
		return
	}

	tab := v.hirelingTable()
	rows := []d2player.NPCMenuRow{}
	lines := []string{}

	for _, slot := range o.table.Offered() {
		offer, ok := tab.MakeOffer(o.table, slot, v.localPlayer.Stats.Level)
		if !ok {
			continue
		}

		line := v.offerLine(offer)
		rows = append(rows, d2player.NPCMenuRow{StringID: slot, Fallback: line, Action: d2player.NPCActionHireOffer})
		lines = append(lines, fmt.Sprintf("%s [slot %d, %s]", line, slot, offer.Rec.HireDesc))
	}

	if info, ok := v.monsters.Merc(v.localPlayer); ok && info.Save.Dead {
		rows = append(rows, d2player.NPCMenuRow{
			Fallback: fmt.Sprintf("Revive %s  %dg", v.mercDisplayName(info.Rec, int(info.Save.NameID)), info.ReviveCost),
			Action:   d2player.NPCActionReviveMerc,
		})
	}

	v.gameControls.NPCMenu.Open(npc.Label(), rows, 0, 0, func(row d2player.NPCMenuRow) {
		v.onHireChoice(seller, row)
	})
	v.anchorNPCMenu(v.gameControls.NPCMenu, npc)

	v.Infof("MERC offers seller=%d npc=%q difficulty=%d owner_level=%d count=%d", seller, npc.Label(), v.mercDifficulty(),
		v.localPlayer.Stats.Level, len(lines))

	for _, l := range lines {
		v.Infof("MERC offer %s", l)
	}
}

// openRevive shows only the revive row (Tyrael).
func (v *Game) openRevive(npc d2interface.MapEntity) {
	info, ok := v.monsters.Merc(v.localPlayer)
	if !ok || !info.Save.Dead {
		v.Infof("NPC menu: %q has no dead mercenary to revive", npc.Label())

		return
	}

	rows := []d2player.NPCMenuRow{{
		Fallback: fmt.Sprintf("Revive %s  %dg", v.mercDisplayName(info.Rec, int(info.Save.NameID)), info.ReviveCost),
		Action:   d2player.NPCActionReviveMerc,
	}}

	v.gameControls.NPCMenu.Open(npc.Label(), rows, 0, 0, func(row d2player.NPCMenuRow) {
		v.onHireChoice(d2hireling.SellerTyrael, row)
	})
	v.anchorNPCMenu(v.gameControls.NPCMenu, npc)
}

func (v *Game) onHireChoice(seller int, row d2player.NPCMenuRow) {
	switch row.Action {
	case d2player.NPCActionHireOffer:
		if !v.hireGateOpen(seller) {
			v.Infof("MERC hire refused: quest gate of seller %d (HIRE_ProcessHireOffer)", seller)

			break
		}

		if err := v.hireOffer(seller, row.StringID); err != nil {
			v.Infof("MERC hire failed: %v", err)
		}
	case d2player.NPCActionReviveMerc:
		if err := v.reviveMerc(); err != nil {
			v.Infof("MERC revive failed: %v", err)
		}
	default:
		v.Infof("NPC menu: Cancel")
	}

	v.gameControls.NPCMenu.Close()
	v.npcTarget = nil
}

// hireGateOpen is the quest gate of HIRE_ProcessHireOffer (VERIFIED): Qual-Kehk
// needs Rescue on Mount Arreat done, Kashya needs Sisters' Burial Grounds done
// for heroes below level 8, both read from the quest record of the game
// difficulty. The OD2_AUTOMERC scenario calls hireOffer directly and skips it.
func (v *Game) hireGateOpen(seller int) bool {
	p := v.localPlayer
	rt := v.quests()

	return d2hireling.HireAllowed(seller, v.mercDifficulty()-1, p.Stats.Level, func(slot int) bool {
		return rt != nil && rt.g != nil && rt.g.Rec != nil && rt.g.Rec.Get(slot, 0)
	})
}

// hireOffer pays for and spawns the merc of an offer slot (HIRE_ProcessHireOffer:
// the previous merc and its items are discarded). The quest gate of Kashya for
// low levels is not modelled.
func (v *Game) hireOffer(seller, slot int) error {
	o := v.offerTable(seller)
	p := v.localPlayer

	if o == nil || p == nil || v.monsters == nil {
		return errors.New("no offers")
	}

	tab := v.hirelingTable()

	if slot < 0 || slot >= len(o.table.Slots) || !o.table.Slots[slot].Offered || o.table.Slots[slot].Hired {
		return fmt.Errorf("slot %d is not on offer", slot)
	}

	offer, ok := tab.MakeOffer(o.table, slot, p.Stats.Level)
	if !ok {
		return errors.New("no such offer")
	}

	if p.Gold < offer.Cost {
		return fmt.Errorf("not enough gold: %d < %d", p.Gold, offer.Cost)
	}

	save := d2monsters.MercSave{
		ID: offer.Seed, NameID: uint16(offer.NameID), Type: uint16(offer.Rec.ID),
		Experience: tab.StartExp(offer.Rec.ID, offer.Stats.Level),
	}

	before := p.Gold

	if _, err := v.monsters.SpawnMerc(p, save); err != nil {
		return err
	}

	v.gameControls.AddGold(-offer.Cost)

	p.Merc = &d2hero.MercState{ID: save.ID, NameID: save.NameID, Type: save.Type, Experience: save.Experience, Replaced: true}
	v.merc.spawnedFor = v.monsters

	o.table.Slots[slot].Hired = true
	if o.table.Regenerate(tab, o.rng) {
		v.Infof("MERC offers regenerated seller=%d", seller)
	}

	v.Infof("MERC hire name=%q type=%d level=%d cost=%d gold=%d->%d", v.mercDisplayName(offer.Rec, offer.NameID),
		save.Type, offer.Stats.Level, offer.Cost, before, p.Gold)

	return v.OnPlayerSave()
}

// reviveMerc revives the dead merc for min(50000, lvl^2/2*15) gold.
func (v *Game) reviveMerc() error {
	p := v.localPlayer

	info, ok := v.monsters.Merc(p)
	if !ok || !info.Save.Dead {
		return errors.New("no dead mercenary")
	}

	if p.Gold < info.ReviveCost {
		return fmt.Errorf("not enough gold: %d < %d", p.Gold, info.ReviveCost)
	}

	if err := v.monsters.ReviveMerc(p); err != nil {
		return err
	}

	v.gameControls.AddGold(-info.ReviveCost)

	if p.Merc != nil {
		p.Merc.Dead = false
	}

	v.Infof("MERC revive paid cost=%d gold=%d", info.ReviveCost, p.Gold)

	return v.OnPlayerSave()
}

// ---- OD2_AUTOMERC ----

// advanceMercTest implements OD2_AUTOMERC=<1|monster[,count]>: it uses the
// hero's saved merc or hires one at Kashya's table (logging the offers),
// spawns hostile monsters around the hero (the hero stands by, the merc
// fights), optionally kills and revives the merc (OD2_AUTOMERC_KILL=1), walks
// the hero away to log the follow behaviour, then prints a summary.
// OD2_AUTOMERC_SECONDS limits the fight phase; OD2_AUTOEXIT quits afterwards.
func (v *Game) advanceMercTest(elapsed float64) {
	ref := os.Getenv("OD2_AUTOMERC")
	if ref == "" || v.localPlayer == nil || v.monsters == nil {
		return
	}

	t := v.merc.test
	if t == nil {
		if ref == "1" {
			ref = mercTestDefault
		}

		t = &mercTest{ref: ref, count: 1}

		if parts := strings.SplitN(ref, ",", 2); len(parts) == 2 {
			t.ref = parts[0]

			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				t.count = n
			}
		}

		v.merc.test = t
	}

	if t.done {
		return
	}

	t.elapsed += elapsed
	t.phaseT += elapsed

	if t.elapsed < mercTestDelay {
		return
	}

	switch t.phase {
	case 0:
		v.mercTestSetup(t)
	case 1:
		v.mercTestFight(t)
	case 2:
		v.mercTestRevive(t)
	case 3:
		v.mercTestFollow(t, elapsed)
	default:
		v.mercTestFinish(t)
	}
}

func (v *Game) next(t *mercTest) {
	t.phase++
	t.phaseT = 0
}

func (v *Game) mercTestSetup(t *mercTest) {
	p, d := v.localPlayer, v.monsters

	if d.Hirelings() == nil {
		v.Errorf("AUTOMERC: no hireling table")
		t.done = true
		v.autoTestExit()

		return
	}

	if _, ok := d.Merc(p); ok {
		v.Infof("AUTOMERC using the merc of the save")
	} else {
		// hire the first offer of Kashya (class 150) the hero can pay for
		o := v.offerTable(150)
		if o == nil {
			v.Errorf("AUTOMERC: Kashya has no offers")
			t.done = true
			v.autoTestExit()

			return
		}

		tab := v.hirelingTable()
		v.Infof("MERC offers seller=150 npc=\"Kashya\" difficulty=%d owner_level=%d count=%d", v.mercDifficulty(),
			p.Stats.Level, len(o.table.Offered()))

		first := -1

		for _, slot := range o.table.Offered() {
			offer, _ := tab.MakeOffer(o.table, slot, p.Stats.Level)
			v.Infof("MERC offer %s [slot %d, %s]", v.offerLine(offer), slot, offer.Rec.HireDesc)

			if first < 0 {
				first = slot
			}
		}

		offer, _ := tab.MakeOffer(o.table, first, p.Stats.Level)
		if p.Gold < offer.Cost {
			v.gameControls.AddGold(offer.Cost - p.Gold)
			v.Infof("AUTOMERC gave the hero gold to afford the offer (%d)", offer.Cost)
		}

		if err := v.hireOffer(150, first); err != nil {
			v.Errorf("AUTOMERC hire: %v", err)
			t.done = true
			v.autoTestExit()

			return
		}
	}

	info, _ := d.Merc(p)
	v.Infof("AUTOMERC merc name=%q level=%d hp=%d/%d dead=%v", v.mercDisplayName(info.Rec, int(info.Save.NameID)), info.Level,
		info.HP, info.MaxHP, info.Save.Dead)

	if info.Save.Dead { // a dead merc in the save: revive first
		if p.Gold < info.ReviveCost {
			v.gameControls.AddGold(info.ReviveCost - p.Gold)
		}

		if err := v.reviveMerc(); err != nil {
			v.Errorf("AUTOMERC revive: %v", err)
		}
	}

	// OD2_AUTOMERC_HEROLEVEL raises the hero's level so a low-level merc earns experience
	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOMERC_HEROLEVEL")); err == nil && n > 0 {
		p.Stats.Level = n
		v.Infof("AUTOMERC hero level set to %d", n)
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOMERC_EXP")); err == nil && n > 0 {
		d.GrantMercExp(p, uint32(n))
	}

	stat := d.FindStat(t.ref)
	if stat == nil {
		v.Errorf("AUTOMERC: unknown monster %q", t.ref)
		t.done = true
		v.autoTestExit()

		return
	}

	hx, hy := int(p.Position.X()), int(p.Position.Y())
	v.Infof("AUTOMERC start monster=%s count=%d hero=(%d,%d)", stat.Key, t.count, hx, hy)

	var leader *d2mapentity.Monster

	for i := 0; i < t.count; i++ {
		angle := 2 * math.Pi * float64(i) / float64(t.count)
		x := hx + int(math.Round(math.Cos(angle)*monsterTestRing))
		y := hy + int(math.Round(math.Sin(angle)*monsterTestRing))

		m, err := d.SpawnNear(stat, x, y, 2)
		if err != nil {
			v.Errorf("AUTOMERC: spawn failed: %v", err)
			continue
		}

		if leader == nil {
			leader = m
		} else {
			d.Group(leader, m)
		}
	}

	v.next(t)
}

func (v *Game) livingMonsters() int {
	n := 0

	for _, m := range v.monsters.Monsters() {
		if m.Alive() {
			n++
		}
	}

	return n
}

func (v *Game) mercTestFight(t *mercTest) {
	limit := mercTestFightSecs
	if secs, err := strconv.ParseFloat(os.Getenv("OD2_AUTOMERC_SECONDS"), 64); err == nil && secs > 0 {
		limit = secs
	}

	if v.livingMonsters() == 0 || t.phaseT > limit {
		v.Infof("AUTOMERC fight over alive_monsters=%d after=%.1fs", v.livingMonsters(), t.phaseT)

		if os.Getenv("OD2_AUTOMERC_KILL") != "" {
			v.monsters.KillMerc(v.localPlayer, "autotest")
		}

		v.next(t)
	}
}

func (v *Game) mercTestRevive(t *mercTest) {
	info, ok := v.monsters.Merc(v.localPlayer)
	if !ok || !info.Save.Dead {
		v.next(t)
		t.startX, t.startY = v.localPlayer.GetPositionF()

		return
	}

	if t.phaseT < 2 { // let the death animation play
		return
	}

	if os.Getenv("OD2_AUTOMERC_KILL") == "dead" { // leave the merc dead: the save must show the dead flag
		v.Infof("AUTOMERC leaving the merc dead")
		t.phase = 4

		return
	}

	p := v.localPlayer
	if p.Gold < info.ReviveCost {
		v.gameControls.AddGold(info.ReviveCost - p.Gold)
	}

	if err := v.reviveMerc(); err != nil {
		v.Errorf("AUTOMERC revive: %v", err)
	}

	v.next(t)
	t.startX, t.startY = p.GetPositionF()
}

func (v *Game) mercTestFollow(t *mercTest, elapsed float64) {
	p := v.localPlayer
	info, ok := v.monsters.Merc(p)

	if t.phaseT == elapsed {
		// walk away from the merc to see it follow
		if x, y, found := v.monsters.FarPoint(p, mercTestFollowGap); found {
			v.Infof("AUTOMERC follow: hero walks about %d subtiles away to tile (%.0f,%.0f)", mercTestFollowGap, x, y)
			v.OnPlayerMove(x, y)
		} else {
			v.Infof("AUTOMERC follow: no reachable point %d subtiles away", mercTestFollowGap)
		}
	}

	t.logAcc += elapsed
	if t.logAcc >= 1 {
		t.logAcc = 0

		if ok {
			mx, my := v.monsters.MercPosition(p)
			hx, hy := int(p.Position.X()), int(p.Position.Y())
			v.Infof("AUTOMERC follow dist=%d merc=(%d,%d) hero=(%d,%d) hp=%d/%d", d2monster.Distance(mx-hx, my-hy),
				mx, my, hx, hy, info.HP, info.MaxHP)
		}
	}

	if t.phaseT > 12 {
		v.next(t)
	}
}

func (v *Game) mercTestFinish(t *mercTest) {
	c := v.monsters.Counters
	info, _ := v.monsters.Merc(v.localPlayer)

	v.Infof("AUTOMERC summary spawns=%d attacks=%d hits=%d skills=%d deaths=%d revives=%d teleports=%d levelups=%d monster_deaths=%d "+
		"monster_attacks=%d merc_level=%d merc_exp=%d merc_hp=%d/%d hero_hp=%d/%d gold=%d",
		c.MercSpawns, c.MercAttacks, c.MercHits, c.MercSkills, c.MercDeaths, c.MercRevives, c.MercTeleports, c.MercLevelUps,
		c.Deaths, c.Attacks, info.Level, info.Save.Experience, info.HP, info.MaxHP,
		v.localPlayer.Stats.Health, v.localPlayer.Stats.MaxHealth, v.localPlayer.Gold)

	t.done = true

	v.autoTestExit()
}
