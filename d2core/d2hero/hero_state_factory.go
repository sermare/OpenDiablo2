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
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
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
	d2sTables *d2s.ItemTables // loaded on first use by SaveD2S
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

	known := make(map[string]bool, len(existing))
	for _, h := range existing {
		known[strings.ToLower(h.HeroName)] = true
	}

	imported := make([]*HeroState, 0)

	for _, dir := range dirs {
		imported = append(imported, f.importD2SDir(dir, known)...)
	}

	return imported
}

func (f *HeroStateFactory) importD2SDir(dir string, known map[string]bool) []*HeroState {
	files, _ := ioutil.ReadDir(dir)
	imported := make([]*HeroState, 0)

	for _, file := range files {
		name := file.Name()
		if file.IsDir() || !strings.EqualFold(filepath.Ext(name), ".d2s") ||
			known[strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))] {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Clean(filepath.Join(dir, name)))
		if err != nil {
			continue
		}

		state, err := f.ImportD2S(data)
		if err != nil {
			fmt.Printf("could not import %s: %v\n", name, err)
			continue
		}

		if known[strings.ToLower(state.HeroName)] {
			continue
		}

		if err := f.Save(state); err != nil {
			fmt.Printf("could not save imported hero %s: %v\n", state.HeroName, err)
			continue
		}

		known[strings.ToLower(state.HeroName)] = true

		imported = append(imported, state)
	}

	return imported
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

// HasGameStates returns true if the player has any previously saved game
func (f *HeroStateFactory) HasGameStates() bool {
	basePath, _ := f.getGameBaseSavePath()
	files, _ := ioutil.ReadDir(basePath)

	return len(files) > 0
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
		hs.SkillDescriptionRecord = f.asset.Records.Skill.Descriptions[hs.SkillRecord.Skilldesc]
		hs.SkillPoints = hs.Shallow.SkillPoints
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
	if err := ioutil.WriteFile(state.FilePath, fileJSON, writefilePermission); err != nil {
		return err
	}

	return nil
}
