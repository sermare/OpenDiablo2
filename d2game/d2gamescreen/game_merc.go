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
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
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
	// carry is what the merc takes along from a level to the next: the hit
	// points it had (a unit that follows its owner is moved, not made anew, so
	// it is not healed), and whether it was dead (a dead merc stays behind).
	carry mercCarry
	// gameRng is the generator the offer tables draw from (see offerTable)
	gameRng *d2rand.Seed
}

type mercCarry struct {
	valid bool
	hp    int
	dead  bool
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
	// the OD2_AUTOMERC_LEVELS phases (levelsPhase)
	level   int // the level the merc is taken to
	home    int // the level it comes back to
	hpLeave int
	// OD2_AUTOMERC_TYPES: hireling ids hired one after the other, each fights
	// a while (mercTestTypes)
	types  []int
	probed bool
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

	// OD2_NOMERC=1 leaves the merc home (the realm scenario counts the kills of the hero alone)
	soloTest = soloTest || os.Getenv("OD2_NOMERC") != ""

	if v.merc.spawnedFor != d && p.Merc != nil && d.Hirelings() != nil && !soloTest {
		v.merc.spawnedFor = v.monsters
		v.arriveMerc(d, p)
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
		Drink:      func(e d2inventory.PotionEffect) { v.monsters.DrinkMerc(p, e) },
		DrinkState: func(sp d2inventory.StatePotion) { v.monsters.DrinkMercState(p, sp) },
		View:       func() (d2player.MercView, bool) { return v.mercView(p) },
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

// mercView is the snapshot the mercenary panel shows: the living merc of this level, or the dead one
// (here or left behind in another level).
func (v *Game) mercView(p *d2mapentity.Player) (d2player.MercView, bool) {
	if p == nil || p.Merc == nil || v.monsters == nil {
		return d2player.MercView{}, false
	}

	info, ok := v.monsters.Merc(p)
	if !ok {
		info, ok = v.deadMerc()
		if !ok {
			return d2player.MercView{}, false
		}
	}

	return d2player.MercView{
		Name: v.mercDisplayName(info.Rec, int(info.Save.NameID)), Level: info.Level,
		Exp: int(info.Save.Experience), NextExp: info.Stats.NextXP,
		Str: info.Stats.Str, Dex: info.Stats.Dex, DmgMin: info.Stats.DmgMin, DmgMax: info.Stats.DmgMax,
		Defense: info.Stats.Defense, Resist: info.Gear.Resist, HP: info.HP, MaxHP: info.MaxHP, Dead: info.Save.Dead, Class: info.Rec.Class, TypeName: v.tr(info.Rec.Hireling),
	}, true
}

// arriveMerc brings the hero's merc into a freshly built level. A living merc
// follows the hero through the exit, the stairs, the waypoint or the town
// portal and arrives next to him with the hit points it had (MERC_RelocatePetsWithOwner
// moves living pets only); a merc that died stays behind: it has no unit in
// the new level until a hireling NPC revives it. The first spawn of a game
// (from the save) gives a dead merc a corpse next to the hero and a living one
// full life.
func (v *Game) arriveMerc(d *d2monsters.Director, p *d2mapentity.Player) {
	c := v.merc.carry
	save := saveOf(p.Merc)
	if v.gameControls != nil {
		save.Gear = v.gameControls.MercStatItems() // the saved 'jf' items count from the first frame
	}

	spawn, hp := mercArrival(c, save.Dead)
	if !spawn {
		v.Infof("MERC stays behind dead (revive it at a mercenary vendor) level=%d", v.currentLevel())
		return
	}

	if _, err := d.SpawnMercHP(p, save, hp); err != nil {
		v.Errorf("MERC spawn: %v", err)
		return
	}

	if c.valid {
		info, _ := d.Merc(p)
		v.Infof("MERC arrive level=%d hp=%d/%d (carried hp=%d) pos_hero=(%.0f,%.0f)", v.currentLevel(), info.HP, info.MaxHP, hp,
			p.Position.X(), p.Position.Y())
	}
}

// mercArrival decides what happens to the merc in a freshly built level: a
// merc that travelled (carry valid) and was dead stays behind; a living one
// arrives with the life it had (hp 0 means full life: the first spawn of a game,
// where a dead merc from the save gets its corpse next to the hero).
func mercArrival(c mercCarry, saveDead bool) (spawn bool, hp int) {
	if c.valid && (c.dead || saveDead) {
		return false, 0
	}

	if c.valid {
		return true, c.hp
	}

	return true, 0
}

// captureMerc records the merc's state before a level change drops the
// director that holds it.
func (v *Game) captureMerc() {
	p, d := v.localPlayer, v.monsters

	if p == nil || d == nil || p.Merc == nil {
		return
	}

	if info, ok := d.Merc(p); ok {
		p.Merc.Experience, p.Merc.Dead = info.Save.Experience, info.Save.Dead
		v.merc.carry = mercCarry{valid: true, hp: info.HP, dead: info.Save.Dead}
		v.Infof("MERC leaves its level hp=%d/%d dead=%v exp=%d", info.HP, info.MaxHP, info.Save.Dead,
			info.Save.Experience)

		return
	}

	// no unit (it stayed behind dead in an earlier level): the save's flags carry on
	v.merc.carry = mercCarry{valid: true, dead: p.Merc.Dead}
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

	// NPCSRV_AllocHireOfferTable draws every slot seed and the ten offered slots
	// from the game's one seed (game+0x1d24, an LCG with the multiplier
	// 0x6ac690c5: V in hirelings.md), so the vendors of a game share one stream
	// and what a vendor offers depends on the order the hero meets them. The
	// exe's stream is also used by other game logic before the first visit, so
	// its exact state at that moment is not reproducible here: a generator
	// seeded with the game seed stands in (UNVERIFIED).
	if v.merc.gameRng == nil {
		v.merc.gameRng = d2rand.New(uint32(v.gameClient.MapEngine.Seed()))
	}

	rng := v.merc.gameRng

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

// tr translates a string.tbl key (the key itself when it has no text).
func (v *Game) tr(key string) string {
	if s := v.asset.TranslateString(key); s != "" {
		return s
	}

	return key
}

// menuLine is the text of one row of the hire list in the form of the exe's dialog (UI_OpenMercHireDialog 0x4b24c0):
// "<name> - Lvl: N  Life: N  Def: N  Cost: N", then the HireDesc label of the row.
func (v *Game) menuLine(r d2hireling.MenuRow) string {
	line, desc := v.menuLines(r)
	if desc != "" {
		line += "  (" + desc + ")"
	}

	return line
}

// menuLines are the two lines of a list entry of the exe's dialog: the numbers and the HireDesc label ("" for none).
func (v *Game) menuLines(r d2hireling.MenuRow) (line, desc string) {
	name := r.NameKey
	if s := v.asset.TranslateString(r.NameKey); s != "" && s != r.NameKey {
		name = s
	}

	return d2player.HireDialogLine(v.tr, name, r.Level, r.HP, r.Defense, r.Price), d2player.HireDialogDesc(v.tr, r.HireDesc)
}

// hireEntries gives the entries of the list box: two lines per offer (the numbers, then the HireDesc label) and the revive row.
func (v *Game) hireEntries(seller int, rows []d2player.NPCMenuRow) []d2player.HireListEntry {
	o := v.offerTable(seller)
	if o == nil {
		return nil
	}

	menu := v.hirelingTable().HireMenu(o.table, v.localPlayer.Stats.Level, nil)
	entries := make([]d2player.HireListEntry, 0, len(rows))

	for i, row := range rows {
		e := d2player.HireListEntry{Line1: row.Fallback, Row: row}
		if row.Action == d2player.NPCActionHireOffer && i < len(menu) {
			e.Line1, e.Line2 = v.menuLines(menu[i])
		}

		entries = append(entries, e)
	}

	return entries
}

func skillNames(sk []d2hireling.SkillAt) string {
	out := ""

	for i, s := range sk {
		if i > 0 {
			out += "+"
		}

		out += fmt.Sprintf("%s:%d", s.Name, s.Level)
	}

	return out
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

	title, rows, lines, ok := v.hireDialog(seller)
	if !ok {
		v.Infof("NPC menu: Hire at %q has no offers (class %d)", npc.Label(), seller)
		return
	}

	// the exe replaces the NPC menu by a list box dialog (UI_OpenMercHireDialog 0x4b24c0); Cancel returns to the NPC menu
	v.gameControls.NPCMenu.Close()
	v.gameControls.HireList.Open(title, v.tr(d2player.HireKeyCancel), v.hireEntries(seller, rows), func(row d2player.NPCMenuRow) {
		v.onHireChoice(seller, row)
	})

	v.Infof("MERC hire dialog title=%q rows=%d", title, len(lines))
	v.Infof("MERC offers seller=%d npc=%q difficulty=%d owner_level=%d count=%d", seller, npc.Label(), v.mercDifficulty(),
		v.localPlayer.Stats.Level, len(lines))

	for _, l := range lines {
		v.Infof("MERC offer %s", l)
	}
}

// hireDialog builds what the hire list of a seller shows: its title, its rows (the offers, then the revive row of a dead
// merc) and a log line per offer.
func (v *Game) hireDialog(seller int) (title string, rows []d2player.NPCMenuRow, lines []string, ok bool) {
	o := v.offerTable(seller)
	if o == nil || v.localPlayer == nil {
		return "", nil, nil, false
	}

	tab := v.hirelingTable()

	for _, r := range tab.HireMenu(o.table, v.localPlayer.Stats.Level, nil) {
		line := v.menuLine(r)
		rows = append(rows, d2player.NPCMenuRow{StringID: r.Slot, Fallback: line, Action: d2player.NPCActionHireOffer})
		lines = append(lines, fmt.Sprintf("%s [slot %d, %s, hp=%d def=%d dmg=%d-%d skills=%s]", line, r.Slot, r.HireDesc,
			r.HP, r.Defense, r.DmgMin, r.DmgMax, skillNames(r.Skills)))
	}

	if info, ok := v.deadMerc(); ok {
		rows = append(rows, d2player.NPCMenuRow{
			Fallback: v.reviveLine(info),
			Action:   d2player.NPCActionReviveMerc,
		})
	}

	// the exe titles the dialog with the hero's gold and says so when nothing is left to hire (stash gold, stat 0xf, is not
	// added: the game does not carry it yet)
	title = d2player.HireDialogTitle(v.tr, v.localPlayer.Gold)
	if len(lines) == 0 {
		title = d2player.HireDialogEmpty(v.tr)
	}

	return title, rows, lines, true
}

// reviveLine is the revive row of the hire list: string hireresurrect2 "Resurrect %s: %d" with the merc's name and the price.
func (v *Game) reviveLine(info d2monsters.MercInfo) string {
	return d2player.ReviveLine(v.tr("hireresurrect2"), v.mercDisplayName(info.Rec, int(info.Save.NameID)), info.ReviveCost)
}

// openRevive shows only the revive row (Tyrael).
func (v *Game) openRevive(npc d2interface.MapEntity) {
	info, ok := v.monsters.Merc(v.localPlayer)
	if !ok || !info.Save.Dead {
		v.Infof("NPC menu: %q has no dead mercenary to revive", npc.Label())

		return
	}

	rows := []d2player.NPCMenuRow{{
		Fallback: v.reviveLine(info),
		Action:   d2player.NPCActionReviveMerc,
	}}

	v.gameControls.NPCMenu.Open(npc.Label(), rows, 0, 0, func(row d2player.NPCMenuRow) {
		v.onHireChoice(d2hireling.SellerTyrael, row)
	})
	v.anchorNPCMenu(v.gameControls.NPCMenu, npc)
}

// errNotEnoughGold is returned by hireOfferFor when the hero cannot pay (reply code 12 of the exe, string 0xd22).
var errNotEnoughGold = errors.New("not enough gold")

func (v *Game) onHireChoice(seller int, row d2player.NPCMenuRow) { v.hireChoice(seller, row, false) }

// hireChoice is a click on an entry of the hire list (confirmed: the answer Yes of the replace dialog).
func (v *Game) hireChoice(seller int, row d2player.NPCMenuRow, confirmed bool) {
	switch row.Action {
	case d2player.NPCActionHireOffer:
		if !v.hireGateOpen(seller) {
			v.Infof("MERC hire refused: quest gate of seller %d (HIRE_ProcessHireOffer)", seller)

			break
		}

		if !confirmed && d2player.HireClick(v.localPlayer != nil && v.localPlayer.Merc != nil) == d2player.HireConfirmReplace {
			v.confirmHireReplace(seller, row)
			v.Infof("MERC hire click slot=%d: confirm %q", row.StringID, v.tr(d2player.HireKeyReplace))

			return
		}

		err := v.hireOffer(seller, row.StringID)

		switch {
		case err == nil:
		case errors.Is(err, errNotEnoughGold):
			v.Infof("MERC hire failed: %v", err)
			v.hireMessage(seller, d2player.HireReplyNoGold)

			return
		default:
			v.Infof("MERC hire failed: %v", err)
		}
	case d2player.NPCActionReviveMerc:
		if err := v.reviveMerc(); err != nil {
			v.Infof("MERC revive failed: %v", err)
		}
	default:
		v.Infof("NPC menu: Cancel")

		// Cancel and "No" return to the NPC menu (UI_ReturnToNpcMenu 0x4b2480)
		if npc := v.npcTarget; npc != nil && seller != d2hireling.SellerTyrael {
			v.gameControls.HireList.Close()
			v.openNPCMenu(v.gameControls.NPCMenu, npc)

			return
		}
	}

	v.gameControls.NPCMenu.Close()
	v.gameControls.HireList.Close()
	v.npcTarget = nil
}

// confirmHireReplace is the dialog of mode 5 (UI_CreateNpcItemConfirmDialog 0x4af750): the title says the new mercenary replaces the
// current one, the answers are Yes and No.
func (v *Game) confirmHireReplace(seller int, row d2player.NPCMenuRow) {
	yes := row
	yes.Fallback = v.tr(d2player.HireKeyYes)
	no := d2player.NPCMenuRow{Fallback: v.tr(d2player.HireKeyNo), Action: d2player.NPCActionCancel}

	v.gameControls.HireList.Close()
	v.gameControls.NPCMenu.OpenPrompt(v.tr(d2player.HireKeyReplace), []d2player.NPCMenuRow{yes, no}, func(r d2player.NPCMenuRow) {
		v.gameControls.NPCMenu.Close()
		v.hireChoice(seller, r, true)
	})

	if v.npcTarget != nil {
		v.anchorNPCMenu(v.gameControls.NPCMenu, v.npcTarget)
	}
}

// hireMessage shows the text of a reply code of the hire request with a Continue button (UI_ApplyNpcTransactionResultB 0x4b1ea0).
func (v *Game) hireMessage(seller, code int) {
	key := d2player.HireReplyKey(code)
	if key == "" {
		return
	}

	text := v.tr(key)
	ok := d2player.NPCMenuRow{Fallback: v.tr(d2player.HireKeyOK), Action: d2player.NPCActionCancel}

	v.gameControls.HireList.Close()
	v.gameControls.NPCMenu.OpenPrompt(text, []d2player.NPCMenuRow{ok}, func(r d2player.NPCMenuRow) {
		v.gameControls.NPCMenu.Close()
		v.hireChoice(seller, r, true)
	})
	v.Infof("MERC hire message code=%d text=%q", code, text)

	if v.npcTarget != nil {
		v.anchorNPCMenu(v.gameControls.NPCMenu, v.npcTarget)
	}
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
func (v *Game) hireOffer(seller, slot int) error { return v.hireOfferFor(seller, slot, false) }

// hireOfferFor is hireOffer; free hires without paying (a quest reward: Kashya's Sisters' Burial Grounds, Asheara's
// Blade of the Old Religion, D2MOO D2GAME_NPC_AssignMercenary, called by the exe at 0x5b6ea0 for Asheara).
func (v *Game) hireOfferFor(seller, slot int, free bool) error {
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

	if free {
		offer.Cost = 0
	}

	if p.Gold < offer.Cost {
		return fmt.Errorf("%w: %d < %d", errNotEnoughGold, p.Gold, offer.Cost)
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
	v.merc.carry = mercCarry{}

	o.table.Slots[slot].Hired = true
	if o.table.Regenerate(tab, o.rng) {
		v.Infof("MERC offers regenerated seller=%d", seller)
	}

	v.Infof("MERC hire name=%q type=%d level=%d cost=%d gold=%d->%d", v.mercDisplayName(offer.Rec, offer.NameID),
		save.Type, offer.Stats.Level, offer.Cost, before, p.Gold)

	return v.OnPlayerSave()
}

// grantQuestMerc is the free mercenary a quest gives a hero who has none: the first one the seller offers. UNVERIFIED
// which of the offers the original picks (D2MOO: the first available record of the vendor).
func (v *Game) grantQuestMerc(seller int) {
	p := v.localPlayer
	if p == nil || v.monsters == nil || p.Merc != nil {
		return
	}

	o := v.offerTable(seller)
	if o == nil || v.hirelingTable() == nil {
		return
	}

	for slot := range o.table.Slots {
		if !o.table.Slots[slot].Offered || o.table.Slots[slot].Hired {
			continue
		}

		if err := v.hireOfferFor(seller, slot, true); err != nil {
			v.Infof("MERC quest reward not given: %v", err)
		} else {
			v.Infof("MERC quest reward: seller %d gives a mercenary for free", seller)
		}

		return
	}
}

// deadMerc describes the hero's dead merc: the corpse in this level, or the
// one that stayed behind in an earlier level (it has no unit here, only the
// save's flags).
func (v *Game) deadMerc() (d2monsters.MercInfo, bool) {
	p := v.localPlayer
	if p == nil || v.monsters == nil {
		return d2monsters.MercInfo{}, false
	}

	if info, ok := v.monsters.Merc(p); ok {
		return info, info.Save.Dead
	}

	tab := v.hirelingTable()
	if p.Merc == nil || !p.Merc.Dead || tab == nil {
		return d2monsters.MercInfo{}, false
	}

	save := saveOf(p.Merc)
	level := tab.LevelFromExp(int(save.Type), save.Experience)
	st, rec := tab.StatsFor(int(save.Type), level)

	if rec == nil {
		return d2monsters.MercInfo{}, false
	}

	return d2monsters.MercInfo{Save: save, Level: level, MaxHP: st.MaxHP, Rec: rec, Stats: st,
		ReviveCost: d2hireling.ReviveCost(level)}, true
}

// reviveMerc revives the dead merc for min(50000, lvl^2/2*15) gold. A merc
// that stayed behind in another level comes back with a fresh unit next to
// the hero (HIRE_ServerHandleReviveMercenary, MERC_ReviveUnit: full life, at
// the owner).
func (v *Game) reviveMerc() error {
	p := v.localPlayer

	info, ok := v.deadMerc()
	if !ok {
		return errors.New("no dead mercenary")
	}

	if p.Gold < info.ReviveCost {
		return fmt.Errorf("not enough gold: %d < %d", p.Gold, info.ReviveCost)
	}

	if _, has := v.monsters.Merc(p); has {
		if err := v.monsters.ReviveMerc(p); err != nil {
			return err
		}
	} else {
		save := info.Save
		save.Dead = false

		if _, err := v.monsters.SpawnMerc(p, save); err != nil {
			return err
		}

		v.merc.spawnedFor = v.monsters
		v.Infof("MERC revive: a new unit for the merc that stayed behind level=%d", info.Level)
	}

	v.merc.carry = mercCarry{}

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

	// the long scenarios keep the hero alive: a hardcore hero who dies ends the game
	if os.Getenv("OD2_AUTOMERC_TYPES") != "" || os.Getenv("OD2_AUTOMERC_LEVELS") != "" {
		if p := v.localPlayer; p.Stats.Health < p.Stats.MaxHealth/2 {
			p.Stats.Health = p.Stats.MaxHealth
		}
	}

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
	case 4:
		v.mercTestFinish(t)
	case mercTypesFirst, mercTypesFirst + 1:
		v.mercTestTypes(t)
	case mercLevelsFirst, mercLevelsFirst + 1, mercLevelsFirst + 2, mercLevelsFirst + 3, mercLevelsFirst + 4, mercLevelsFirst + 5:
		v.mercTestLevels(t)
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

	if os.Getenv("OD2_AUTOMERC_UI") != "" {
		v.mercTestUI()
	}

	if !v.mercTestSpawn(t) {
		return
	}

	v.next(t)
}

// mercTestUI is OD2_AUTOMERC_UI=1: the hire lists of the four hirelings' vendors, the portrait and the potion dropped on it.
func (v *Game) mercTestUI() {
	p, d := v.localPlayer, v.monsters

	for _, seller := range []int{d2hireling.SellerKashya, d2hireling.SellerGreiz, d2hireling.SellerAsheara, d2hireling.SellerQualKehk} {
		title, rows, _, ok := v.hireDialog(seller)
		first := ""

		if len(rows) > 0 {
			first = rows[0].Fallback
		}

		v.Infof("AUTOMERC ui: hire seller=%d ok=%v title=%q rows=%d first=%q", seller, ok, title, len(rows), first)
	}

	v.mercTestHireClicks()

	info, ok := d.Merc(p)
	if !ok {
		v.Infof("AUTOMERC ui: no living merc")

		return
	}

	d.SetMercHP(p, info.MaxHP*3/10)
	hurt, _ := d.Merc(p)
	fill, _, drawn := d2player.MercPortraitBar(hurt.HP, hurt.MaxHP)
	v.Infof("AUTOMERC ui: merc hurt hp=%d/%d portrait_bar=%d/46 drawn=%v", hurt.HP, hurt.MaxHP, fill, drawn)

	v.gameControls.AutoMercUI(func(format string, a ...interface{}) { v.Infof("AUTOMERC ui: "+format, a...) })

	healed, _ := d.Merc(p)
	v.Infof("AUTOMERC ui: potion hp %d -> %d healed=%v", hurt.HP, healed.HP, healed.HP > hurt.HP)
}

// mercTestHireClicks drives the click flow of the hire list (w44): with a mercenary the click asks for the replace confirmation
// (answer No leaves everything as it was), and a hero without the gold gets the not-enough-gold message.
func (v *Game) mercTestHireClicks() {
	p := v.localPlayer
	_, rows, _, ok := v.hireDialog(d2hireling.SellerKashya)

	if !ok || len(rows) == 0 || p == nil {
		v.Infof("AUTOMERC ui: hire click: no offers")

		return
	}

	menu := v.gameControls.NPCMenu
	had := p.Merc != nil

	if !had {
		v.Infof("AUTOMERC ui: hire click: no mercenary to replace, skipped")

		return
	}

	v.hireChoice(d2hireling.SellerKashya, rows[0], false)

	asked := menu.IsOpen() && len(menu.Rows()) == 2 && menu.RowLabel(menu.Rows()[0]) == v.tr(d2player.HireKeyYes)
	v.Infof("AUTOMERC ui: hire click had_merc=%v confirm_open=%v rows=%d title=%q", had, asked, len(menu.Rows()), v.tr(d2player.HireKeyReplace))

	if asked {
		menu.Choose(1) // No
	}

	v.Infof("AUTOMERC ui: hire click answered No: menu_open=%v gold=%d", menu.IsOpen(), p.Gold)

	gold := p.Gold
	p.Gold = 0

	v.hireChoice(d2hireling.SellerKashya, rows[0], true)
	v.Infof("AUTOMERC ui: hire click broke: message_open=%v rows=%d", menu.IsOpen(), len(menu.Rows()))
	menu.Close()
	v.gameControls.HireList.Close()

	p.Gold = gold
	v.npcTarget = nil
}

// mercTestSpawn puts the scenario's hostile monsters in a ring around the hero.
func (v *Game) mercTestSpawn(t *mercTest) bool {
	p, d := v.localPlayer, v.monsters

	stat := d.FindStat(t.ref)
	if stat == nil {
		v.Errorf("AUTOMERC: unknown monster %q", t.ref)
		t.done = true
		v.autoTestExit()

		return false
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

	return true
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
		v.afterFollow(t)
	}
}

// afterFollow chooses the phase after the follow phase: the hireling types,
// the level changes, or the end.
func (v *Game) afterFollow(t *mercTest) {
	for _, f := range strings.Split(os.Getenv("OD2_AUTOMERC_TYPES"), ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
			t.types = append(t.types, n)
		}
	}

	switch {
	case len(t.types) > 0:
		t.phase, t.phaseT = mercTypesFirst, 0
	case os.Getenv("OD2_AUTOMERC_LEVELS") != "":
		t.phase, t.phaseT = mercLevelsFirst, 0
	default:
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

// ---- OD2_AUTOMERC_LEVELS: the merc through level changes ----

// The phases run after the follow phase when OD2_AUTOMERC_LEVELS is set:
//
//	+0 hurt the merc to half life and take the hero to OD2_AUTOMERC_LEVEL (default 29, a
//	   waypoint level; with OD2_REALMAPS=1 that is a real DRLG map)
//	+1 wait for the arrival: the merc must come along with the life it had
//	+2 fight monsters spawned around the hero in the new level, then kill the merc
//	+3 take the hero back to town (OD2_AUTOMERC_HOME, default 1)
//	+4 the dead merc must have stayed behind; revive it at the "vendor"
//	+5 the revived merc is at full life next to the hero; finish
const mercLevelsFirst = 10

func (v *Game) mercTestLevels(t *mercTest) {
	p, d := v.localPlayer, v.monsters

	switch t.phase - mercLevelsFirst {
	case 0:
		t.level, t.home = mercEnvInt("OD2_AUTOMERC_LEVEL", 29), mercEnvInt("OD2_AUTOMERC_HOME", 1)

		d.ClearHostiles()

		info, ok := d.Merc(p)
		if !ok {
			v.Errorf("AUTOMERC levels: the hero has no merc")
			v.finishLevels(t)

			return
		}

		if info.Save.Dead { // the last hireling of the types phase died
			if err := d.ReviveMerc(p); err != nil {
				v.Errorf("AUTOMERC levels: revive: %v", err)
			}

			info, _ = d.Merc(p)
		}

		d.SetMercHP(p, info.MaxHP/2)
		info, _ = d.Merc(p)
		t.hpLeave = info.HP
		v.Infof("AUTOMERC levels: merc hp=%d/%d, taking the hero from level %d to level %d", info.HP, info.MaxHP, v.currentLevel(), t.level)

		if !v.startLevelChange(t.level, d2level.WaypointStartType(t.level), "waypoint") {
			v.Errorf("AUTOMERC levels: level %d cannot be entered", t.level)
			v.finishLevels(t)

			return
		}

		v.next(t)
	case 1:
		if v.levels.trans != nil || v.monsters == nil || v.currentLevel() != t.level {
			return
		}

		info, ok := v.monsters.Merc(p)
		if !ok {
			if t.phaseT > 5 {
				v.Errorf("AUTOMERC levels: the merc did not arrive in level %d", t.level)
				v.finishLevels(t)
			}

			return
		}

		v.Infof("AUTOMERC levels: arrived level=%d merc hp=%d/%d expected_hp=%d alive=%v", t.level, info.HP, info.MaxHP, t.hpLeave,
			!info.Save.Dead)

		v.mercTestSpawn(t)
		v.next(t)
	case 2:
		limit := 25.0
		if v.livingMonsters() == 0 || t.phaseT > limit {
			c := v.monsters.Counters
			v.Infof("AUTOMERC levels: fight over level=%d alive_monsters=%d merc_attacks=%d merc_hits=%d merc_skills=%d", t.level,
				v.livingMonsters(), c.MercAttacks, c.MercHits, c.MercSkills)
			v.monsters.KillMerc(p, "autotest-levels")
			v.next(t)
		}
	case 3:
		if t.phaseT < 2 { // let the death animation play
			return
		}

		if !v.startLevelChange(t.home, d2level.StartPortal, "portal") {
			v.Errorf("AUTOMERC levels: level %d cannot be entered", t.home)
			v.finishLevels(t)

			return
		}

		v.next(t)
	case 4:
		if v.levels.trans != nil || v.monsters == nil || v.currentLevel() != t.home || t.phaseT < 1.5 {
			return
		}

		if _, has := v.monsters.Merc(p); has {
			v.Errorf("AUTOMERC levels: a dead merc followed the hero to level %d", t.home)
		} else {
			v.Infof("AUTOMERC levels: the dead merc stayed behind, level=%d", t.home)
		}

		if info, ok := v.deadMerc(); ok {
			if p.Gold < info.ReviveCost {
				v.gameControls.AddGold(info.ReviveCost - p.Gold)
			}

			if err := v.reviveMerc(); err != nil {
				v.Errorf("AUTOMERC levels: revive: %v", err)
			}
		} else {
			v.Errorf("AUTOMERC levels: no dead merc to revive")
		}

		v.next(t)
	default:
		if t.phaseT < 1.5 {
			return
		}

		if info, ok := v.monsters.Merc(p); ok {
			v.Infof("AUTOMERC levels: revived level=%d merc hp=%d/%d alive=%v", v.currentLevel(), info.HP, info.MaxHP, !info.Save.Dead)
		} else {
			v.Errorf("AUTOMERC levels: the revived merc has no unit")
		}

		v.finishLevels(t)
	}
}

func (v *Game) finishLevels(t *mercTest) { t.phase, t.phaseT = 4, 0 }

func mercEnvInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}

	return def
}

// ---- OD2_AUTOMERC_TYPES: every kind of hireling fights ----

const (
	mercTypesFirst = 20
	mercTypeSecs   = 14.0
)

// mercTestTypes hires the listed hireling ids one after the other, lets each
// fight the scenario's monsters for a while and logs what its skills did
// through the skill engine.
func (v *Game) mercTestTypes(t *mercTest) {
	p, d := v.localPlayer, v.monsters
	tab := v.hirelingTable()

	if t.phase == mercTypesFirst {
		if len(t.types) == 0 {
			v.leaveTypes(t)

			return
		}

		id := t.types[0]
		t.types = t.types[1:]

		d.ClearHostiles()

		level := mercEnvInt("OD2_AUTOMERC_MERCLEVEL", 40)
		if p.Stats.Level < level+1 {
			p.Stats.Level = level + 1
		}

		save := d2monsters.MercSave{ID: uint32(0x1000 + id), Type: uint16(id), Experience: tab.StartExp(id, level)}

		if _, err := d.SpawnMerc(p, save); err != nil {
			v.Errorf("AUTOMERC types: hireling %d: %v", id, err)

			return
		}

		p.Merc = &d2hero.MercState{ID: save.ID, Type: save.Type, Experience: save.Experience, Replaced: true}
		v.merc.spawnedFor = d
		info, _ := d.Merc(p)

		v.Infof("AUTOMERC types: hired id=%d class=%d level=%d skills=%s", id, info.Rec.Class, info.Level, v.mercSkillList(info.Rec))
		t.hpLeave = d.Counters.MercSkills
		v.mercTestSpawn(t)
		v.next(t)

		return
	}

	if !t.probed {
		// every skill of the hireling once, through the engine
		t.probed = true

		for _, m := range d.Monsters() {
			if m.Alive() {
				v.Infof("AUTOMERC types: probe %s", strings.Join(d.ProbeMercSkills(p, m), " "))

				break
			}
		}
	}

	if v.livingMonsters() == 0 || t.phaseT > mercTypeSecs {
		t.probed = false
		c := d.Counters
		sk := v.skillEngine()

		v.Infof("AUTOMERC types: fight over merc_attacks=%d merc_hits=%d merc_skills=%d (this type %d) casts=%d missiles=%d "+
			"missile_hits=%d melee=%d damage=%d", c.MercAttacks, c.MercHits, c.MercSkills, c.MercSkills-t.hpLeave,
			sk.Counters.Casts, sk.Counters.Missiles, sk.Counters.Hits, sk.Counters.Melee, sk.Counters.Damage)
		t.phase, t.phaseT = mercTypesFirst, 0
	}
}

func (v *Game) leaveTypes(t *mercTest) {
	if os.Getenv("OD2_AUTOMERC_LEVELS") != "" {
		t.phase, t.phaseT = mercLevelsFirst, 0

		return
	}

	t.phase, t.phaseT = 4, 0
}

// mercSkillList names the hireling's six skills.
func (v *Game) mercSkillList(rec *d2hireling.Record) string {
	var out []string

	for _, sk := range rec.Skills {
		if sk.Name != "" {
			out = append(out, fmt.Sprintf("%s/mode%d", sk.Name, sk.Mode))
		}
	}

	return strings.Join(out, ",")
}
