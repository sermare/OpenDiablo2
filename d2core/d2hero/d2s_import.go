package d2hero

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// d2sSkillsPerClass is the number of skill slots a .d2s file stores.
const d2sSkillsPerClass = 30

var d2sClassToHero = map[d2s.Class]d2enum.Hero{
	d2s.Amazon:      d2enum.HeroAmazon,
	d2s.Sorceress:   d2enum.HeroSorceress,
	d2s.Necromancer: d2enum.HeroNecromancer,
	d2s.Paladin:     d2enum.HeroPaladin,
	d2s.Barbarian:   d2enum.HeroBarbarian,
	d2s.Druid:       d2enum.HeroDruid,
	d2s.Assassin:    d2enum.HeroAssassin,
}

// ImportD2S converts the contents of a Diablo II .d2s save into a hero state.
// Name, class, level, experience, attributes, health/mana, gold and the
// spent skill points, items, quests, waypoints and NPC flags are imported.
func (f *HeroStateFactory) ImportD2S(data []byte) (*HeroState, error) {
	header, err := d2s.ParseHeader(data)
	if err != nil {
		return nil, err
	}

	hero, ok := d2sClassToHero[header.Class]
	if !ok {
		return nil, fmt.Errorf("d2s: class %v has no matching hero", header.Class)
	}

	classStats := f.asset.Records.Character.Stats[hero]
	stats := f.CreateHeroStatsState(hero, classStats)

	state, err := f.CreateHeroState(header.Name, hero, stats)
	if err != nil {
		return nil, err
	}

	state.MapSeed = header.MapSeed
	state.Imported = importedInfo(header)

	// a brand new character has no body: keep the class defaults (level 1, Act 1,
	// the class' starting skill and the left/right skills of a new game)
	if !header.HasBody() {
		state.Stats.Level = clampLevel(int(header.Level))
		state.Stats.NextLevelExp = f.asset.Records.GetExperienceBreakpoint(hero, state.Stats.Level)
		state.Items = f.starterItems(hero)

		return state, nil
	}

	body, err := d2s.ParseBody(data, nil)
	if err != nil {
		return nil, err
	}

	applyD2SAttributes(state, &body.Attributes, f)
	state.Progress = &HeroProgress{Quests: quests(body), Waypoints: body.Waypoints, NPC: *body.NPCFlags()}

	f.importD2SItems(state, data)

	if err := f.applyD2SSkills(state, hero, body.SkillPoints); err != nil {
		return nil, err
	}

	f.applyD2SActiveSkills(state, header)

	return state, nil
}

func maxInt(a, b int) int {
	if b > a {
		return b
	}

	return a
}

func clampLevel(level int) int {
	if level < 1 {
		return 1
	}

	return level
}

func importedInfo(h *d2s.Header) *ImportedInfo {
	info := &ImportedInfo{
		Hardcore:       h.IsHardcore(),
		Expansion:      h.IsExpansion(),
		Ladder:         h.IsLadder(),
		Dead:           h.IsDead(),
		WeaponSetII:    h.ActiveWeaponSet != 0,
		LastPlayed:     h.LastPlayed,
		SwapLeftSkill:  hotkeyID(h.LeftSwapSkill),
		SwapRightSkill: hotkeyID(h.RightSwapSkill),
	}

	for _, k := range h.Hotkeys {
		info.Hotkeys = append(info.Hotkeys, hotkeyID(k))
	}

	return info
}

// hotkeyID converts a stored skill id to an int, with -1 for "no skill".
func hotkeyID(id uint32) int {
	if id == d2s.NoSkill || id > d2s.NoSkill {
		return -1
	}

	return int(id)
}

// applyD2SActiveSkills selects the saved left and right mouse skills of the
// active weapon set. A skill the hero has no entry for (e.g. one granted by an
// item) is added at level 0 so the HUD can still show its icon; an unknown id
// falls back to Attack.
func (f *HeroStateFactory) applyD2SActiveSkills(state *HeroState, h *d2s.Header) {
	left, right := h.ActiveSkills()
	state.LeftSkill = f.ensureSkill(state, hotkeyID(left))
	state.RightSkill = f.ensureSkill(state, hotkeyID(right))
}

