package d2quest

import (
	_ "embed" // the message table is embedded
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Mode is the kind of an NPC message in a speech table (quests-2.md
// section 2). The server sends mode 1 as mode 0.
type Mode int

// The message modes of the speech tables.
const (
	// ModeSpoken lines are voiced when the player clicks Talk, once, and are
	// acknowledged to the server (the "message heard" event).
	ModeSpoken Mode = 0
	// ModeGiver is the quest-giver variant of a spoken line; the server turns
	// it into ModeSpoken before sending (QUESTS_InitScrollTextChain).
	ModeGiver Mode = 1
	// ModeTopic lines are listed as clickable topics of the Talk submenu.
	ModeTopic Mode = 2
	// ModeBark is an overhead speech bubble (rogue and guard barks); never
	// present in the quest tables themselves.
	ModeBark Mode = 3
)

// Speech is one entry of a speech table: an NPC class says message Msg.
type Speech struct {
	NPC  int
	Msg  int
	Mode Mode
	// Quest is the id of the quest whose table produced the entry (filled in
	// when a dialog is built).
	Quest int
}

// MsgSound is the Sounds.txt row of a message id (message id -> row lookup of
// SOUND_LookupQuestMessageSound, 0x4dd560; the table was dumped from the
// 1.14b binary, see data/messages.csv).
type MsgSound struct {
	Index  int    // Sounds.txt "Index" column
	Handle string // Sounds.txt handle
	File   string // Sounds.txt file name
}

//go:embed data/messages.csv
var messagesCSV string

// npcByName maps the NPC names used in data/messages.csv to monster classes.
// ACT2GUARD2/4/5 are the guard classes 331/377/378 listed in the menu table
// (which of them is which is unverified).
//
//nolint:gochecknoglobals // static lookup data
var npcByName = map[string]int{
	"CAIN1": NPCCain1, "GHEED": NPCGheed, "AKARA": NPCAkara, "KASHYA": NPCKashya, "CHARSI": NPCCharsi,
	"WARRIV1": NPCWarriv1, "WARRIV2": NPCWarriv2, "ATMA": NPCAtma, "DROGNAN": NPCDrognan, "FARA": NPCFara,
	"GREIZ": NPCGreiz, "ELZIX": NPCElzix, "GEGLASH": NPCGeglash, "JERHYN": NPCJerhyn, "LYSANDER": NPCLysander,
	"MESHIF1": NPCMeshif1, "CAIN2": NPCCain2, "CAIN3": NPCCain3, "CAIN4": NPCCain4, "TYRAEL1": NPCTyrael1,
	"ASHEARA": NPCAsheara, "HRATLI": NPCHratli, "ALKOR": NPCAlkor, "ORMUS": NPCOrmus, "IZUALGHOST": 406,
	"MESHIF2": NPCMeshif2, "CAIN5": NPCCain5, "NAVI": NPCNavi, "NATALYA": NPCNatalya, "TYRAEL2": NPCTyrael2,
	"MALACHAI": NPCMalachai, "LARZUK": NPCLarzuk, "DREHYA": NPCDrehya, "MALAH": NPCMalah,
	"NIHLATHAK": NPCNihlathak, "QUAL_KEHK": NPCQualKehk, "CAIN6": NPCCain6, "TYRAEL3": NPCTyrael3,
	"DREHYAICED": NPCAnyaFrozen, "ANCIENTSTATUE1": NPCAncientStatue1, "ANCIENTSTATUE2": NPCAncientStatue2,
	"ANCIENTSTATUE3": NPCAncientStatue3,
	"ACT2GUARD2":     331, "ACT2GUARD4": 377, "ACT2GUARD5": 378,
}

type messageData struct {
	tables map[string][][]Speech // by quest prefix, e.g. "A1Q1", then table index
	sounds map[int]MsgSound
}

//nolint:gochecknoglobals // lazily parsed embedded table
var (
	messageOnce sync.Once
	messages    messageData
)

// introMessages are the intro and gossip lines that are not part of a
// quest's speech table (quests-2.md section 4.3). Indexes come from the
// notes and are checked against Sounds.txt by a test when D2_TABLES is set.
//
//nolint:gochecknoglobals // static lookup data
var introMessages = map[int]MsgSound{
	11:  {3499, "ESOUND_AKARA_ACT1_INTRO", `act1\akara\aka_act1_intro.wav`},
	12:  {3500, "ESOUND_AKARA_ACT1_INTRO_SOR", `act1\akara\aka_act1_intro_sor.wav`},
	24:  {4095, "ESOUND_KASHYA_ACT1_INTRO", `act1\kashya\kas_act1_intro.wav`},
	25:  {4096, "ESOUND_KASHYA_ACT1_INTRO_AMA", `act1\kashya\kas_act1_intro_ama.wav`},
	36:  {3753, "ESOUND_CHARSI_ACT1_INTRO", `act1\charsi\cha_act1_intro.wav`},
	37:  {3754, "ESOUND_CHARSI_ACT1_INTRO_BAR", `act1\charsi\cha_act1_intro_bar.wav`},
	45:  {3927, "ESOUND_GHEED_ACT1_INTRO", `act1\gheed\ghe_act1_intro.wav`},
	46:  {3928, "ESOUND_GHEED_ACT1_INTRO_NEC", `act1\gheed\ghe_act1_intro_nec.wav`},
	253: {4064, "ESOUND_JERHYN_ACT2_INTRO", `act2\jerhyn\jer_act2_intro.wav`},
	302: {4309, "ESOUND_TYRAEL_ACT2_INTRO", `act2\tyreal\tyr_act2_intro.wav`},
}

func loadMessages() {
	messages = messageData{tables: map[string][][]Speech{}, sounds: map[int]MsgSound{}}

	rows, err := csv.NewReader(strings.NewReader(messagesCSV)).ReadAll()
	if err != nil {
		panic(fmt.Sprintf("d2quest: bad embedded messages.csv: %v", err))
	}

	for _, r := range rows[1:] {
		prefix, _, _ := strings.Cut(r[0], " ")

		idx, _ := strconv.Atoi(r[1])
		msg, _ := strconv.Atoi(r[3])
		mode, _ := strconv.Atoi(r[4])
		snd, _ := strconv.Atoi(r[5])

		npc, ok := npcByName[r[2]]
		if !ok {
			npc = -1
		}

		t := messages.tables[prefix]
		for len(t) <= idx {
			t = append(t, nil)
		}

		t[idx] = append(t[idx], Speech{NPC: npc, Msg: msg, Mode: Mode(mode)})
		messages.tables[prefix] = t

		if snd != 0 {
			messages.sounds[msg] = MsgSound{Index: snd, Handle: r[6], File: strings.ReplaceAll(r[7], `\`, "/")}
		}
	}

	for m, s := range introMessages {
		s.File = strings.ReplaceAll(s.File, `\`, "/")
		messages.sounds[m] = s
	}
}

// SoundForMessage returns the Sounds.txt row spoken for a message id.
func SoundForMessage(msg int) (MsgSound, bool) {
	messageOnce.Do(loadMessages)
	s, ok := messages.sounds[msg]

	return s, ok
}

// MessageCount is the number of message ids with a sound row.
func MessageCount() int {
	messageOnce.Do(loadMessages)

	return len(messages.sounds)
}

// speechTables returns the speech tables of a quest by its prefix in
// data/messages.csv ("A1Q1"). The result is shared; do not modify it.
func speechTables(prefix string) [][]Speech {
	messageOnce.Do(loadMessages)

	return messages.tables[prefix]
}

// normalise converts the mode the way the server does before sending.
func (s Speech) normalise() Speech {
	if s.Mode == ModeGiver {
		s.Mode = ModeSpoken
	}

	return s
}

// textKeys are the string.tbl keys of the intro and prologue lines.
//
//nolint:gochecknoglobals // static lookup data
var textKeys = map[int]string{
	0: "WarrivAct1IntroGossip1", 1: "WarrivAct1IntroPalGossip1",
	11: "AkaraIntroGossip1", 12: "AkaraIntroSorGossip1",
	24: "KashyaIntroGossip1", 25: "KashyaIntroAmaGossip1",
	36: "CharsiIntroGossip1", 37: "CharsiIntroBarGossip1",
	45: "GheedIntroGossip1", 46: "GheedIntroNecGossip1",
	// unverified order of the five signpost lines
	59: "RogueSignpostGossip1", 60: "RogueSignpostGossip2", 61: "RogueSignpostGossip3",
	62: "RogueSignpostGossip4", 63: "RogueSignpostGossip5",
	253: "JerhynActIntroGossip1",
	// keys found by playing the quests (TestTextKeysExist): the handles of these lines do not follow the
	// NPC_ACT_Q_KIND rule
	302: "TyraelActIntroGossip1", 465: "HratliActIntroGossip1", 466: "HratliActIntroSorGossip1",
	664: "TyraelAct4Gossip1", 20002: "AncientsAct5IntroGossip1",
	336: "A2Q2EarlyReturnCapCain", 337: "A2Q2EarlyReturnStaveCain", 338: "A2Q2EarlyReturnCubeCain",
	544: "A3Q2EarlyReturnHeartCain", 545: "A3Q2EarlyReturnEyeCain", 546: "A3Q2EarlyReturnBrainCain",
	547: "A3Q2EarlyReturnFlailCain", 548: "A3Q2SuccessfulCain",
	678: "A4Q3InitHasStoneCain", 679: "A4Q3InitNoStoneCain",
	20000: "A4Q2ExpansionSuccessTyrael", 20001: "A4Q2ExpansionSuccessCain",
	20127: "A5Q3FoundAnyaMalah", 20128: "A5Q3FoundAnyaCain", 20129: "A5Q3FoundAnyaLarzuk",
	20130: "A5Q3FoundAnyaQualKehk", 20131: "A5Q3FoundAnyaAnya",
}

// TextKey returns the string.tbl key of a message's text ("" when unknown).
// The keys follow the Sounds.txt handle names (AKARA_ACT1_Q1_INIT ->
// A1Q1InitAkara); the rule was checked by hand against the keys of the 1.14b
// string.tbl for Act 1 and the Radament quest. Other acts are unverified.
func TextKey(msg int) string {
	if k, ok := textKeys[msg]; ok {
		return k
	}

	s, ok := SoundForMessage(msg)
	if !ok {
		return ""
	}

	return textKeyFromHandle(s.Handle)
}

// textKeyKinds maps the handle suffix to the key infix.
//
//nolint:gochecknoglobals // static lookup data
var textKeyKinds = map[string]string{
	"INIT": "Init", "AFTER": "AfterInit", "EARLY": "EarlyReturn", "SUCCESS": "Successful",
	"AFTER_SCROLL": "AfterInitScroll", "EARLY_SCROLL": "EarlyReturnS", "SUCCESS_SCROLL": "SuccessfulScroll",
	"INSTRUCTIONS": "Instructions", "RESCUED_HERO": "RescuedByHero", "RESCUED_ROGUES": "RescuedByRogues",
}

func textKeyFromHandle(handle string) string {
	h := strings.TrimPrefix(handle, "ESOUND_")
	parts := strings.SplitN(h, "_", 4) // NPC ACT1 Q1 KIND
	// NPC names can contain an underscore (QUAL_KEHK); none of the Act 1/2
	// quest lines do.
	if len(parts) != 4 || !strings.HasPrefix(parts[1], "ACT") || !strings.HasPrefix(parts[2], "Q") {
		return ""
	}

	npc, act, q, kind := parts[0], parts[1][3:], parts[2][1:], parts[3]
	name := strings.ToUpper(npc[:1]) + strings.ToLower(npc[1:])

	switch name {
	case "Warriv1":
		name = "Warriv"
	case "Warriv2":
		name = "WarrivAct2"
	case "Warriv":
		if act == "2" {
			name = "WarrivAct2"
		}
	case "Meshif1", "Meshif2":
		name = "Meshif"
	case "Cain1", "Cain2", "Cain3", "Cain4", "Cain5", "Cain6":
		name = "Cain"
	case "Qualkehk":
		name = "QualKehk" // the Act 5 keys spell it so (A5Q2InitQualKehk)
	}

	// Act 3 keys of the two NPCs that also exist in other acts carry an "Act3" suffix (A3Q1AfterInitCainAct3; the Khalim's Will
	// keys, A3Q2, do not)
	if act == "3" && q != "2" && (name == "Cain" || name == "Meshif") {
		name += "Act3"
	}

	infix, ok := textKeyKinds[kind]
	if !ok {
		return ""
	}

	key := "A" + act + "Q" + q + infix + name

	// exceptions found in the string table
	switch key {
	case "A1Q1AfterInitCharsi":
		key = "A1Q1AfterInitCharsiMain"
	case "A1Q4SuccessfulAkara", "A1Q4SuccessfulKashya", "A1Q4SuccessfulGheed", "A1Q4SuccessfulCharsi",
		"A1Q4SuccessfulWarriv", "A1Q4SuccessfulCain":
		key = "A1Q4QuestSuccessful" + name
	case "A1Q4AfterInitGheed":
		key = "A1Q4AfterInitScrollGheed"
	case "A1Q6EarlyReturnKashya":
		key = "A1Q6EarlyReturn2Kashya"
	case "A2Q2EarlyReturnSCain":
		key = "A2Q2EarlyReturnScrollCain"
	case "A2Q2SuccessfulCain":
		key = "A2Q2SuccessfulStaffCain"
	case "A2Q4SuccessfulGreiz":
		key = "A2Q4SuccessfulGriez" // (sic) the table spells it so
	}

	return key
}
