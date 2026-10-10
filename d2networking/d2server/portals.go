package d2server

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// The server keeps the town portal pairs of the game (package d2portal): a
// client reports the pair its hero opened, the server stores it as that hero's
// only pair and tells everybody the new list, so every client can draw both
// ends in whichever level its hero stands.

// onPortalOpen stores or closes the pair of the sender.
func (g *GameServer) onPortalOpen(client ClientConnection, packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalPortalOpen(packet.PacketData)
	if err != nil {
		return err
	}

	me := client.GetUniqueID()

	g.soc.mu.Lock()
	name := g.nameOf(me)
	g.soc.mu.Unlock()

	if p.Close {
		if _, ok := g.soc.portals.Close(me); ok {
			g.Infof("PORTAL closed owner=%q", name)
			g.broadcastPortals(fmt.Sprintf("%s's town portal closed", name))
		}

		return nil
	}

	pair := p.Pair
	pair.Owner, pair.OwnerName = me, name // the sender is the owner, whatever it says

	pair.ID = 0

	if old, ok := g.soc.portals.Get(me); ok && old.Field == pair.Field {
		pair.ID = old.ID // the town end moved (cast again in town): the same pair
	}

	stored, replaced, had := g.soc.portals.Open(pair)

	g.Infof("PORTAL opened owner=%q id=%d field=%d town=%d replaced=%v", name, stored.ID, stored.Field.Level, stored.Town.Level,
		had && replaced.ID != stored.ID)
	g.broadcastPortals(fmt.Sprintf("%s opened a portal to town", name))

	return nil
}

// broadcastPortals sends the open pairs to every client.
func (g *GameServer) broadcastPortals(notice string) {
	pkt, err := d2netpacket.CreatePortalUpdatePacket(d2netpacket.PortalUpdatePacket{Pairs: g.soc.portals.Pairs(), Notice: notice})
	if err != nil {
		g.Errorf("PortalUpdate: %v", err)
		return
	}

	conns := make([]ClientConnection, 0, len(g.connections))

	for _, c := range g.connections {
		conns = append(conns, c)
	}

	for _, c := range conns {
		if err := c.SendPacketToClient(pkt); err != nil {
			g.Errorf("PortalUpdate to %s: %v", c.GetUniqueID(), err)
		}
	}
}
