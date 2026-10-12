package d2player

import (
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"
)

// The restock flag is set by the town-leaving check only: for the vendors of
// the act being left, and only when their stamp is more than 240 s old
// (0x534c20, VERIFIED); a later check needs a new interval after the stamp.
func TestLeaveTownFlagsRestock(t *testing.T) {
	start := time.Unix(1000, 0)
	clock := start

	w := &TradeWindow{
		due: map[string]bool{}, stamp: map[string]time.Time{}, keyVendor: map[string]d2vendor.Vendor{},
		now: func() time.Time { return clock },
	}

	a1, a2 := d2vendor.Act1[0], d2vendor.Act2[0]
	w.keyVendor[a1.Name], w.keyVendor[a2.Name] = a1, a2
	w.stamp[a1.Name], w.stamp[a2.Name] = start, start

	clock = start.Add(d2vendor.RestockInterval * time.Millisecond)
	w.LeaveTown(0, true)

	if w.due[a1.Name] {
		t.Fatal("exactly the interval is not enough")
	}

	clock = clock.Add(time.Millisecond)
	w.LeaveTown(0, true)

	if !w.due[a1.Name] || w.due[a2.Name] {
		t.Fatalf("only the act 1 vendor is flagged: %v", w.due)
	}

	if !w.stamp[a1.Name].Equal(clock) || !w.stamp[a2.Name].Equal(start) {
		t.Fatal("the flagged vendor is restamped")
	}

	// leaving again right away does not extend or clear anything
	w.due[a1.Name] = false
	w.LeaveTown(0, true)

	if w.due[a1.Name] {
		t.Fatal("restamped vendor must wait another interval")
	}
}

// Nobody left in the town (always, with one hero): every opened vendor of the act is flagged whatever the age
// of its stock; the other acts and the gamble stocks are not (0x534c20 with the "town empty" argument, VERIFIED;
// the gamble inventories are freed by the regular window's restock, not flagged).
func TestLeaveTownEmptyTownFlagsEveryOpenedVendor(t *testing.T) {
	start := time.Unix(1000, 0)

	w := &TradeWindow{
		due: map[string]bool{}, stamp: map[string]time.Time{}, keyVendor: map[string]d2vendor.Vendor{},
		now: func() time.Time { return start },
	}

	a1, a2 := d2vendor.Act1[0], d2vendor.Act2[0]
	w.keyVendor[a1.Name], w.keyVendor[a2.Name] = a1, a2
	w.keyVendor[a1.Name+gambleKeySuffix] = a1
	w.stamp[a1.Name], w.stamp[a2.Name] = start, start

	w.LeaveTown(0, false)

	if !w.due[a1.Name] || w.due[a2.Name] || w.due[a1.Name+gambleKeySuffix] {
		t.Fatalf("only the act 1 regular stock is flagged: %v", w.due)
	}
}

// Stash gold pays what the inventory gold cannot.
func TestTradePayFallsBackToStash(t *testing.T) {
	hero := &d2mapentity.Player{Gold: 100}
	stash := 50
	w := &TradeWindow{hero: hero, stashGold: &stash}

	if !w.canPay(150) || w.canPay(151) {
		t.Fatal("inventory plus stash is the budget")
	}

	if err := w.pay(120); err != nil || hero.Gold != 0 || stash != 30 {
		t.Fatalf("gold %d stash %d err %v", hero.Gold, stash, err)
	}

	if err := w.pay(31); err != ErrNotEnoughGold || stash != 30 {
		t.Fatalf("a short payment changes nothing: stash %d err %v", stash, err)
	}
}
