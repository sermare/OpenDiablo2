package d2trade

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The golden testdata/price.json.gz was produced by running TRADE_CalcItemPrice
// (Game.exe 1.14b, 0x62f100) and TRADE_CalcGamblePrice (0x629570) under an x86
// emulator. Item base rows are the real ones; the item unit's flags, quality,
// affix ids and stats are made up, and the lookups behind them (affix rows,
// npc row, quest bits, stats) are served from the case. It holds numbers only.

type gambleRec struct {
	Lv    int     `json:"lv"`
	Cost  int     `json:"cost"`
	Mn    int     `json:"mn"`
	Mx    int     `json:"mx"`
	Lvl   int     `json:"lvl"`
	Gc    int     `json:"gc"`
	Ring  int     `json:"ring"`
	Exc   *[2]int `json:"exc"`
	Elite *[2]int `json:"elite"`
	R     int     `json:"r"`
	Code  uint32  `json:"code"`
}

func (g gambleRec) gamble() Gamble {
	out := Gamble{
		ReqLevel: g.Lvl, Cost: g.Cost, MinStack: g.Mn, MaxStack: g.Mx, GambleCost: g.Gc, IsRingOrAmulet: g.Ring != 0,
	}

	if g.Exc != nil {
		out.HasExc, out.ExcReq, out.ExcCost = true, g.Exc[0], g.Exc[1]
	}

	if g.Elite != nil {
		out.HasElite, out.EliteReq, out.EliteCost = true, g.Elite[0], g.Elite[1]
	}

	return out
}

type priceCase struct {
	M      int               `json:"m"`
	D      int               `json:"d"`
	Q      int               `json:"q"`
	Fl     int               `json:"fl"`
	El     int               `json:"el"`
	Ids    []int             `json:"ids"`
	Auto   int               `json:"auto"`
	Uidx   int               `json:"uidx"`
	Gflat  int               `json:"gflat"`
	Aff    map[string][2]int `json:"aff"`
	Uq     map[string][2]int `json:"uq"`
	St     map[string][2]int `json:"st"`
	Lvl    int               `json:"lvl"`
	Red    int               `json:"red"`
	Qty    int               `json:"qty"`
	Def    int               `json:"defense"`
	Cur    int               `json:"cur"`
	MaxDur int               `json:"maxdur"`
	Rep    int               `json:"rep"`
	Replen int               `json:"replen"`
	Repqty int               `json:"repqty"`
	Rech   int               `json:"recharge"`
	Scroll int               `json:"scroll"`
	Bp     []int             `json:"bp"`
	DStat  [3]int            `json:"dstat"`
	DSkill [3]int            `json:"dskill"`
	DSock  [3]int            `json:"dsock"`
	Qa     [3]int            `json:"qa"`
	Npc    struct {
		Pay int    `json:"pay"`
		Ven int    `json:"ven"`
		Rep int    `json:"rep"`
		Qs  [3]int `json:"qs"`
		Qb  [3]int `json:"qb"`
		Qr  [3]int `json:"qr"`
		Mb  [3]int `json:"mb"`
	} `json:"npc"`
	F struct {
		Cost      int `json:"cost"`
		MinAC     int `json:"minac"`
		MaxAC     int `json:"maxac"`
		Ammo      int `json:"ammo"`
		Stackable int `json:"stackable"`
		MaxStack  int `json:"maxstack"`
		Tome      int `json:"tome"`
		BodyPart  int `json:"bodypart"`
		Armour    int `json:"armour"`
		HasDur    int `json:"hasdur"`
		Throwable int `json:"throwable"`
		ClassSpec int `json:"classspec"`
		MagicPlus int `json:"magicplus"`
	} `json:"f"`
	Gam *gambleRec `json:"gam"`
	R   int        `json:"r"`
}

type priceGolden struct {
	Gamble []gambleRec `json:"gamble"`
	Price  []priceCase `json:"price"`
}

