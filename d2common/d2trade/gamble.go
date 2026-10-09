package d2trade

// Gamble describes the base item row used by TRADE_CalcGamblePrice.
type Gamble struct {
	ReqLevel   int
	Cost       int
	MinStack   int
	MaxStack   int
	GambleCost int // gamble cost column, returned as is for rings and amulets
	// IsRingOrAmulet: item code "rin " or "amu ".
	IsRingOrAmulet bool

	// Exceptional and elite upgrade rows; Has* false when the code is
	// "0   " or empty.
	HasExc, HasElite   bool
	ExcReq, EliteReq   int
	ExcCost, EliteCost int
}

// GamblePrice implements TRADE_CalcGamblePrice for player level l.
func GamblePrice(l int, g Gamble) int {
	if g.IsRingOrAmulet {
		return g.GambleCost
	}

	avg := (g.MinStack + g.MaxStack) / 2
	if avg < 1 {
		avg = 1
	}

	base := g.Cost * avg

	pExc, pElite := 0, 0
	if g.HasExc {
		pExc = maxInt(0, (l-g.ExcReq)*100/2+1)
	}

	if g.HasElite {
		pElite = maxInt(0, (l-g.EliteReq)*100/4+1)
	}

	lp := maxInt(l, 5)
	t := maxInt(0, g.ReqLevel-45) - g.ReqLevel/2 + lp

	inner := (t*250)/3 + ((10000-pExc-pElite)*base+g.EliteCost*pElite+g.ExcCost*pExc)/10000

	return inner * ((2*lp+1)/3 + 20) / 15
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
