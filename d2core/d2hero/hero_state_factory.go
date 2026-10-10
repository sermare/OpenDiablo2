package d2hero

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
)

const (
	mkdirPermission     = 0750
	writefilePermission = 0600
)

// NewHeroStateFactory creates a new HeroStateFactory and initializes it.
func NewHeroStateFactory(asset *d2asset.AssetManager) (*HeroStateFactory, error) {
	inventoryItemFactory, err := d2inventory.NewInventoryItemFactory(asset)
	if err != nil {
		return nil, err
	}

	factory := &HeroStateFactory{
		asset:                asset,
		InventoryItemFactory: inventoryItemFactory,
	}

	return factory, nil
}

// HeroStateFactory is responsible for creating player state objects
type HeroStateFactory struct {
	asset *d2asset.AssetManager
	*d2inventory.InventoryItemFactory
	d2sTables *d2s.ItemTables  // loaded on first use by SaveD2S
	statBases d2statlist.Bases // armor/weapon base data for the stat list, loaded on first use
	sets      *setEnv          // set bonus table of the stat list, loaded on first use

	// lastEquipStatus is the verdict of the last activation pass (for logs).
	lastEquipStatus []EquipStatus
	equipTried      bool // EquipRules was attempted
	equipRules      d2equip.Rules
	equipBases      d2equip.Bases
}

// CreateHeroState creates a HeroState instance and returns a pointer to it
func (f *HeroStateFactory) CreateHeroState(
	heroName string,
	hero d2enum.Hero,
	statsState *HeroStatsState,
) (*HeroState, error) {
	result := &HeroState{
		HeroName:  heroName,
		HeroType:  hero,
		Act:       1,
		Stats:     statsState,
		Equipment: f.DefaultHeroItems[hero],
		FilePath:  "",
	}

	defaultStats := f.asset.Records.Character.Stats[hero]
	skillState, err := f.CreateHeroSkillsState(defaultStats, hero)

	if err != nil {
		return nil, err
	}

	result.Skills = skillState

	return result, nil
}

// GetAllHeroStates returns all player saves
func (f *HeroStateFactory) GetAllHeroStates() ([]*HeroState, error) {
	basePath, _ := f.getGameBaseSavePath()
	files, _ := ioutil.ReadDir(basePath)
	result := make([]*HeroState, 0)

	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() || len(fileName) < 5 || !strings.EqualFold(fileName[len(fileName)-4:], ".od2") {
			continue
		}

		gameState := f.LoadHeroState(filepath.Join(basePath, file.Name()))
		if gameState == nil || gameState.HeroType == d2enum.HeroNone {

		} else if gameState.Stats == nil || gameState.Skills == nil {
			// temporarily loading default class stats if the character was created before saving stats/skills was introduced
			// to be removed in the future
			classStats := f.asset.Records.Character.Stats[gameState.HeroType]
			gameState.Stats = f.CreateHeroStatsState(gameState.HeroType, classStats)

			skillState, err := f.CreateHeroSkillsState(classStats, gameState.HeroType)
			if err != nil {
				return nil, err
			}

			gameState.Skills = skillState

			if err := f.Save(gameState); err != nil {
				fmt.Printf("failed to save game state!, err: %v\n", err)
			}
		}

		result = append(result, gameState)
	}

	return append(result, f.importD2SCharacters(result)...), nil
}

// importD2SCharacters imports the real Diablo II characters (.d2s files) found
// in the directory named by OD2_D2S_DIR that are not already in the list. The
// originals are only read; each import is saved as a new .od2 file.
func (f *HeroStateFactory) importD2SCharacters(existing []*HeroState) []*HeroState {
	dirs := ImportDirs()
	if len(dirs) == 0 {
		return nil
	}

	byName := make(map[string]*HeroState, len(existing))
	for _, h := range existing {
		byName[strings.ToLower(h.HeroName)] = h
	}

	imported := make([]*HeroState, 0)

	for _, dir := range dirs {
		imported = append(imported, f.importD2SDir(dir, byName)...)
	}

	return imported
}