func (f *HeroStateFactory) ensureSkill(state *HeroState, id int) int {
	if id < 0 {
		return 0
	}

	if _, ok := state.Skills[id]; ok {
		return id
	}

	rec := f.asset.Records.Skill.Details[id]
	if rec == nil {
		return 0
	}

	skill, err := f.CreateHeroSkill(0, rec.Skill)
	if err != nil {
		return 0
	}

	state.Skills[skill.ID] = skill

	return skill.ID
}

func quests(body *d2s.Body) [3]d2s.QuestRecord {
	var out [3]d2s.QuestRecord
	for i := range out {
		out[i] = *body.QuestRecord(i)
	}

	return out
}

func applyD2SAttributes(state *HeroState, a *d2s.Attributes, f *HeroStateFactory) {
	s := state.Stats
	s.Level = int(a.Level)
	s.Experience = int(a.Experience)
	s.Strength = int(a.Strength)
	s.Energy = int(a.Energy)
	s.Dexterity = int(a.Dexterity)
	s.Vitality = int(a.Vitality)
	s.StatsPoints = int(a.UnusedStats)
	s.SkillPoints = int(a.UnusedSkillPoints)
	// The saved maximums are the base values (class, level, vitality/energy); the
	// saved current values already include what the equipment adds, which this
	// engine does not apply yet. Raising the maximum to the current value keeps the
	// panel and globes from showing e.g. 1241/869.
	s.Health = int(a.CurrentHP)
	s.MaxHealth = maxInt(int(a.MaxHP), s.Health)
	s.Mana = int(a.CurrentMana)
	s.MaxMana = maxInt(int(a.MaxMana), s.Mana)
	s.Stamina = float64(a.CurrentStamina)
	s.MaxStamina = maxInt(int(a.MaxStamina), int(a.CurrentStamina))
	s.NextLevelExp = f.asset.Records.GetExperienceBreakpoint(state.HeroType, s.Level)
	state.Gold = int(a.Gold)
}

// classSkillIDs returns the ids of the hero class' skills in ascending id
// order, which is the order a .d2s stores its 30 skill allocations in.
func (f *HeroStateFactory) classSkillIDs(hero d2enum.Hero) []int {
	token := strings.ToLower(hero.GetToken3())
	ids := make([]int, 0, d2sSkillsPerClass)

	for id, rec := range f.asset.Records.Skill.Details {
		if rec.Charclass == token {
			ids = append(ids, id)
		}
	}

	sort.Ints(ids)

	return ids
}

func (f *HeroStateFactory) applyD2SSkills(state *HeroState, hero d2enum.Hero, points [d2sSkillsPerClass]byte) error {
	ids := f.classSkillIDs(hero)

	for i, p := range points {
		if p == 0 || i >= len(ids) {
			continue
		}

		rec := f.asset.Records.Skill.Details[ids[i]]

		skill, err := f.CreateHeroSkill(int(p), rec.Skill)
		if err != nil {
			continue // skills without a description cannot be shown yet
		}

		state.Skills[skill.ID] = skill
	}

	return nil
}

// importD2SItems adds the equipped items. Failures only cost the equipment:
// the character is still imported.
func (f *HeroStateFactory) importD2SItems(state *HeroState, data []byte) {
	tables, err := f.loadD2SItemTables()
	if err != nil {
		fmt.Printf("d2s: item tables unavailable, skipping equipment: %v\n", err)
		return
	}

	character, err := d2s.Parse(data, tables)
	if err != nil {
		fmt.Printf("d2s: could not read items of %s, skipping equipment: %v\n", state.HeroName, err)
		return
	}

	// the starter gear of the class is only for new characters; a saved hero wears what it saved
	state.Equipment = d2inventory.CharacterEquipment{}

	f.applyD2SEquipment(state, character.Items, tables, state.Imported != nil && state.Imported.WeaponSetII)
	state.Items = f.importedItems(character.Items)
}
