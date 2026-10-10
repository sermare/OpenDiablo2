package d2server

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// A refused offer edit still consumes its sequence number: the refusal update
// carries the offer the server really holds and must be taken by a client whose
// local offer is ahead, otherwise the window keeps showing an offer the server
// never applied (and an accept would commit something else).
func TestServerTradeRefusedOfferEchoesSeqAndConverges(t *testing.T) {
	g, a, b := testServer()

	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: "Bob"})
	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: true})

	var pend d2playertrade.Pending

	pend.Toggle(item("rin", 0, 0))
	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer, Offer: pend.Offer, Seq: pend.Seq})

	// the second edit puts the same slot in twice: the server refuses it
	bad := d2playertrade.Offer{Items: []d2hero.StoredItem{item("rin", 0, 0), item("rin", 0, 0)}}
	pend.Offer, pend.Seq = bad, pend.Seq+1
	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer, Offer: bad, Seq: pend.Seq})

	u := lastTrade(t, a)
	if u.Reason == "" || len(u.Yours.Items) != 1 || u.YourSeq != pend.Seq {
		t.Fatalf("refusal update: %+v (want reason, 1 item, seq %d)", u, pend.Seq)
	}

	if !pend.Apply(u.YourSeq, u.Yours) || len(pend.Offer.Items) != 1 {
		t.Fatalf("client did not converge on the server's offer: %+v", pend.Offer)
	}
}
