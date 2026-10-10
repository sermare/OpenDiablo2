package d2gamescreen

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// The social layer of a network game: party, trade and player versus player.
// The server (package d2server, social.go) owns the roster and the trades; the
// game screen sends commands (console commands, the party panel, the trade
// window) and applies what the server answers.

// socialState is the game screen's part of it.
type socialState struct {
	hooked       bool
	tradePartner string // name of the trade partner while a trade window is open
	tradeOpen    bool
}

// me is the local player's id.
func (v *Game) me() string { return v.gameClient.PlayerID }

// relationOf is how the local hero sees another player (the roster's relation).
func (v *Game) relationOf(p *d2mapentity.Player) d2enum.PlayersRelationships {
	return v.gameClient.Roster.Relation(v.me(), p.ID())
}

// hookNetwork connects the client's packet hooks (at once: the roster and the
// invitations arrive while the level is still loading).
func (v *Game) hookNetwork() {
	v.gameClient.OnTrade = v.onTrade
	v.gameClient.OnPvPHit = v.onPvPHit
	v.gameClient.OnPartyXP = v.onPartyXP
	v.gameClient.OnRoster = v.onRoster
	v.hookPortals()
	v.hookRealm()
}

// advanceSocial connects the screen's pieces to the roster once they exist.
func (v *Game) advanceSocial(_ float64) {
	if v.social.hooked || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	v.social.hooked = true

	v.gameControls.SetTradeHandler(v)
	v.gameControls.OnTownPortal = v.useTownPortalItem
	v.syncPortals() // pairs that arrived while the map was still loading
	v.gameControls.SetRelationSource(v.relationOf)

	if pp := v.gameControls.PartyPanel; pp != nil {
		pp.SetHooks(d2playerPartyHooks(v))
	}

	v.onRoster("") // the roster arrived before the hooks: log where it stands
}

// ---- console commands ----

func (v *Game) resolveTarget(name string) (string, error) {
	if v.gameClient.IsSinglePlayer() {
		return "", errors.New("there is nobody else in a single player game")
	}

	return v.gameClient.ResolvePlayer(name)
}

// commandParty is "party invite|accept|decline|leave|list <name or ->".
func (v *Game) commandParty(args []string) error {
	op, name := args[0], args[1]

	switch op {
	case "list":
		v.Infof("%s", v.gameClient.RosterPanelSummary())

		return nil
	case "accept", "decline", "leave":
		return v.gameClient.SendParty(op, "")
	case "invite":
		id, err := v.resolveTarget(name)
		if err != nil {
			return err
		}

		return v.gameClient.SendParty(op, id)
	}

	return fmt.Errorf("party: unknown operation %q (invite, accept, decline, leave, list)", op)
}

// commandHostile is "hostile <name> <0|1>": declare or withdraw hostility.
func (v *Game) commandHostile(args []string) error {
	id, err := v.resolveTarget(args[0])
	if err != nil {
		return err
	}

	op := d2netpacket.PartyPeace
	if args[1] == "1" {
		op = d2netpacket.PartyHostile
	}

	return v.gameClient.SendParty(op, id)
}

// commandRoster logs the roster and the contents of the party panel.
func (v *Game) commandRoster(_ []string) error {
	v.Infof("%s", v.gameClient.RosterPanelSummary())

	if v.gameControls != nil && v.gameControls.PartyPanel != nil {
		v.Infof("ROSTER PANEL rows open=%v [%s]", v.gameControls.PartyPanel.IsOpen(),
			strings.Join(v.gameControls.PartyPanel.RosterRows(), " | "))
	}

	return nil
}

// commandGiveItem puts a new item into the hero's inventory (console aid).
func (v *Game) commandGiveItem(args []string) error {
	name, err := v.gameControls.GiveItem(args[0])
	if err != nil {
		return err
	}

	v.Infof("GIVEITEM code=%s name=%q", args[0], name)
	v.questItemPickedUp(args[0]) // the quest system counts a quest item the hero now carries

	return nil
}

