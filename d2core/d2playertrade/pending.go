package d2playertrade

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"

// Pending is the client's side of the offer protocol. The window edits the
// local offer (Toggle, SetGold) and every edit takes the next sequence number,
// which travels with the offer to the server; the server echoes the number of
// the last offer it applied in every update. An update whose echo is older than
// the latest local edit was made before the server saw that edit (the other
// side's change crossed our edit on the wire), so its copy of our offer is
// stale and must not replace the local one. Without this, 'add item' followed
// by 'set gold' lost the item when an update crossed the first edit.
type Pending struct {
	Offer Offer  // what this player offers now (local, possibly ahead of the server)
	Seq   uint32 // sequence number of the latest local edit; 0 = none yet
}

// Reset starts a new trade: nothing offered, no edits.
func (p *Pending) Reset() { *p = Pending{} }

// Toggle puts the item into the offer, or takes it out if it is in already
// (same position and code). It returns the new sequence number.
func (p *Pending) Toggle(s d2hero.StoredItem) uint32 {
	for i, it := range p.Offer.Items {
		if it.X == s.X && it.Y == s.Y && it.Code == s.Code {
			p.Offer.Items = append(append([]d2hero.StoredItem{}, p.Offer.Items[:i]...), p.Offer.Items[i+1:]...)
			return p.bump()
		}
	}

	p.Offer.Items = append(append([]d2hero.StoredItem{}, p.Offer.Items...), s)

	return p.bump()
}

// SetGold sets the gold of the offer and returns the new sequence number.
func (p *Pending) SetGold(n int) uint32 {
	p.Offer.Gold = n
	return p.bump()
}

func (p *Pending) bump() uint32 {
	p.Seq++
	return p.Seq
}

// Apply handles the server's copy of our offer, echoing ack as the sequence
// number of the last edit the server had applied when it built the update. It
// reports whether the copy was taken; a stale copy (ack < Seq) leaves the local
// offer untouched.
func (p *Pending) Apply(ack uint32, yours Offer) bool {
	if ack < p.Seq {
		return false
	}

	p.Offer = yours

	return true
}