func (f *HeroStateFactory) importD2SDir(dir string, byName map[string]*HeroState) []*HeroState {
	files, _ := ioutil.ReadDir(dir)
	imported := make([]*HeroState, 0)

	for _, file := range files {
		name := file.Name()
		if file.IsDir() || !strings.EqualFold(filepath.Ext(name), ".d2s") {
			continue
		}

		path := filepath.Clean(filepath.Join(dir, name))

		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		state, err := f.ImportD2S(data)
		if err != nil {
			fmt.Printf("could not import %s: %v\n", name, err)
			continue
		}

		state.Imported.Source = path

		prev := byName[strings.ToLower(state.HeroName)]
		if prev != nil && !isImportOf(prev, state) {
			continue // an engine-made hero of the same name: leave it alone
		}

		if prev != nil {
			// an earlier import of this character: refresh it in place
			state.FilePath = prev.FilePath
		}

		if err := f.Save(state); err != nil {
			fmt.Printf("could not save imported hero %s: %v\n", state.HeroName, err)
			continue
		}

		if prev != nil {
			*prev = *state
			continue
		}

		byName[strings.ToLower(state.HeroName)] = state

		imported = append(imported, state)
	}

	return imported
}

// isImportOf reports whether an existing hero is an earlier import of the same
// character: flagged as imported, or (saved before that flag existed) carrying
// the same name and the level generator seed of the .d2s, which no hero made in
// this engine has.
func isImportOf(prev, state *HeroState) bool {
	if !strings.EqualFold(prev.HeroName, state.HeroName) {
		return false
	}

	return prev.Imported != nil || (state.MapSeed != 0 && prev.MapSeed == state.MapSeed)
}

// SaveImported saves an imported hero. A character imported before (same name)
// keeps its .od2 file, so importing the same .d2s again does not pile up copies.
func (f *HeroStateFactory) SaveImported(state *HeroState) error {
	if state.FilePath == "" {
		basePath, _ := f.getGameBaseSavePath()
		files, _ := ioutil.ReadDir(basePath)

		for _, file := range files {
			if file.IsDir() || !strings.EqualFold(filepath.Ext(file.Name()), ".od2") {
				continue
			}

			prev := f.LoadHeroState(filepath.Join(basePath, file.Name()))
			if prev != nil && isImportOf(prev, state) {
				state.FilePath = prev.FilePath
				break
			}
		}
	}

	return f.Save(state)
}

// CreateHeroSkillsState will assemble the hero skills from the class stats record.
func (f *HeroStateFactory) CreateHeroSkillsState(classStats *d2records.CharStatRecord, heroType d2enum.Hero) (map[int]*HeroSkill, error) {
	baseSkills := map[int]*HeroSkill{}

	for idx := range classStats.BaseSkill {
		skillName := &classStats.BaseSkill[idx]

		if *skillName == "" {
			continue
		}

		skill, err := f.CreateHeroSkill(1, *skillName)
		if err != nil {
			continue
		}

		baseSkills[skill.ID] = skill
	}

	skillList := f.asset.Records.Skill.Details
	token := strings.ToLower(heroType.GetToken3())

	for idx := range skillList {
		if skillList[idx].Charclass == token {
			skill, _ := f.CreateHeroSkill(0, skillList[idx].Skill)
			baseSkills[skill.ID] = skill
		}
	}

	skillRecord, err := f.CreateHeroSkill(1, "Attack")
	if err != nil {
		return nil, err
	}

	baseSkills[skillRecord.ID] = skillRecord

	return baseSkills, nil
}

// CreateHeroSkill creates an instance of a skill
func (f *HeroStateFactory) CreateHeroSkill(points int, name string) (*HeroSkill, error) {
	skillRecord := f.asset.Records.GetSkillByName(name)
	if skillRecord == nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	skillDescRecord, found := f.asset.Records.Skill.Descriptions[skillRecord.Skilldesc]
	if !found {
		return nil, fmt.Errorf("skill Description not found: %s", name)
	}

	result := &HeroSkill{
		SkillPoints:            points,
		SkillRecord:            skillRecord,
		SkillDescriptionRecord: skillDescRecord,
		Shallow:                &shallowHeroSkill{SkillID: skillRecord.ID, SkillPoints: points},
	}

	return result, nil
}

