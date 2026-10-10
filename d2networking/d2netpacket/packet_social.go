package d2netpacket

import (
	"encoding/json"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2party"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2portal"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// The social packets (party, trade, player versus player) have no counterpart
// in the game protocol that has been verified, so over OD2_PROTO=d2gs they ride
// the 0xAE / 0x6c tunnel like the hero state (package d2gsnet). UNVERIFIED: the
// real ids (party 0x5e..0x5f, 0x8c..0x8f, trade 0x38 and the PlrTrade.cpp
// handlers) were not decoded.

// Party commands (PartyCommandPacket.Op).
const (
	PartyInvite  = "invite"  // Target: player id; the target gets an invitation
	PartyAccept  = "accept"  // accept the pending invitation
	PartyDecline = "decline" // refuse it
	PartyLeave   = "leave"   // leave the party
	PartyHostile = "hostile" // Target: player id; declare hostility
	PartyPeace   = "peace"   // Target: player id; withdraw it
)

// PartyCommandPacket is a request of a client about parties and hostility.
type PartyCommandPacket struct {
	Op     string `json:"op"`
	Target string `json:"target,omitempty"`
}

// RosterUpdatePacket tells the clients the whole roster and what just happened.
type RosterUpdatePacket struct {
	Roster d2party.Snapshot `json:"roster"`
	Notice string           `json:"notice,omitempty"`
	// Player is who the notice is for ("" for everybody).
	Player string `json:"player,omitempty"`
}

// Trade commands (TradeCommandPacket.Op).
const (
	TradeRequest = "request" // Target: player id
	TradeRespond = "respond" // Accept: answer to the request
	TradeOffer   = "offer"   // Offer: replaces the sender's offer
	TradeAccept  = "accept"
	TradeCancel  = "cancel"
)

// TradeCommandPacket is a request of a client about its trade.
type TradeCommandPacket struct {
	Op     string              `json:"op"`
	Target string              `json:"target,omitempty"`
	Accept bool                `json:"accept,omitempty"`
	Offer  d2playertrade.Offer `json:"offer"`
}

// TradeUpdatePacket is the trade as one of the two players sees it.
type TradeUpdatePacket struct {
	State        string              `json:"state"` // requested, open, done, cancelled
	Partner      string              `json:"partner"`
	PartnerName  string              `json:"partnerName"`
	Requester    bool                `json:"requester,omitempty"` // this player asked for the trade
	Yours        d2playertrade.Offer `json:"yours"`
	Theirs       d2playertrade.Offer `json:"theirs"`
	YouAccepted  bool                `json:"youAccepted,omitempty"`
	TheyAccepted bool                `json:"theyAccepted,omitempty"`
	Reason       string              `json:"reason,omitempty"`
	// Moved is set when the trade is done: what this player gave and got.
	Moved *d2playertrade.Moved `json:"moved,omitempty"`
}

// PvPHitPacket is a hit of one hostile player on another, resolved by the
// attacker's client (like its hits on monsters); the defender's client applies
// the damage to its hero.
type PvPHitPacket struct {
	Attacker string `json:"attacker"`
	Target   string `json:"target"`
	Damage   int    `json:"damage"` // after the player-versus-player scale
	Raw      int    `json:"raw"`    // before it
	// Skill and Parts describe a skill hit: the skill name and the scaled
	// damage by type (phys, fire, lightning, magic, cold); the defender applies
	// its own resists to each. Empty for a melee swing (Damage is physical).
	Skill string `json:"skill,omitempty"`
	Parts []int  `json:"parts,omitempty"`
	// Kill reverses the direction: the defender's client tells the killer
	// (Target) that the sender (Attacker, the victim) died of its hit. Level
	// and Class describe the victim; Hardcore says whether it was a hardcore
	// hero (the killer then receives an ear). UNVERIFIED: the original has no
	// such packet in this form.
	Kill     bool `json:"kill,omitempty"`
	Level    int  `json:"level,omitempty"`
	Class    int  `json:"class,omitempty"`
	Hardcore bool `json:"hardcore,omitempty"`
}

// PartyXPPacket carries the experience of a kill. The killer's client sends it
// with XP set (the whole amount); the server splits it by party and sends each
// member (Player) its Amount.
type PartyXPPacket struct {
	Killer  string `json:"killer"`
	Monster string `json:"monster,omitempty"`
	XP      int    `json:"xp"`
	// MonsterLevel lets the server scale each share by the member level; 0 (an
	// old client) keeps the plain level-proportional split.
	MonsterLevel int    `json:"monsterLevel,omitempty"`
	Player       string `json:"player,omitempty"`
	Amount       int    `json:"amount,omitempty"`
}

// PortalOpenPacket is a client telling the server it opened its town portal
// pair (Pair) or closed it (Close). The server takes the sender as the owner.
type PortalOpenPacket struct {
	Pair  d2portal.Pair `json:"pair"`
	Close bool          `json:"close,omitempty"`
}

// PortalUpdatePacket is the server's list of open pairs, sent to everybody
// when it changes (and to a player who joins).
type PortalUpdatePacket struct {
	Pairs  []d2portal.Pair `json:"pairs"`
	Notice string          `json:"notice,omitempty"`
}

// CreatePortalOpenPacket builds a PortalOpen packet.
func CreatePortalOpenPacket(p PortalOpenPacket) (NetPacket, error) {
	return social(d2netpackettype.PortalOpen, p)
}

// CreatePortalUpdatePacket builds a PortalUpdate packet.
func CreatePortalUpdatePacket(p PortalUpdatePacket) (NetPacket, error) {
	return social(d2netpackettype.PortalUpdate, p)
}

// UnmarshalPortalOpen decodes a PortalOpen packet's data.
func UnmarshalPortalOpen(b []byte) (p PortalOpenPacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

// UnmarshalPortalUpdate decodes a PortalUpdate packet's data.
func UnmarshalPortalUpdate(b []byte) (p PortalUpdatePacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

func social(t d2netpackettype.NetPacketType, v interface{}) (NetPacket, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return NetPacket{PacketType: t}, err
	}

	return NetPacket{PacketType: t, PacketData: b}, nil
}

// CreatePartyCommandPacket builds a PartyCommand packet.
func CreatePartyCommandPacket(op, target string) (NetPacket, error) {
	return social(d2netpackettype.PartyCommand, PartyCommandPacket{Op: op, Target: target})
}

// CreateRosterUpdatePacket builds a RosterUpdate packet.
func CreateRosterUpdatePacket(p RosterUpdatePacket) (NetPacket, error) {
	return social(d2netpackettype.RosterUpdate, p)
}

// CreateTradeCommandPacket builds a TradeCommand packet.
func CreateTradeCommandPacket(p TradeCommandPacket) (NetPacket, error) {
	return social(d2netpackettype.TradeCommand, p)
}

// CreateTradeUpdatePacket builds a TradeUpdate packet.
func CreateTradeUpdatePacket(p TradeUpdatePacket) (NetPacket, error) {
	return social(d2netpackettype.TradeUpdate, p)
}

// CreatePvPHitPacket builds a PvPHit packet.
func CreatePvPHitPacket(p PvPHitPacket) (NetPacket, error) {
	return social(d2netpackettype.PvPHit, p)
}

// CreatePartyXPPacket builds a PartyXP packet.
func CreatePartyXPPacket(p PartyXPPacket) (NetPacket, error) {
	return social(d2netpackettype.PartyXP, p)
}

// UnmarshalPartyCommand decodes a PartyCommand packet's data.
func UnmarshalPartyCommand(b []byte) (p PartyCommandPacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

// UnmarshalRosterUpdate decodes a RosterUpdate packet's data.
func UnmarshalRosterUpdate(b []byte) (p RosterUpdatePacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

// UnmarshalTradeCommand decodes a TradeCommand packet's data.
func UnmarshalTradeCommand(b []byte) (p TradeCommandPacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

// UnmarshalTradeUpdate decodes a TradeUpdate packet's data.
func UnmarshalTradeUpdate(b []byte) (p TradeUpdatePacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

// UnmarshalPvPHit decodes a PvPHit packet's data.
func UnmarshalPvPHit(b []byte) (p PvPHitPacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}

// UnmarshalPartyXP decodes a PartyXP packet's data.
func UnmarshalPartyXP(b []byte) (p PartyXPPacket, err error) {
	err = json.Unmarshal(b, &p)

	return
}