// commandKillNear kills the nearest monster with the hero as the killer (console
// aid for the party experience scenario; the kill goes through the normal path).
func (v *Game) commandKillNear(_ []string) error {
	d := v.monsterDirector()
	if d == nil {
		return errors.New("no monsters yet")
	}

	m := v.nearestMonster()
	if m == nil {
		return errors.New("no monster near")
	}

	v.Infof("KILLNEAR %s", m.Label())
	d.Damage(m, 1<<20, v.localPlayer)

	return nil
}

// ---- party experience ----

// partyXP is the monster director's hook: with party members around, the
// experience of the hero's kill goes to the server, which splits it.
func (v *Game) partyXP(src *d2mapentity.Player, xp, monsterLevel int, monster string) bool {
	if src != v.localPlayer || v.gameClient.IsSinglePlayer() || len(v.gameClient.Roster.PartyMembers(v.me())) < 2 {
		return false
	}

	pkt, err := d2netpacket.CreatePartyXPPacket(d2netpacket.PartyXPPacket{Killer: v.me(), Monster: monster, XP: xp, MonsterLevel: monsterLevel})
	if err != nil || v.gameClient.SendPacketToServer(pkt) != nil {
		return false
	}

	v.Infof("PARTYXP kill monster=%q xp=%d sent to the party", monster, xp)

	return true
}

func (v *Game) onPartyXP(p d2netpacket.PartyXPPacket) {
	if p.Player != v.me() || v.localPlayer == nil {
		return
	}

	before := v.localPlayer.Stats.Experience
	amount := p.Amount

	if p.MonsterLevel > 0 { // the server already scaled by level; the item +% experience is ours
		amount += amount * v.localPlayer.Stats.ItemExperiencePct() / 100
	}

	v.localPlayer.Stats.Experience += amount
	killer := p.Killer

	if m, ok := v.gameClient.Roster.Member(p.Killer); ok {
		killer = m.Name
	}

	v.Infof("PARTYXP award amount=%d of=%d killer=%q monster=%q experience %d->%d", amount, p.XP, killer, p.Monster,
		before, v.localPlayer.Stats.Experience)
}

// ---- player versus player ----

// commandPvP swings at another player.
func (v *Game) commandPvP(args []string) error {
	id, err := v.resolveTarget(args[0])
	if err != nil {
		return err
	}

	target := v.gameClient.Players[id]
	if target == nil {
		return fmt.Errorf("player %s is not here", args[0])
	}

	return v.pvpSwing(target)
}

// pvpSwing is a melee swing at another hero with the hit pipeline of the
// hero's attacks on monsters; the hit is announced to the server, which checks
// the hostility and relays it to the defender.
func (v *Game) pvpSwing(target *d2mapentity.Player) error {
	r := v.gameClient.Roster

	if !r.CanAttack(v.me(), target.ID()) {
		reason := "not hostile"
		if r.SameParty(v.me(), target.ID()) {
			reason = "party members do not hurt each other"
		}

		v.Infof("PVP BLOCKED target=%q reason=%q", target.Name(), reason)

		return nil
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())
	tx, ty := int(target.Position.X()), int(target.Position.Y())

	if d := d2monster.EdgeDistance(hx-tx, hy-ty, 1); d > heroMeleeReach {
		v.Infof("PVP out of reach target=%q distance=%d reach=%d", target.Name(), d, heroMeleeReach)

		return nil
	}

	d := v.monsterDirector()
	if d == nil {
		return errors.New("no combat director yet")
	}

	v.localPlayer.StopMoving()
	v.localPlayer.SetDirection(v.localPlayer.Position.DirectionTo(target.Position.Vector))

	s := d.HeroStrikePlayer(v.localPlayer, target)
	if !s.Hit {
		v.Infof("PVP SWING target=%q hit=false chance=%d roll=%d", target.Name(), s.Chance, s.Roll)

		return nil
	}

	v.Infof("PVP SWING target=%q hit=true raw=%d scaled=%d pct=%d crit=%v", target.Name(), s.Raw, s.Scaled,
		d2combat.PvPPercent(), s.Crit)

	pkt, err := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Attacker: v.me(), Target: target.ID(),
		Damage: s.Scaled, Raw: s.Raw})
	if err != nil {
		return err
	}

	return v.gameClient.SendPacketToServer(pkt)
}

