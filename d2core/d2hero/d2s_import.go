package d2hero

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
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
	state.Expansion, state.Hardcore, state.Ladder = header.IsExpansion(), header.IsHardcore(), header.IsLadder()

	if header.IsDead() {
		state.Death = &DeathState{Died: true}
	}

	state.D2SBase = append([]byte(nil), data...)
	state.Merc = MercFromHeader(header.Mercenary)

	if diff, _, ok := header.ActiveDifficulty(); ok {
		state.Difficulty = d2enum.DifficultyType(diff)
	}

	// a brand new character has no body: keep the class defaults
	if !header.HasBody() {
		state.Containers = f.StartingContainers(hero)

		return state, nil
	}

	body, err := d2s.ParseBody(data, nil)
	if err != nil {
		return nil, err
	}

	applyD2SAttributes(state, &body.Attributes, f)
	state.Progress = &HeroProgress{Quests: quests(body), Waypoints: body.Waypoints, NPC: *body.NPCFlags()}

	f.importD2SItems(state, data)
	f.giveStartingItemsToFreshHero(state, hero)

	if err := f.applyD2SSkills(state, hero, body.SkillPoints); err != nil {
		return nil, err
	}

	applyD2SSkillBar(state, header)

	f.RecalcStats(state)
	fmt.Printf("stats: %s %s\n", state.HeroName, StatsSummary(state.Stats))

	return state, nil
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
	s.Health = int(a.CurrentHP)
	s.MaxHealth = int(a.MaxHP)
	s.Mana = int(a.CurrentMana)
	s.MaxMana = int(a.MaxMana)
	s.Stamina = float64(a.CurrentStamina)
	s.MaxStamina = int(a.MaxStamina)
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

	f.applyD2SEquipment(state, character.Items, tables)
	f.applyD2SContainers(state, character.Items)

	if state.Containers != nil {
		importEquipped(state.Containers, data, character.Items, func(code string) bool { return f.asset.Records.Item.All[code] != nil })
	}
}

// applyD2SContainers puts the inventory (page 1), cube (4), stash (5) and belt
// items of a save into the hero's containers. Items without an OpenDiablo2
// record are skipped with a warning.
func (f *HeroStateFactory) applyD2SContainers(state *HeroState, items []d2s.Item) {
	known := func(code string) bool { return f.asset.Records.Item.All[code] != nil }
	containers := &HeroContainers{Items: []StoredItem{}}

	for i := range items {
		if it := &items[i]; it.Location == d2s.LocationEquipped && it.Equipped == d2sSlotBelt && known(trimCode(it.Code)) {
			containers.BeltCode = trimCode(it.Code)
		}

		stored, skip := StoredFromD2S(&items[i], known)
		if skip == "" {
			containers.Items = append(containers.Items, stored)
			continue
		}

		// equipped items and the like are not a container's business
		if items[i].Location == d2s.LocationStored || items[i].Location == d2s.LocationBelt {
			fmt.Printf("d2s: skipping item %q of %s: %s\n", items[i].Code, state.HeroName, skip)
		}
	}

	state.Containers = containers
	fmt.Printf("d2s: %s containers: inventory=%d belt=%d cube=%d stash=%d\n", state.HeroName,
		len(containers.Page(PageInventory)), len(containers.Page(PageBelt)),
		len(containers.Page(PageCube)), len(containers.Page(PageStash)))
}

// applyD2SSkillBar reads the assigned skills (hotkeys), the left/right skill
// and the swap-set skills of the header. The active skills must be skills the
// hero has; anything else falls back to Attack (id 0).
func applyD2SSkillBar(state *HeroState, header *d2s.Header) {
	bar := SkillBarFromBlock(header.SkillBlock())
	state.SkillBar = bar

	has := func(id int) int {
		if s := state.Skills[id]; s != nil && s.SkillPoints > 0 {
			return id
		}

		return 0
	}

	state.LeftSkill, state.RightSkill = has(bar.Left.Skill), has(bar.Right.Skill)
	fmt.Printf("d2s: %s skills left=%d right=%d swap=%d/%d hotkeys=%v\n", state.HeroName,
		bar.Left.Skill, bar.Right.Skill, bar.LeftSwap.Skill, bar.RightSwap.Skill, hotkeyIDs(bar))
}

func hotkeyIDs(b *SkillBar) []int {
	out := make([]int, len(b.Hotkeys))
	for i, s := range b.Hotkeys {
		out[i] = s.Skill
	}

	return out
}
