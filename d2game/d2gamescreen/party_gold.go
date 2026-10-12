package d2gamescreen

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// partyGold hands a gold pickup to the host when the hero is in a party of two
// or more (PARTY_DistributeGoldPickup 0x53e640, Roster.ShareGold on the server).
// The pile leaves the ground here; each member (the picker included) receives its
// share in onPartyGold. It reports false when the pickup is a plain one.
func (v *Game) partyGold(it *d2mapentity.Item) bool {
	if v.gameClient.IsSinglePlayer() || len(v.gameClient.Roster.PartyMembers(v.me())) < 2 {
		return false
	}

	pkt, err := d2netpacket.CreatePartyGoldPacket(d2netpacket.PartyGoldPacket{Picker: v.me(), Amount: it.Gold,
		PlayerDropped: it.DropOwner != ""})
	if err != nil || v.gameClient.SendPacketToServer(pkt) != nil {
		return false
	}

	v.gameClient.MapEngine.RemoveEntity(it)
	v.Infof("PARTYGOLD pickup amount=%d sent to the party", it.Gold)

	return true
}

// onPartyGold receives this hero's share of a pickup: the purse takes it up to
// the carry limit (the server only knows the last saved purse); what does not
// fit, and the overflow of the whole split (picker only), stays on the ground.
func (v *Game) onPartyGold(p d2netpacket.PartyGoldPacket) {
	if p.Player != v.me() || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	before := v.localPlayer.Gold
	over := v.gameControls.PickUpGold(p.Add) + p.Overflow

	if over > 0 {
		x, y := v.localPlayer.GetPositionF()

		if ent, err := v.gameClient.MapEngine.NewGoldPile(over, v.asset.TranslateString("gld"),
			int(math.Round(x*subtilesInTile)), int(math.Round(y*subtilesInTile))); err == nil {
			v.gameClient.MapEngine.AddEntity(ent)
			v.announceGround(ent, "")
		}
	}

	picker := p.Picker
	if m, ok := v.gameClient.Roster.Member(p.Picker); ok {
		picker = m.Name
	}

	v.Infof("PARTYGOLD award add=%d of=%d picker=%q overflow=%d gold %d->%d", p.Add, p.Amount, picker, over,
		before, v.localPlayer.Gold)
}