// commandPvPCast is "pvpcast <skill> <player>": the skill goes at the other
// player's position (scenarios: the autoscript cast needs fixed coordinates).
func (v *Game) commandPvPCast(args []string) error {
	id, err := v.resolveTarget(args[1])
	if err != nil {
		return err
	}

	target := v.gameClient.Players[id]
	if target == nil {
		return fmt.Errorf("player %s is not here", args[1])
	}

	eng := v.skillEngine()
	if eng == nil {
		return errors.New("no skill engine yet")
	}

	skill := strings.ReplaceAll(args[0], "_", " ")

	sid := eng.SkillID(skill)
	if sid < 0 {
		return fmt.Errorf("unknown skill %q", skill)
	}

	x, y := target.GetPositionF()
	v.Infof("PVP CAST skill=%q target=%q at=(%.1f,%.1f)", skill, target.Name(), x, y)
	v.OnPlayerCast(sid, x, y)

	return nil
}

// commandPvPWalk is "pvpwalk <dx> <dy>": walk by that many tiles.
func (v *Game) commandPvPWalk(args []string) error {
	dx, err1 := strconv.ParseFloat(args[0], 64)
	dy, err2 := strconv.ParseFloat(args[1], 64)

	if err1 != nil || err2 != nil {
		return errors.New("pvpwalk needs two numbers")
	}

	x, y := v.localPlayer.GetPositionF()
	v.OnPlayerMove(x+dx, y+dy)

	return nil
}

// commandSetHP is "sethp <n>": the hero's life points (scenarios).
func (v *Game) commandSetHP(args []string) error {
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return errors.New("sethp needs a positive number")
	}

	v.localPlayer.Stats.Health = n
	v.Infof("HP set to %d/%d", n, v.localPlayer.Stats.MaxHealth)

	return nil
}

// skillRivals lists the heroes the local hero's skills may hurt: declared
// hostile (and not in its party), not in town, alive and in this level.
func (v *Game) skillRivals() []*d2mapentity.Player {
	if v.localPlayer == nil || v.gameClient.IsSinglePlayer() || v.localPlayer.IsInTown() {
		return nil
	}

	var out []*d2mapentity.Player

	for id, p := range v.gameClient.Players {
		if id == v.me() || p == nil || p.IsInTown() || !v.gameClient.Roster.CanAttack(v.me(), id) {
			continue
		}

		out = append(out, p)
	}

	return out
}

// sendSkillPvP announces a scaled skill hit on a hostile hero to the server,
// which checks the hostility and relays it to the defender.
func (v *Game) sendSkillPvP(h d2skills.PvPHit) {
	pkt, err := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Attacker: v.me(), Target: h.Target.ID(),
		Damage: h.Parts.Total(), Raw: h.Raw, Skill: h.Skill, Parts: h.Parts.Slice()})
	if err != nil {
		v.Errorf("PVP skill packet: %v", err)
		return
	}

	v.Infof("PVP SKILL target=%q skill=%q raw=%d scaled=%d pct=%d", h.Target.Name(), h.Skill, h.Raw, h.Parts.Total(),
		d2combat.PvPPercent())

	if err := v.gameClient.SendPacketToServer(pkt); err != nil {
		v.Errorf("PVP skill packet: %v", err)
	}
}

