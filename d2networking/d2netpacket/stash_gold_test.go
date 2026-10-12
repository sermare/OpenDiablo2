package d2netpacket

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestAddPlayerStashGoldRoundTrip(t *testing.T) {
	g := 1234

	for _, tc := range []struct {
		name string
		in   *int
		want *int
	}{{"set", &g, &g}, {"unknown", nil, nil}} {
		pkt, err := CreateAddPlayerPacket("id", "n", 0, 0, d2enum.HeroBarbarian, nil, nil, d2inventory.CharacterEquipment{}, 0, 0, 0, nil,
			d2enum.DifficultyNormal, WithStashGold(tc.in))
		if err != nil {
			t.Fatal(err)
		}

		out, err := UnmarshalAddPlayer(pkt.PacketData)
		if err != nil {
			t.Fatal(err)
		}

		switch {
		case tc.want == nil && out.StashGold != nil, tc.want != nil && (out.StashGold == nil || *out.StashGold != *tc.want):
			t.Errorf("%s: got %v want %v", tc.name, out.StashGold, tc.want)
		}
	}
}
