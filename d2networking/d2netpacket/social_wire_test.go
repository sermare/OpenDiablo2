package d2netpacket

import (
	"encoding/json"
	"strings"
	"testing"
)

// The trade sequence numbers are optional on the wire: a client or server that
// does not number offers produces and reads the old JSON.
func TestTradeSeqWireCompat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		seq  uint32
	}{
		{"old command", `{"op":"offer","offer":{"gold":3}}`, 0},
		{"numbered command", `{"op":"offer","offer":{"gold":3},"seq":7}`, 7},
	}

	for _, tt := range tests {
		var c TradeCommandPacket
		if err := json.Unmarshal([]byte(tt.in), &c); err != nil || c.Seq != tt.seq || c.Offer.Gold != 3 {
			t.Errorf("%s: %+v %v", tt.name, c, err)
		}
	}

	var u TradeUpdatePacket
	if err := json.Unmarshal([]byte(`{"state":"open","yours":{},"theirs":{}}`), &u); err != nil || u.YourSeq != 0 {
		t.Errorf("old update: %+v %v", u, err)
	}

	b, _ := json.Marshal(TradeCommandPacket{Op: TradeOffer})
	b2, _ := json.Marshal(TradeUpdatePacket{State: "open"})

	if strings.Contains(string(b), "seq") || strings.Contains(string(b2), "yourSeq") {
		t.Errorf("zero seq must be omitted: %s %s", b, b2)
	}
}