// onPvPHit applies a hit of a hostile player to the local hero.
func (v *Game) onPvPHit(p d2netpacket.PvPHitPacket) {
	if p.Kill {
		v.onPvPKill(p)
		return
	}

	if p.Target != v.me() || v.localPlayer == nil || v.localPlayer.IsDead() {
		return
	}

	st := v.localPlayer.Stats
	physResist, reduce := 0, 0
	def := d2combat.PvPDefender{}

	if t := st.Totals; t != nil {
		physResist, reduce = t.PhysResist, t.DamageReduction
		def = d2combat.PvPDefender{
			PhysResist: t.PhysResist, MagicResist: t.MagicResist, Reduce: t.DamageReduction, MagicReduce: t.MagicReduction,
			FireResist: t.ResistShown[d2statlist.ResFire], ColdResist: t.ResistShown[d2statlist.ResCold],
			LightResist: t.ResistShown[d2statlist.ResLight],
		}
	}

	taken := d2combat.PvPReceive(p.Damage, physResist, reduce)

	if parts, ok := d2combat.PvPPartsFromSlice(p.Parts); ok {
		taken = d2combat.PvPReceiveParts(parts, def)
	}

	before := st.Health

	st.Health -= taken
	if st.Health < 0 {
		st.Health = 0
	}

	attacker := p.Attacker
	if m, ok := v.gameClient.Roster.Member(p.Attacker); ok {
		attacker = m.Name
	}

	v.Infof("PVP HIT attacker=%q raw=%d scaled=%d taken=%d hp %d->%d/%d skill=%q", attacker, p.Raw, p.Damage, taken, before,
		st.Health, st.MaxHealth, p.Skill)

	if before > 0 && st.Health == 0 {
		// the hero dies like to a monster (corpse, experience loss); the victim
		// also tells the killer, who gets an ear for a hardcore victim
		// (d2combat.PvPKillGivesEar)
		v.Infof("PVP KILLED by=%q ear=%v hardcore=%v", attacker, d2combat.PvPKillGivesEar(v.localPlayer.Hardcore),
			v.localPlayer.Hardcore)

		class, _ := d2hero.D2SClassOf(v.localPlayer.Class)

		pkt, err := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Attacker: v.me(), Target: p.Attacker, Kill: true,
			Level: st.Level, Class: int(class), Hardcore: v.localPlayer.Hardcore})
		if err == nil {
			err = v.gameClient.SendPacketToServer(pkt)
		}

		if err != nil {
			v.Errorf("PVP kill packet: %v", err)
		}
	}
}

// onPvPKill is the killer's side of a PvP death: the victim's client reports
// it; for a hardcore victim the killer receives an ear, which is dropped on the
// ground where the victim fell (the original puts the ear in the killer's
// inventory or on the ground; the drop is UNVERIFIED).
func (v *Game) onPvPKill(p d2netpacket.PvPHitPacket) {
	if p.Target != v.me() || v.localPlayer == nil {
		return
	}

	victim := p.Attacker
	if m, ok := v.gameClient.Roster.Member(p.Attacker); ok {
		victim = m.Name
	}

	if !d2combat.PvPKillGivesEar(p.Hardcore) {
		v.Infof("PVP KILL victim=%q level=%d softcore, no ear", victim, p.Level)
		return
	}

	x, y := v.localPlayer.GetPositionF()
	if pl := v.gameClient.Players[p.Attacker]; pl != nil {
		x, y = pl.GetPositionF()
	}

	ear, err := v.itemFactory().NewEar(victim, p.Class, p.Level)
	if err != nil {
		v.Errorf("PVP EAR: %v", err)
		return
	}

	cells := v.freeDropCells(int(math.Floor(x)), int(math.Floor(y)), 1, false)
	if len(cells) == 0 {
		v.Warningf("PVP EAR: no free ground cell")
		return
	}

	if _, err := v.spawnGroundItem(ear, cells[0]); err != nil {
		v.Errorf("PVP EAR: %v", err)
		return
	}

	v.Infof("PVP EAR dropped victim=%q level=%d class=%d label=%q pos=(%d,%d)", victim, p.Level, p.Class, plainLabel(ear.Label()),
		cells[0].X, cells[0].Y)
}

