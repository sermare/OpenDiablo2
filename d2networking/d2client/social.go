package d2client

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// handleRosterPacket replaces the roster copy and logs what changed for the
// player (invitations, parties, hostility).
func (g *GameClient) handleRosterPacket(packet d2netpacket.NetPacket) error {
	u, err := d2netpacket.UnmarshalRosterUpdate(packet.PacketData)
	if err != nil {
		return err
	}

	before := g.Roster.InvitedBy(g.PlayerID)
	g.Roster.Restore(u.Roster)
	g.Infof("%s", g.Roster.Summary())

	if u.Notice != "" && (u.Player == "" || u.Player == g.PlayerID) {
		g.Infof("PARTY notice: %s", u.Notice)
	}

	if from := g.Roster.InvitedBy(g.PlayerID); from != "" && from != before {
		name := from
		if m, ok := g.Roster.Member(from); ok {
			name = m.Name
		}

		g.Infof("PARTY invitation from %q (party accept / party decline)", name)
	}

	if g.OnRoster != nil {
		notice := u.Notice
		if u.Player != "" && u.Player != g.PlayerID {
			notice = "" // addressed to somebody else
		}

		g.OnRoster(notice)
	}

	return nil
}

func (g *GameClient) handleTradePacket(packet d2netpacket.NetPacket) error {
	u, err := d2netpacket.UnmarshalTradeUpdate(packet.PacketData)
	if err != nil {
		return err
	}

	if g.OnTrade != nil {
		g.OnTrade(u)
	}

	return nil
}

func (g *GameClient) handlePvPPacket(packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalPvPHit(packet.PacketData)
	if err != nil {
		return err
	}

	if g.OnPvPHit != nil {
		g.OnPvPHit(p)
	}

	return nil
}

func (g *GameClient) handlePartyXPPacket(packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalPartyXP(packet.PacketData)
	if err != nil {
		return err
	}

	if g.OnPartyXP != nil {
		g.OnPartyXP(p)
	}

	return nil
}

// ResolvePlayer returns the id of a player given by id or by hero name.
func (g *GameClient) ResolvePlayer(nameOrID string) (string, error) {
	if g.Roster.Has(nameOrID) {
		return nameOrID, nil
	}

	if id, ok := g.Roster.FindByName(nameOrID); ok {
		return id, nil
	}

	for id, p := range g.Players { // before the first roster update arrives
		if p.Name() == nameOrID {
			return id, nil
		}
	}

	return "", fmt.Errorf("no player %q in this game", nameOrID)
}

// SendParty sends a party command.
func (g *GameClient) SendParty(op, target string) error {
	pkt, err := d2netpacket.CreatePartyCommandPacket(op, target)
	if err != nil {
		return err
	}

	return g.SendPacketToServer(pkt)
}

// RosterLines describes the roster for the panel and the logs, one line per
// player: "Name Class L12 party|hostile|neutral|you".
func (g *GameClient) RosterLines() []string {
	snap := g.Roster.Snapshot()
	lines := make([]string, 0, len(snap.Players))

	for _, in := range snap.Players {
		state := "neutral"

		switch {
		case in.ID == g.PlayerID:
			state = "you"
		case g.Roster.Relation(g.PlayerID, in.ID) == d2enum.PlayerRelationFriend:
			state = "party"
		case g.Roster.Relation(g.PlayerID, in.ID) == d2enum.PlayerRelationEnemy:
			state = "hostile"
		case g.Roster.InvitedBy(g.PlayerID) == in.ID:
			state = "invites you"
		case g.Roster.InvitedBy(in.ID) == g.PlayerID:
			state = "invited"
		}

		lines = append(lines, fmt.Sprintf("%s %s L%d %s", in.Name, in.Class, in.Level, state))
	}

	sort.Strings(lines)

	return lines
}

// RosterPanelSummary is the roster panel as a log line.
func (g *GameClient) RosterPanelSummary() string {
	return "ROSTER PANEL " + strings.Join(g.RosterLines(), " | ")
}
