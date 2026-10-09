package d2trade

import "testing"

func TestItemPrice(t *testing.T) {
	npc := NPC{PlayerPays: 1024, VendorPays: 512, Repair: 1024}

	tests := []struct {
		name string
		item Item
		p    Params
		want int
	}{
		{"nil-safe starter item", Item{Starter: true, BaseCost: 500}, Params{Mode: ModeBuy, NPC: npc}, 1},
		{"buy plain", Item{BaseCost: 100, Identified: true}, Params{Mode: ModeBuy, NPC: npc}, 100},
		{"sell plain uses vendor-pays mult", Item{BaseCost: 100, Identified: true}, Params{Mode: ModeSell, NPC: npc}, 50},
		{"sell ethereal is a quarter",
			Item{BaseCost: 1000, Identified: true, Ethereal: true},
			Params{Mode: ModeSell, NPC: NPC{PlayerPays: 1024, VendorPays: 1024}}, 250},
		{"sell ethereal and class specific",
			Item{BaseCost: 1000, Identified: true, Ethereal: true, ClassSpecificType: true},
			Params{Mode: ModeSell, NPC: NPC{PlayerPays: 1024, VendorPays: 1024}}, 62},
		{"sell class specific only",
			Item{BaseCost: 1000, ClassSpecificType: true},
			Params{Mode: ModeSell, NPC: NPC{VendorPays: 1024}}, 250},
		{"sell capped by max buy",
			Item{BaseCost: 100000},
			Params{Mode: ModeSell, NPC: NPC{VendorPays: 1024, MaxBuy: [3]int{500, 600, 700}}, Difficulty: 1}, 600},
		{"buy with reduced prices", Item{BaseCost: 1000},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}, ReducedPrices: 10}, 900},
		{"reduced prices clamp at 99", Item{BaseCost: 1000},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}, ReducedPrices: 150}, 10},
		{"arrows 100 at cost 256 are 25 gold", Item{BaseCost: 256, Quantity: 100, MaxStack: 350, IsAmmo: true, Stackable: true},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 25},
		{"low quality halves", Item{BaseCost: 101, Identified: true, Quality: QualityLow},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 51},
		{"affix term", Item{BaseCost: 100, Identified: true, Terms: []Term{{Mult: 512, Add: 7}}},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 157},
		{"affix ignored when unidentified", Item{BaseCost: 100, Terms: []Term{{Mult: 512, Add: 7}}},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 100},
		{"socketed added", Item{BaseCost: 100, Identified: true, SocketedHalfCost: 30},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 130},
		{"armour scales with defense", Item{BaseCost: 1000, IsArmour: true, Defense: 50, MaxAC: 100},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 500},
		{"quest group", Item{BaseCost: 1000},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024, Quest: [3]Quest{{Active: true, Buy: 512}}}}, 500},
		{"quest group inactive", Item{BaseCost: 1000},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024, Quest: [3]Quest{{Buy: 512}}}}, 1000},
		{"quantity multiplies non-ammo buy", Item{BaseCost: 10, Quantity: 3},
			Params{Mode: ModeBuy, NPC: NPC{PlayerPays: 1024}}, 30},
		{"repair half damaged", Item{BaseCost: 1000, Repairable: true, HasDurability: true, MaxDur: 100, CurDur: 50},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 500},
		{"repair with recharge", Item{BaseCost: 1000, Repairable: true, HasDurability: true, MaxDur: 100, CurDur: 50, RechargeCost: 20},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 520},
		{"repair undamaged floors at 1", Item{BaseCost: 1000, Repairable: true, HasDurability: true, MaxDur: 100, CurDur: 100},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 1},
		{"repair not repairable", Item{BaseCost: 1000, HasDurability: true, MaxDur: 100, CurDur: 1},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 0},
		// UNVERIFIED reading of the notes: stat 0xfc measures the missing amount against MaxDur-1
		{"repair replenishing item uses maxdur-1", Item{BaseCost: 1000, Repairable: true, HasDurability: true, MaxDur: 100, CurDur: 50, Replenishes: true},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 490},
		{"repair ethereal skips recharge", Item{BaseCost: 1000, Repairable: true, Ethereal: true, HasDurability: true, MaxDur: 100, CurDur: 50, RechargeCost: 20},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 500},
		{"repair npc mult and reduced prices", Item{BaseCost: 1000, Repairable: true, HasDurability: true, MaxDur: 100, CurDur: 0},
			Params{Mode: ModeRepair, ReducedPrices: 10, NPC: NPC{Repair: 512}}, 450},
		{"repair broken item pays the full price", Item{BaseCost: 1000, Repairable: true, HasDurability: true, MaxDur: 100, CurDur: 0},
			Params{Mode: ModeRepair, NPC: NPC{Repair: 1024}}, 1000},
		{"broken ethereal sells for floor", Item{BaseCost: 1000, Ethereal: true, HasDurability: true, MaxDur: 10, CurDur: 0},
			Params{Mode: ModeSell, NPC: NPC{VendorPays: 1024}}, 1},
		{"gamble ring flat from formula", Item{}, Params{Mode: ModeGamble, Gamble: Gamble{IsRingOrAmulet: true, GambleCost: 1000}, ReducedPrices: 10}, 900},
		{"gamble flat column", Item{}, Params{Mode: ModeGamble, GambleFlat: true, GambleCostColumn: 777}, 777},
		{"gamble formula", Item{}, Params{Mode: ModeGamble, PlayerLevel: 30, Gamble: Gamble{Cost: 100}}, 6933},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			if got := ItemPrice(&tt.item, tt.p); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}

	if ItemPrice(nil, Params{}) != Invalid {
		t.Error("nil item must be Invalid")
	}
}

func TestOverflowGuard(t *testing.T) {
	// above 65536 the multiplication is done on x>>10 first (precision loss is original behaviour)
	if got := mulFixed(100000, 1024); got != (100000>>10)*1024 {
		t.Errorf("got %d", got)
	}

	if got := mulFixed(1000, 512); got != 500 {
		t.Errorf("got %d", got)
	}
}

func TestGamblePrice(t *testing.T) {
	g := Gamble{Cost: 1000, ReqLevel: 60, HasExc: true, ExcReq: 20, ExcCost: 3000, HasElite: true, EliteReq: 50, EliteCost: 9000}
	// L=80: pExc=(60*100/2)+1=3001, pElite=(30*100/4)+1=751
	// t=(60-45)-30+80=65; (65*250)/3=5416
	// (10000-3001-751)*1000=6248000; +9000*751=6759000; +3000*3001=9003000 -> 22010000/10000=2201
	// inner=7617; * ((161)/3+20=73) / 15 = 37069
	if got := GamblePrice(80, g); got != 37069 {
		t.Errorf("got %d", got)
	}

	// a level below the upgrade requirement contributes nothing
	if got := GamblePrice(1, Gamble{Cost: 100, HasExc: true, ExcReq: 20, ExcCost: 5000}); got != GamblePrice(1, Gamble{Cost: 100}) {
		t.Errorf("exc below req changed price: %d", got)
	}
}

func TestIdentifyCost(t *testing.T) {
	tests := []struct {
		n     int
		quest bool
		want  int
	}{{0, false, 0}, {3, false, 300}, {3, true, 0}, {-1, false, 0}}
	for _, tt := range tests {
		if got := IdentifyCost(tt.n, tt.quest); got != tt.want {
			t.Errorf("IdentifyCost(%d,%v)=%d want %d", tt.n, tt.quest, got, tt.want)
		}
	}
}
