package d2netpacket

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

func TestGroundItemPacketRoundTrip(t *testing.T) {
	in := GroundItemPacket{Op: GroundPlaced, Ref: "r", Sender: "a", Entries: []GroundEntry{
		{ID: "g1", Key: "k", Level: 3, X: 12, Y: 7, Item: &d2hero.StoredItem{Code: "ssd", Seed: 99, Quality: 4}, Owner: "Ann"},
		{ID: "g2", Level: 3, Gold: 250},
	}}

	pkt, err := CreateGroundItemPacket(in)
	if err != nil {
		t.Fatal(err)
	}

	out, err := UnmarshalGroundItem(pkt.PacketData)
	if err != nil || !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip: %+v %v", out, err)
	}
}