// ---- trade ----

// commandTrade is "trade request|yes|no|add|remove|gold|accept|cancel <arg or ->".
func (v *Game) commandTrade(args []string) error {
	op, arg := args[0], args[1]

	switch op {
	case "request":
		id, err := v.resolveTarget(arg)
		if err != nil {
			return err
		}

		v.syncForTrade()

		return v.sendTrade(d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: id})
	case "yes", "no":
		v.syncForTrade()

		return v.sendTrade(d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: op == "yes"})
	case "add", "remove":
		return v.tradeItemCommand(op, arg)
	case "gold":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return errors.New("trade gold needs a number")
		}

		if !v.gameControls.PTrade.IsOpen() {
			return errors.New("no trade window is open")
		}

		v.syncForTrade()
		v.gameControls.PTrade.SetGold(n)

		return nil
	case "accept":
		v.TradeAccept()

		return nil
	case "cancel":
		v.TradeCancel()

		return nil
	}

	return fmt.Errorf("trade: unknown operation %q", op)
}

func (v *Game) tradeItemCommand(op, code string) error {
	w := v.gameControls.PTrade
	if !w.IsOpen() {
		return errors.New("no trade window is open")
	}

	v.syncForTrade()

	if op == "add" {
		s, ok := v.gameControls.FindInventoryItem(code, w.Offer().Items)
		if !ok {
			return fmt.Errorf("no (other) %q in the inventory", code)
		}

		w.Toggle(s)

		return nil
	}

	for _, s := range w.Offer().Items {
		if strings.EqualFold(s.Code, code) {
			w.Toggle(s)

			return nil
		}
	}

	return fmt.Errorf("%q is not in the offer", code)
}

// syncForTrade saves the hero so the server sees the current inventory and
// gold (it moves items between the saved states of the two heroes).
func (v *Game) syncForTrade() {
	if err := v.OnPlayerSave(); err != nil {
		v.Errorf("TRADE: saving the hero: %v", err)
	}
}

func (v *Game) sendTrade(p d2netpacket.TradeCommandPacket) error {
	pkt, err := d2netpacket.CreateTradeCommandPacket(p)
	if err != nil {
		return err
	}

	return v.gameClient.SendPacketToServer(pkt)
}

// TradeOffer, TradeAccept and TradeCancel implement d2player.PlayerTradeHandler.

// TradeOffer sends the hero's current offer.
func (v *Game) TradeOffer(offer d2playertrade.Offer) {
	v.syncForTrade()

	if err := v.sendTrade(d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer, Offer: offer}); err != nil {
		v.Errorf("TRADE offer: %v", err)
	}
}

// TradeAccept accepts the offers as they are.
func (v *Game) TradeAccept() {
	if err := v.sendTrade(d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeAccept}); err != nil {
		v.Errorf("TRADE accept: %v", err)
	}
}

// TradeCancel ends the trade.
func (v *Game) TradeCancel() {
	if err := v.sendTrade(d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeCancel}); err != nil {
		v.Errorf("TRADE cancel: %v", err)
	}
}

