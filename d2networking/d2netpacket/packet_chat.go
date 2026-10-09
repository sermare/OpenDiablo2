package d2netpacket

import (
	"encoding/json"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// ChatPacket is a chat line. A client sends it with only Text set; the server
// fills PlayerID and Name of the sender and relays it to every client.
type ChatPacket struct {
	PlayerID string `json:"playerId"`
	Name     string `json:"name"`
	Text     string `json:"text"`
}

// CreateChatPacket returns a NetPacket which declares a ChatPacket.
func CreateChatPacket(playerID, name, text string) (NetPacket, error) {
	b, err := json.Marshal(ChatPacket{PlayerID: playerID, Name: name, Text: text})
	if err != nil {
		return NetPacket{PacketType: d2netpackettype.Chat}, err
	}

	return NetPacket{PacketType: d2netpackettype.Chat, PacketData: b}, nil
}

// UnmarshalChat unmarshals the given data to a ChatPacket struct.
func UnmarshalChat(packet []byte) (ChatPacket, error) {
	var p ChatPacket
	if err := json.Unmarshal(packet, &p); err != nil {
		return p, err
	}

	return p, nil
}
