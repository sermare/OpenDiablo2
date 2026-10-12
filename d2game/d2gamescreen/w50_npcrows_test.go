package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2npc"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// TestW50LegacyMenuTableAgreesWithTheExe: the fallback table of d2player (used only for classes d2npc does not know)
// must give the same rows as the static table of the exe (0x725d60, read from the file in w50: 48 rows, all equal to
// d2npc), except for the documented deviation of Qual-Kehk.
func TestW50LegacyMenuTableAgreesWithTheExe(t *testing.T) {
	action := map[int]d2player.NPCMenuAction{
		d2npc.StrTalk: d2player.NPCActionTalk, d2npc.StrTrade: d2player.NPCActionTrade,
		d2npc.StrTradeRepair: d2player.NPCActionTradeRepair, d2npc.StrGamble: d2player.NPCActionGamble,
		d2npc.StrHire: d2player.NPCActionHire, d2npc.StrIdentify: d2player.NPCActionIdentify,
		d2npc.StrGoWest: d2player.NPCActionTravelWest, d2npc.StrSailWest: d2player.NPCActionSailWest,
	}

	seen := map[int]bool{}

	for _, class := range d2npc.TableClasses() {
		if seen[class] {
			continue // the first row of a class wins
		}

		seen[class] = true

		exe, _ := d2npc.StaticRows(class)

		legacy, ok := d2player.NPCMenuFor(class)
		if !ok {
			continue // talk-only classes the legacy table does not list
		}

		want := make([]d2player.NPCMenuAction, 0, len(exe))
		for _, o := range exe {
			want = append(want, action[o.StringID])
		}

		if class == d2npc.ClassQualKehk {
			// UNVERIFIED design choice kept on purpose: the exe's row has only Talk (and no other code adds Hire), the game
			// still lets Qual-Kehk hire (npcMenuRows)
			want = []d2player.NPCMenuAction{d2player.NPCActionTalk, d2player.NPCActionHire}
		}

		if len(legacy) != len(want) {
			t.Errorf("class %d: legacy rows %v, exe %v", class, legacy, want)

			continue
		}

		for i := range legacy {
			if legacy[i].Action != want[i] {
				t.Errorf("class %d row %d: legacy %v, exe %v", class, i, legacy[i].Action, want[i])
			}
		}
	}
}

// TestW50Act5Rows pins the Act 5 rows the exe table gives (also read from the file): Qual-Kehk and the other talkers.
func TestW50Act5Rows(t *testing.T) {
	for _, c := range []struct {
		class int
		want  []d2npc.Kind
	}{
		{514, []d2npc.Kind{d2npc.KindTalk, d2npc.KindGamble}},                  // Nihlathak (town): no Trade row
		{515, []d2npc.Kind{d2npc.KindTalk}},                                    // Qual-Kehk
		{520, []d2npc.Kind{d2npc.KindTalk, d2npc.KindIdentify}},                // Cain, Act 5
		{511, []d2npc.Kind{d2npc.KindTalk, d2npc.KindTradeRepair}},             // Larzuk
		{512, []d2npc.Kind{d2npc.KindTalk, d2npc.KindTrade, d2npc.KindGamble}}, // Anya (Drehya)
		{513, []d2npc.Kind{d2npc.KindTalk, d2npc.KindTrade}},                   // Malah
		{521, []d2npc.Kind{d2npc.KindTalk}},                                    // the Act 5 town guard
	} {
		rows, ok := d2npc.StaticRows(c.class)
		if !ok {
			t.Fatalf("class %d has no row", c.class)
		}

		if len(rows) != len(c.want) {
			t.Errorf("class %d: %v", c.class, rows)

			continue
		}

		for i := range rows {
			if rows[i].Kind != c.want[i] {
				t.Errorf("class %d row %d: kind %v, want %v", c.class, i, rows[i].Kind, c.want[i])
			}
		}
	}
}