// onTrade shows the server's view of the trade.
func (v *Game) onTrade(u d2netpacket.TradeUpdatePacket) {
	gc := v.gameControls
	if gc == nil {
		return
	}

	switch u.State {
	case d2playertrade.Requested.String():
		if u.Requester {
			v.Infof("TRADE requested of %q", u.PartnerName)
		} else {
			v.Infof("TRADE request from %q (trade yes - / trade no -)", u.PartnerName)
		}
	case d2playertrade.Open.String():
		if !gc.PTrade.IsOpen() {
			v.Infof("TRADE open with %q", u.PartnerName)
			gc.OpenPlayerTrade(u.PartnerName)
		}

		if u.Reason != "" {
			v.Infof("TRADE notice: %s", u.Reason)
		}

		gc.PTrade.Update(u.Yours, u.Theirs, u.YouAccepted, u.TheyAccepted, u.Reason)
		v.Infof("TRADE window with=%q yours=%v gold=%d theirs=%v gold=%d you_accepted=%v they_accepted=%v", u.PartnerName,
			gc.TradeItemNames(u.Yours.Items), u.Yours.Gold, gc.TradeItemNames(u.Theirs.Items), u.Theirs.Gold,
			u.YouAccepted, u.TheyAccepted)
	case d2playertrade.Cancelled.String():
		v.Infof("TRADE cancelled with=%q reason=%q", u.PartnerName, u.Reason)
		gc.ClosePlayerTrade()
	case d2playertrade.Done.String():
		if u.Moved == nil {
			return
		}

		gave, got := gc.ApplyTrade(u.Moved)
		v.Infof("TRADE done with=%q gave=%v got=%v gold before=%d after=%d", u.PartnerName, gave, got,
			u.Moved.GoldBefore, u.Moved.GoldAfter)
		gc.ClosePlayerTrade()
	}
}

// d2playerPartyHooks connects the party panel's rows to the roster.
func d2playerPartyHooks(v *Game) d2player.PartyHooks {
	return d2player.PartyHooks{
		Relation: v.relationOf,
		Hostile:  func(p *d2mapentity.Player) bool { return v.gameClient.Roster.Hostile(v.me(), p.ID()) },
		Op: func(p *d2mapentity.Player) string {
			r := v.gameClient.Roster

			switch {
			case r.InvitedBy(v.me()) == p.ID():
				return "accept"
			case r.SameParty(v.me(), p.ID()):
				return "leave"
			case r.InvitedBy(p.ID()) == v.me():
				return "" // invited, waiting
			}

			return "invite"
		},
		OnButton: func(p *d2mapentity.Player, op string) {
			target := p.ID()
			if op == "leave" || op == "accept" {
				target = ""
			}

			if err := v.gameClient.SendParty(op, target); err != nil {
				v.Errorf("party %s: %v", op, err)
			}
		},
		OnHostile: func(p *d2mapentity.Player, hostile bool) {
			op := d2netpacket.PartyPeace
			if hostile {
				op = d2netpacket.PartyHostile
			}

			if err := v.gameClient.SendParty(op, p.ID()); err != nil {
				v.Errorf("party %s: %v", op, err)
			}
		},
	}
}

// onRoster logs the roster events through the game screen's log (the
// autoscript's waitlog steps read that log).
func (v *Game) onRoster(notice string) {
	r := v.gameClient.Roster
	inv := ""

	if m, ok := r.Member(r.InvitedBy(v.me())); ok {
		inv = m.Name
	}

	v.Infof("SOCIAL roster n=%d party=%d invited_by=%q notice=%q", r.Len(), r.PartyID(v.me()), inv, notice)
}

// commandAutoBuy is "autobuy <vendor>": the scripted buy of OD2_AUTOTRADE for one
// vendor, from an autoscript (see scripts/verify.d/9e-d2s-item-export.sh).
func (v *Game) commandAutoBuy(args []string) error {
	if len(args) != 1 || v.gameControls == nil || v.gameClient == nil {
		return errors.New("usage: autobuy <vendor> (in a game)")
	}

	for _, e := range v.gameClient.MapEngine.Entities() {
		if e.Label() != args[0] {
			continue
		}

		if !v.openTrade(e, 1) {
			return fmt.Errorf("%s has no trade window", args[0])
		}

		v.gameControls.Trade.RunAutoTest()

		return nil
	}

	return fmt.Errorf("no %s here", args[0])
}

// commandDropInv is "dropinv <code>": the hero drops the first inventory item with that base code.
func (v *Game) commandDropInv(args []string) error {
	if len(args) != 1 || v.gameControls == nil {
		return errors.New("usage: dropinv <code> (in a game)")
	}

	name, err := v.gameControls.DropInventoryItem(args[0])
	if err != nil {
		return err
	}

	v.Infof("DROPINV code=%s name=%q", args[0], name)

	return nil
}
