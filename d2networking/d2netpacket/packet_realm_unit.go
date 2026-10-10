package d2netpacket

import (
	"encoding/json"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// Operations of a RealmUnitPacket.
const (
	RealmUnitSpawn = "spawn" // a monster exists (at X,Y tiles)
	RealmUnitMove  = "move"  // a monster is now heading to X,Y tiles
	RealmUnitHit   = "hit"   // a monster was hit: HP of MaxHP left
	RealmUnitDeath = "death" // a monster died (Killer is the killing hero's player id, "" if unknown)
)

// RealmUnitPacket describes a monster of the realm's authoritative
// simulation to the game screen, which draws it as a mirror monster.
type RealmUnitPacket struct {
	Op     string  `json:"op"`
	UnitID uint32  `json:"unit"`
	Type   uint16  `json:"type,omitempty"` // simulation monster type (spawn)
	Name   string  `json:"name,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	HP     int32   `json:"hp,omitempty"`
	MaxHP  int32   `json:"maxHp,omitempty"`
	Killer string  `json:"killer,omitempty"`
}

// CreateRealmUnitPacket returns a NetPacket which declares a RealmUnitPacket.
func CreateRealmUnitPacket(p RealmUnitPacket) (NetPacket, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return NetPacket{PacketType: d2netpackettype.RealmUnit}, err
	}

	return NetPacket{PacketType: d2netpackettype.RealmUnit, PacketData: b}, nil
}

// UnmarshalRealmUnit unmarshals the given data to a RealmUnitPacket struct.
func UnmarshalRealmUnit(packet []byte) (RealmUnitPacket, error) {
	var p RealmUnitPacket
	if err := json.Unmarshal(packet, &p); err != nil {
		return p, err
	}

	return p, nil
}
