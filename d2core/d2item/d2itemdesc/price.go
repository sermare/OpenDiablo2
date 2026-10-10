package d2itemdesc

import "strconv"

// PriceKind selects one of the vendor tooltip price lines.
type PriceKind int

// Price lines of the vendor window.
const (
	PriceBuy    PriceKind = iota // "Cost: "
	PriceSell                    // "Sell value: "
	PriceRepair                  // "Repair cost: "
)

// string table keys and English fallbacks of the labels (the keys are the
// real ones: "Cost", "Sell", "Repair").
var priceLabels = [...]struct{ key, fallback string }{
	{"Cost", "Cost: "},
	{"Sell", "Sell value: "},
	{"Repair", "Repair cost: "},
}

// PriceLine builds a price line of a vendor tooltip. Prices are gold
// colored; a purchase the hero cannot afford is red (hero gold < 0 means the
// hero's gold is unknown and the line stays gold).
func (t *Tables) PriceLine(kind PriceKind, price, heroGold int) Line {
	l := priceLabels[kind]

	label := t.Tr(l.key)
	if label == "" {
		label = l.fallback
	}

	c := Gold
	if kind == PriceBuy && heroGold >= 0 && price > heroGold {
		c = Red
	}

	return Line{label + strconv.Itoa(price), c}
}
