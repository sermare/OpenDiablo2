package d2quest

import (
	"bytes"
	"os"
	"sort"
	"testing"
)

// TestTextKeysExist checks, against a real string.tbl (D2_STRINGTBL=<extracted data/local/lng/eng/string.tbl>),
// that the key TextKey derives from the sound handle of a quest message is a key of the table. A message whose
// key does not exist is spoken (the sound plays) but shows no subtitle. Without the file the test skips.
func TestTextKeysExist(t *testing.T) {
	path := os.Getenv("D2_STRINGTBL")
	if path == "" {
		t.Skip("D2_STRINGTBL not set")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	keys := map[string]bool{}
	for _, f := range bytes.Split(raw, []byte{0}) {
		keys[string(f)] = true
	}

	messageOnce.Do(loadMessages)

	var missing []int

	// The lines of The Blade of the Old Religion (A3Q4) that the key rule cannot name: the table numbers its
	// Init lines (A3Q4Init1Asheara, Init2.., Init3..) and has no EarlyReturn/Successful keys for them; they
	// are spoken without a subtitle until the numbering is decoded.
	voiceOnly := map[int]bool{527: true, 528: true, 529: true, 530: true, 531: true, 532: true, 533: true, 539: true, 541: true,
		// no key in the table (or none found): the palace guards, the Izual and angel barks, the "_VA" variants of
		// the Act 3 lines, the reward lines of Alkor and Ormus and a few Act 5 lines
		185: true, 186: true, 187: true, 188: true, 189: true, 538: true, 593: true, 596: true, 600: true, 602: true,
		604: true, 606: true, 608: true, 610: true, 613: true, 615: true, 617: true, 630: true, 634: true, 638: true,
		640: true, 644: true, 646: true, 650: true, 652: true, 668: true, 669: true, 675: true, 20104: true, 20149: true,
		20174: true}

	for msg := range messages.sounds {
		if voiceOnly[msg] {
			continue
		}

		if k := TextKey(msg); k == "" || !keys[k] {
			missing = append(missing, msg)
		}
	}

	sort.Ints(missing)

	for _, m := range missing {
		t.Errorf("message %d (%s): key %q is not in string.tbl", m, messages.sounds[m].Handle, TextKey(m))
	}
}