func readPriceGolden(t *testing.T) *priceGolden {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", "price.json.gz"))
	if err != nil {
		t.Skipf("golden missing: %v", err)
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	var g priceGolden
	if err := json.NewDecoder(zr).Decode(&g); err != nil {
		t.Fatal(err)
	}

	return &g
}

func term(m map[string][2]int, id int) *Term {
	if v, ok := m[strconv.Itoa(id)]; ok && id > 0 {
		return &Term{Mult: v[0], Add: v[1]}
	}

	return nil
}

func (c *priceCase) item() *Item {
	it := &Item{
		BaseCost: c.F.Cost, Quantity: c.Qty, MaxStack: c.F.MaxStack, EarLevel: c.El,
		Starter: c.Fl&0x20000 != 0, Identified: c.Fl&0x10 != 0, Ethereal: c.Fl&0x400000 != 0, IsEar: c.Fl&0x10000 != 0,
		Quality: c.Q, IsAmmo: c.F.Ammo != 0, IsTome: c.F.Tome != 0, TomeScrollCost: c.Scroll,
		IsBodyPart: c.F.BodyPart != 0, IsArmour: c.F.Armour != 0, Defense: c.Def, MinAC: c.F.MinAC, MaxAC: c.F.MaxAC,
		Stackable: c.F.Stackable != 0, StatPrice: c.DStat, SkillPrice: c.DSkill, SocketPrice: c.DSock,
		ClassSpecificType: c.F.ClassSpec != 0, HasDurability: c.F.HasDur != 0, MaxDur: c.MaxDur, CurDur: c.Cur,
		Replenishes: c.Replen != 0, Throwable: c.F.Throwable != 0, Repairable: c.Rep != 0, RechargeCost: c.Rech,
		ReplenishQty: c.Repqty != 0,
	}

	if c.Bp != nil {
		w := c.Bp[c.D]
		it.BodyPartWord = &w
	}

	it.Affixes.Auto = term(c.Aff, c.Auto)

	for i := 0; i < 3; i++ {
		it.Affixes.Prefix[i] = term(c.Aff, c.Ids[1+i])
		it.Affixes.Suffix[i] = term(c.Aff, c.Ids[4+i])
	}

	it.Affixes.Unique = term(c.Uq, c.Uidx)
	it.Affixes.Set = term(c.St, c.Uidx)

	return it
}

func (c *priceCase) params() Params {
	p := Params{
		Mode: Mode(c.M), Difficulty: c.D, ReducedPrices: c.Red, PlayerLevel: c.Lvl,
		NPC:        NPC{PlayerPays: c.Npc.Pay, VendorPays: c.Npc.Ven, Repair: c.Npc.Rep, MaxBuy: c.Npc.Mb},
		GambleFlat: c.Gflat < 1,
	}

	for k := range p.NPC.Quest {
		p.NPC.Quest[k] = Quest{Active: c.Qa[k] != 0, Buy: c.Npc.Qs[k], Sell: c.Npc.Qb[k], Repair: c.Npc.Qr[k]}
	}

	if c.Gam != nil {
		p.Gamble = c.Gam.gamble()
		p.GambleCostColumn = c.Gam.Gc
	}

	return p
}

func TestOracleItemPrice(t *testing.T) {
	g := readPriceGolden(t)
	bad, byMode := 0, map[int]int{}

	for i := range g.Price {
		c := &g.Price[i]

		if magicOrBetter(c.Q) != (c.F.MagicPlus != 0) {
			t.Fatalf("magicOrBetter(%d) disagrees with the game", c.Q)
		}

		if got := ItemPrice(c.item(), c.params()); got != c.R {
			bad++
			byMode[c.M]++

			if bad <= 6 {
				t.Errorf("case %d mode %d quality %d flags %#x: got %d, real %d", i, c.M, c.Q, c.Fl, got, c.R)
			}
		}
	}

	t.Logf("%d price cases, %d mismatches (by mode %v)", len(g.Price), bad, byMode)
}

func TestOracleGamblePrice(t *testing.T) {
	g := readPriceGolden(t)
	bad := 0

	for _, c := range g.Gamble {
		if got := GamblePrice(c.Lv, c.gamble()); got != c.R {
			bad++

			if bad <= 6 {
				t.Errorf("level %d row %+v: got %d, real %d", c.Lv, c, got, c.R)
			}
		}
	}

	t.Logf("%d gamble cases, %d mismatches", len(g.Gamble), bad)
}
