package d2client

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// The realm connection (package d2realmclient) feeds the engine the packets
// it already knows (players, moves, casts, chat) plus RealmUnit packets for
// the monsters of the authoritative simulation. These are the client's
// accessors for it.

type realmLink interface {
	RealmUnitPos(unit uint32) (x, y float64, ok bool)
	RealmAttack(unit uint32) error
	RealmSummary() string
}

func (g *GameClient) realm() realmLink {
	r, _ := g.clientConnection.(realmLink)

	return r
}

// IsRealm reports whether the game is played through the realm.
func (g *GameClient) IsRealm() bool { return g.realm() != nil }

// StartPosition is where the engine puts its heroes on the current map.
func (g *GameClient) StartPosition() (x, y float64) { return g.MapEngine.GetStartPosition() }

// RealmUnitPos returns where the simulation has a unit now, in tiles.
func (g *GameClient) RealmUnitPos(unit uint32) (x, y float64, ok bool) {
	if r := g.realm(); r != nil {
		return r.RealmUnitPos(unit)
	}

	return 0, 0, false
}

// RealmAttack tells the realm the local hero attacks a monster.
func (g *GameClient) RealmAttack(unit uint32) error {
	if r := g.realm(); r != nil {
		return r.RealmAttack(unit)
	}

	return nil
}

// RealmSummary describes the simulation as this client sees it (one log line).
func (g *GameClient) RealmSummary() string {
	if r := g.realm(); r != nil {
		return r.RealmSummary()
	}

	return "not a realm game"
}

func (g *GameClient) handleRealmUnitPacket(packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalRealmUnit(packet.PacketData)
	if err != nil {
		return err
	}

	if g.OnRealmUnit != nil {
		g.OnRealmUnit(p)
	}

	return nil
}
