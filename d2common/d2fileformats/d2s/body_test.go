package d2s

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// buildBody appends valid quest, waypoint, NPC, stats and skill sections to a
// header-only save and fixes up its size and checksum.
func buildBody(t testing.TB, stats map[int]uint64, skills [numSkills]byte) []byte {
	t.Helper()

	data := buildSave("Body", Sorceress, 5, StatusExpansion)

	quests := make([]byte, questsSize)
	copy(quests, questsTag[:])
	binary.LittleEndian.PutUint32(quests[4:], 6)
	binary.LittleEndian.PutUint16(quests[8:], questsSize)
	quests[questsHeaderLen] = 0xAA                    // first normal quest byte
	quests[questsHeaderLen+questsPerDiff] = 0xBB      // first nightmare quest byte
	quests[questsHeaderLen+2*questsPerDiff+95] = 0xCC // last hell quest byte
	data = append(data, quests...)

	wp := make([]byte, waypointsSize)
	copy(wp, waypointsTag[:])
	binary.LittleEndian.PutUint16(wp[6:], waypointsSize)
	wp[waypointsHeaderLen+2] = 0x01 // normal: first waypoint
	wp[waypointsHeaderLen+waypointRecordLen+2+4] = 0x7F
	data = append(data, wp...)

	npc := make([]byte, npcSize)
	copy(npc, npcTag[:])
	npc[2] = 0x55
	data = append(data, npc...)

	// stats: ids ascending, then the 0x1FF terminator, padded to a byte
	data = append(data, statsTag[:]...)

	var bits []bool
	put := func(v uint64, n int) {
		for i := 0; i < n; i++ {
			bits = append(bits, v>>i&1 == 1)
		}
	}

	for id := 0; id < 16; id++ {
		v, ok := stats[id]
		if !ok {
			continue
		}

		info, _ := DefaultStatStorage(id)
		put(uint64(id), statIDBits)
		put(v, info.Bits)
	}

	put(statEndID, statIDBits)

	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}

	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i+j] {
				b |= 1 << j
			}
		}

		data = append(data, b)
	}

	data = append(data, skillsTag[:]...)
	data = append(data, skills[:]...)
	data = append(data, 'J', 'M', 0, 0)

	binary.LittleEndian.PutUint32(data[8:], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[checksumOffset:], 0)
	binary.LittleEndian.PutUint32(data[checksumOffset:], Checksum(data))

	return data
}

func TestParseBodySynthetic(t *testing.T) {
	var skills [numSkills]byte

	skills[0], skills[29] = 1, 20

	data := buildBody(t, map[int]uint64{
		StatStrength: 30, StatLevel: 5, StatMaxHP: 150 << 8, StatGold: 12345, StatExperience: 4_000_000_000,
	}, skills)

	if _, err := ParseHeader(data); err != nil {
		t.Fatal(err)
	}

	body, err := ParseBody(data, nil)
	if err != nil {
		t.Fatal(err)
	}

	a := body.Attributes
	if a.Strength != 30 || a.Level != 5 || a.MaxHP != 150 || a.Gold != 12345 || a.Experience != 4_000_000_000 {
		t.Fatalf("attributes: %+v", a)
	}

	if body.Quests[0][0] != 0xAA || body.Quests[1][0] != 0xBB || body.Quests[2][95] != 0xCC {
		t.Fatal("quest bytes in the wrong place")
	}

	if body.Waypoints[0] != 1 || body.Waypoints[1] != 0x7F<<32 || body.Waypoints[2] != 0 {
		t.Fatalf("waypoints: %#x", body.Waypoints)
	}

	if body.NPC[0] != 0x55 || body.SkillPoints[0] != 1 || body.SkillPoints[29] != 20 {
		t.Fatal("npc/skills wrong")
	}

	if !bytes.Equal(data[body.ItemsOffset:body.ItemsOffset+2], []byte("JM")) {
		t.Fatalf("items offset 0x%X does not point at JM", body.ItemsOffset)
	}
}

func TestParseBodyErrors(t *testing.T) {
	data := buildBody(t, map[int]uint64{StatLevel: 1}, [numSkills]byte{})

	if _, err := ParseBody(data[:questsOffset+10], nil); !errors.Is(err, ErrTruncated) {
		t.Fatalf("truncated: %v", err)
	}

	bad := append([]byte(nil), data...)
	bad[questsOffset] = 'X'

	if _, err := ParseBody(bad, nil); !errors.Is(err, ErrBadSection) {
		t.Fatalf("bad tag: %v", err)
	}

	unknown := func(int) (StatStorage, bool) { return StatStorage{}, false }
	if _, err := ParseBody(data, unknown); !errors.Is(err, ErrUnknownCharStat) {
		t.Fatalf("unknown stat: %v", err)
	}
}

// TestRealBody parses the body of a real save and compares it with the
// expected output of the reference parser when both env vars are set.
func TestRealBody(t *testing.T) {
	path, expectedPath := os.Getenv("D2S_SAMPLE_BODY"), os.Getenv("D2S_SAMPLE_BODY_JSON")
	if path == "" || expectedPath == "" {
		t.Skip("set D2S_SAMPLE_BODY and D2S_SAMPLE_BODY_JSON to run")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = ParseHeader(data); err != nil {
		t.Fatal(err)
	}

	body, err := ParseBody(data, nil)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}

	var want struct {
		Attributes map[string]uint64 `json:"attributes"`
		Skills     []struct {
			ID     int `json:"id"`
			Points int `json:"points"`
		} `json:"skills"`
	}

	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}

	a := body.Attributes
	got := map[string]uint64{
		"strength": a.Strength, "energy": a.Energy, "dexterity": a.Dexterity, "vitality": a.Vitality,
		"unused_stats": a.UnusedStats, "unused_skill_points": a.UnusedSkillPoints,
		"current_hp": a.CurrentHP, "max_hp": a.MaxHP, "current_mana": a.CurrentMana, "max_mana": a.MaxMana,
		"current_stamina": a.CurrentStamina, "max_stamina": a.MaxStamina,
		"level": a.Level, "experience": a.Experience, "gold": a.Gold, "stashed_gold": a.StashedGold,
	}

	for k, v := range want.Attributes {
		if got[k] != v {
			t.Errorf("attribute %s = %d, want %d", k, got[k], v)
		}
	}

	if len(want.Skills) != numSkills {
		t.Fatalf("expected %d skills, got %d", numSkills, len(want.Skills))
	}

	for i, s := range want.Skills {
		if int(body.SkillPoints[i]) != s.Points {
			t.Errorf("skill %d (id %d) points = %d, want %d", i, s.ID, body.SkillPoints[i], s.Points)
		}
	}

	if !bytes.Equal(data[body.ItemsOffset:body.ItemsOffset+2], []byte("JM")) {
		t.Fatalf("items offset 0x%X does not point at JM", body.ItemsOffset)
	}

	t.Logf("level %d, str %d, items start at 0x%X", a.Level, a.Strength, body.ItemsOffset)
}