// HasGameStates returns true if the player has any character to list: a saved
// hero of this engine or a real .d2s in one of the import folders. (Without the
// second check a player who only has Diablo II saves would land on the
// character creation screen.)
func (f *HeroStateFactory) HasGameStates() bool {
	basePath, _ := f.getGameBaseSavePath()
	files, _ := ioutil.ReadDir(basePath)

	if len(files) > 0 {
		return true
	}

	for _, dir := range ImportDirs() {
		entries, _ := ioutil.ReadDir(dir)
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".d2s") {
				return true
			}
		}
	}

	return false
}

// CreateTestGameState is used for the map engine previewer
func (f *HeroStateFactory) CreateTestGameState() *HeroState {
	result := &HeroState{}
	return result
}

// LoadHeroState loads the player state from the file
func (f *HeroStateFactory) LoadHeroState(filePath string) *HeroState {
	strData, err := ioutil.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil
	}

	result := &HeroState{
		FilePath: filePath,
	}

	err = json.Unmarshal(strData, result)
	if err != nil {
		return nil
	}

	// Here, we turn the Shallow skill data back into records from the asset manager.
	// This is because this factory has a reference to the asset manager with loaded records.
	// We cant do this while unmarshalling because there is no reference to the asset manager.
	for idx := range result.Skills {
		hs := result.Skills[idx]

		if hs == nil {
			continue
		}

		hs.SkillRecord = f.asset.Records.Skill.Details[hs.Shallow.SkillID]
		if hs.SkillRecord != nil { // a skill id this game data does not have: keep the entry without records
			hs.SkillDescriptionRecord = f.asset.Records.Skill.Descriptions[hs.SkillRecord.Skilldesc]
		}
		hs.SkillPoints = hs.Shallow.SkillPoints
	}

	if result.Stats != nil && f.asset.Records.Character.Stats[result.HeroType] != nil {
		f.RecalcStats(result)
	}

	return result
}

func (f *HeroStateFactory) getGameBaseSavePath() (string, error) {
	configDir := d2config.ConfigDir()
	if configDir == "" {
		return "", errors.New("no user config directory")
	}

	return filepath.Join(configDir, "Saves"), nil
}

func (f *HeroStateFactory) getFirstFreeFileName() string {
	i := 0
	basePath, _ := f.getGameBaseSavePath()

	for {
		filePath := filepath.Join(basePath, strconv.Itoa(i)+".od2")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return filePath
		}
		i++
	}
}

// Save saves the player state to a file
func (f *HeroStateFactory) Save(state *HeroState) error {
	if state.FilePath == "" {
		state.FilePath = f.getFirstFreeFileName()
	}

	if err := os.MkdirAll(filepath.Dir(state.FilePath), mkdirPermission); err != nil {
		return err
	}

	fileJSON, _ := json.MarshalIndent(state, "", "   ")

	return writeFileAtomic(state.FilePath, fileJSON)
}

// DeletedSuffix is appended to the name of a real .d2s that the character
// select screen deletes ("Name.d2s.deleted"). The original game removes the
// file; this keeps a copy the player can rename back.
const DeletedSuffix = ".deleted"

// NameTaken reports whether a character of that name exists already (as a
// listed hero or as a .d2s in the folder a new one would be written to).
func (f *HeroStateFactory) NameTaken(name string) bool {
	if states, err := f.GetAllHeroStates(); err == nil {
		for _, st := range states {
			if strings.EqualFold(st.HeroName, name) {
				return true
			}
		}
	}

	_, err := os.Stat(newD2SPath(&HeroState{HeroName: name}))

	return err == nil
}

// DeleteHero removes a hero from the character list: its .od2 file and, for a
// hero that lives in a real .d2s, that file (renamed to .d2s.deleted, since it
// would be imported again otherwise). The .d2s.bak backup is left alone.
func (f *HeroStateFactory) DeleteHero(state *HeroState) error {
	if state == nil {
		return nil
	}

	if state.FilePath != "" {
		if err := os.Remove(state.FilePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	if state.Imported != nil && state.Imported.Source != "" {
		src := state.Imported.Source
		if err := os.Rename(src, src+DeletedSuffix); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
