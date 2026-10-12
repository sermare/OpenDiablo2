package d2player

import "testing"

// UI_DrawCharacterStatsPanel 0x4a43f0 draws the first line of a two-line caption at the table y - 4 and the second at
// y + 4; the table y of the resistances is 346, 370, 395, 419 (0x723a5c) and the port's frame adds 53.
func TestResistCaptionLinesAroundTableY(t *testing.T) {
	const frame = 53

	for i, tc := range []struct{ tableY, y1, y2 int }{
		{346, labelResFireLine1Y, labelResFireLine2Y},
		{370, labelResColdLine1Y, labelResColdLine2Y},
		{395, labelResLightLine1Y, labelResLightLine2Y},
		{419, labelResPoisLine1Y, labelResPoisLine2Y},
	} {
		if tc.y1 != tc.tableY+frame-4 || tc.y2 != tc.tableY+frame+4 {
			t.Errorf("row %d: lines at %d and %d, want %d and %d", i, tc.y1, tc.y2, tc.tableY+frame-4, tc.tableY+frame+4)
		}
	}
}
