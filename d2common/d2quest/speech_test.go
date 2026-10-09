package d2quest

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSpeechTableShapes(t *testing.T) {
	// per-state entry counts of the Den of Evil table (quests.md section 4 and D2MOO)
	want := []int{1, 5, 5, 5, 3}
	tables := speechTables("A1Q1")

	if len(tables) != len(want) {
		t.Fatalf("A1Q1 has %d tables", len(tables))
	}

	for i, n := range want {
		if len(tables[i]) != n {
			t.Errorf("A1Q1 table %d has %d entries, want %d", i, len(tables[i]), n)
		}
	}

	// the quest giver line is mode 0 in table 0 and 3, everything else is a topic
	if s := tables[0][0]; s.NPC != NPCAkara || s.Msg != 64 || s.Mode != ModeSpoken {
		t.Errorf("table 0: %+v", s)
	}

	for _, s := range tables[3] {
		if s.Msg == 76 && s.Mode != ModeSpoken || s.Msg != 76 && s.Mode != ModeTopic {
			t.Errorf("table 3 entry %+v", s)
		}
	}

	// quest-giver lines of the other Act 1 quests
	givers := map[string][2]int{"A1Q2": {NPCKashya, 81}, "A1Q3": {NPCCharsi, 146}, "A1Q4": {NPCAkara, 97}, "A1Q6": {NPCCain5, 166}}
	for q, g := range givers {
		s := speechTables(q)[0][0]
		if s.NPC != g[0] || s.Msg != g[1] || s.normalise().Mode != ModeSpoken {
			t.Errorf("%s first line %+v", q, s)
		}
	}
}

func TestEveryAct1MessageHasASound(t *testing.T) {
	for _, q := range []string{"A1Q0", "A1Q1", "A1Q2", "A1Q3", "A1Q4", "A1Q5", "A1Q6", "A1Q7", "A2Q0", "A2Q1"} {
		for i, tbl := range speechTables(q) {
			for _, s := range tbl {
				if s.NPC < 0 {
					t.Errorf("%s table %d: unknown NPC in %+v", q, i, s)
				}

				if snd, ok := SoundForMessage(s.Msg); !ok || snd.Index == 0 {
					t.Errorf("%s table %d msg %d has no Sounds.txt row", q, i, s.Msg)
				}
			}
		}
	}

	for _, m := range []int{11, 12, 24, 25, 36, 37, 45, 46, 59, 63} {
		if _, ok := SoundForMessage(m); !ok {
			t.Errorf("intro message %d has no sound", m)
		}
	}

	if MessageCount() < 500 {
		t.Errorf("only %d message ids resolved", MessageCount())
	}
}

func TestSoundRows(t *testing.T) {
	tests := []struct {
		msg   int
		index int
		file  string
	}{
		{64, 3503, "act1/akara/aka_act1_q1_init.wav"},
		{76, 3504, "act1/akara/aka_act1_q1_successful.wav"},
		{92, 4103, "act1/kashya/kas_act1_q2_successful.wav"},
		{163, 3764, "act1/charsi/cha_act1_q3_successful.wav"},
		{118, 3516, "act1/akara/aka_act1_q4_successful.wav"},
		{183, 4357, "act1/warriv/war_act1_q6_successful.wav"},
		{11, 3499, "act1/akara/aka_act1_intro.wav"},
		{0, 4336, "act1/warriv/war_act1_intro.wav"},
	}

	for _, tc := range tests {
		s, ok := SoundForMessage(tc.msg)
		if !ok || s.Index != tc.index || s.File != tc.file {
			t.Errorf("msg %d -> %+v", tc.msg, s)
		}
	}
}

func TestTextKeys(t *testing.T) {
	tests := map[int]string{
		64: "A1Q1InitAkara", 65: "A1Q1AfterInitAkara", 67: "A1Q1AfterInitCharsiMain", 71: "A1Q1EarlyReturnAkara",
		76: "A1Q1SuccessfulAkara", 81: "A1Q2InitKashya", 118: "A1Q4QuestSuccessfulAkara", 99: "A1Q4AfterInitScrollAkara",
		104: "A1Q4EarlyReturnSAkara", 112: "A1Q4InstructionsAkara", 124: "A1Q4RescuedByHeroCain",
		163: "A1Q3SuccessfulCharsi", 184: "A1Q6SuccessfulCain", 101: "A1Q4AfterInitScrollWarriv",
		305: "A2Q1AfterInitGreiz", 11: "AkaraIntroGossip1",
	}

	for msg, want := range tests {
		got := TextKey(msg)
		// 305 is Greiz or another Act 2 NPC depending on the table; only check the prefix
		if msg == 305 {
			if !strings.HasPrefix(got, "A2Q1AfterInit") {
				t.Errorf("msg %d key %q", msg, got)
			}

			continue
		}

		if got != want {
			t.Errorf("msg %d key %q, want %q", msg, got, want)
		}
	}
}

// With D2_TABLES set the sound rows are compared with the extracted
// Sounds.txt of the expansion (the table the notes were dumped against).
func TestSoundRowsAgainstSoundsTxt(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "sound", "d2exp", "Sounds.txt"))
	if err != nil {
		t.Skip(err)
	}

	rows := map[int][2]string{}

	for _, line := range strings.Split(string(data), "\n")[1:] {
		f := strings.Split(line, "\t")
		if len(f) < 3 {
			continue
		}

		if n, err := strconv.Atoi(f[1]); err == nil {
			rows[n] = [2]string{f[0], strings.ReplaceAll(f[2], `\`, "/")}
		}
	}

	messageOnce.Do(loadMessages)

	bad := 0

	for msg, s := range messages.sounds {
		r, ok := rows[s.Index]
		if !ok || r[0] != s.Handle || strings.TrimSpace(r[1]) != s.File {
			bad++

			if bad < 5 {
				t.Errorf("msg %d: table says %+v, Sounds.txt row is %v", msg, s, r)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d rows differ", bad)
	}
}

// With D2_STRING_TBL set (data\local\lng\eng\string.tbl) every text key the
// Act 1 messages produce must exist in the table.
func TestTextKeysExistInStringTbl(t *testing.T) {
	path := os.Getenv("D2_STRING_TBL")
	if path == "" {
		t.Skip("D2_STRING_TBL not set")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	missing := 0

	for _, q := range []string{"A1Q0", "A1Q1", "A1Q2", "A1Q3", "A1Q4", "A1Q5", "A1Q6", "A2Q1"} {
		for _, tbl := range speechTables(q) {
			for _, s := range tbl {
				key := TextKey(s.Msg)
				if key == "" || !bytes.Contains(data, append([]byte(key), 0)) {
					missing++

					t.Errorf("%s msg %d: text key %q not in string.tbl", q, s.Msg, key)
				}
			}
		}
	}

	if missing > 0 {
		t.Logf("%d keys missing", missing)
	}
}
