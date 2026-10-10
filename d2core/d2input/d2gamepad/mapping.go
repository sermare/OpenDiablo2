package d2gamepad

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// Action is something a button can do.
type Action int

// Actions. The first group drives the mouse, the second presses the key the
// player bound to a game event (so remapped keyboard controls keep working),
// the third is handled by the game through SetActionHandler.
const (
	ActionNone Action = iota

	ActionClick      // left mouse button at the cursor (menus, inventory)
	ActionRightClick // right mouse button at the cursor
	ActionLeftSkill  // use the left skill (left mouse button)
	ActionRightSkill // use the right skill (right mouse button)

	ActionPotion1
	ActionPotion2
	ActionPotion3
	ActionPotion4
	ActionInventory
	ActionCharacter
	ActionSkills
	ActionQuests
	ActionParty
	ActionAutomap
	ActionGameMenu // Escape
	ActionShowItems

	ActionCycleLeftSkill
	ActionCycleRightSkill
	ActionPrevLeftSkill
	ActionPrevRightSkill

	actionCount
)

type actionInfo struct {
	name  string
	event d2enum.GameEvent // key actions
	key   d2enum.Key       // fallback key when the game offers no binding
}

var actionInfos = [...]actionInfo{
	ActionNone:            {name: "NONE"},
	ActionClick:           {name: "CLICK"},
	ActionRightClick:      {name: "RIGHT CLICK"},
	ActionLeftSkill:       {name: "LEFT SKILL"},
	ActionRightSkill:      {name: "RIGHT SKILL"},
	ActionPotion1:         {name: "POTION 1", event: d2enum.UseBeltSlot1, key: d2enum.Key1},
	ActionPotion2:         {name: "POTION 2", event: d2enum.UseBeltSlot2, key: d2enum.Key2},
	ActionPotion3:         {name: "POTION 3", event: d2enum.UseBeltSlot3, key: d2enum.Key3},
	ActionPotion4:         {name: "POTION 4", event: d2enum.UseBeltSlot4, key: d2enum.Key4},
	ActionInventory:       {name: "INVENTORY", event: d2enum.ToggleInventoryPanel, key: d2enum.KeyI},
	ActionCharacter:       {name: "CHARACTER", event: d2enum.ToggleCharacterPanel, key: d2enum.KeyC},
	ActionSkills:          {name: "SKILL TREE", event: d2enum.ToggleSkillTreePanel, key: d2enum.KeyT},
	ActionQuests:          {name: "QUEST LOG", event: d2enum.ToggleQuestLog, key: d2enum.KeyQ},
	ActionParty:           {name: "PARTY", event: d2enum.TogglePartyPanel, key: d2enum.KeyP},
	ActionAutomap:         {name: "AUTOMAP", event: d2enum.ToggleAutomap, key: d2enum.KeyTab},
	ActionGameMenu:        {name: "GAME MENU", key: d2enum.KeyEscape},
	ActionShowItems:       {name: "SHOW ITEMS", event: d2enum.HoldShowGroundItems, key: d2enum.KeyAlt},
	ActionCycleLeftSkill:  {name: "NEXT LEFT SKILL"},
	ActionCycleRightSkill: {name: "NEXT RIGHT SKILL"},
	ActionPrevLeftSkill:   {name: "PREV LEFT SKILL"},
	ActionPrevRightSkill:  {name: "PREV RIGHT SKILL"},
}

// String returns the upper-case name used in the options menu.
func (a Action) String() string {
	if a < 0 || a >= actionCount {
		return "?"
	}

	return actionInfos[a].name
}

// ActionNames lists every action name in order (the values of a remap row).
func ActionNames() []string {
	r := make([]string, actionCount)
	for i := range r {
		r[i] = Action(i).String()
	}

	return r
}

// ParseAction finds an action by name (case-insensitive, spaces or underscores).
func ParseAction(s string) (Action, bool) {
	s = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), "_", " "))
	for i := range actionInfos {
		if actionInfos[i].name == s {
			return Action(i), true
		}
	}

	return 0, false
}

func (a Action) isMouse() bool { return a >= ActionClick && a <= ActionRightSkill }
func (a Action) isKey() bool   { return a >= ActionPotion1 && a <= ActionShowItems }
func (a Action) isHandler() bool {
	return a >= ActionCycleLeftSkill && a <= ActionPrevRightSkill
}

// Mapping assigns an action to every button.
type Mapping [NumButtons]Action

// DefaultMapping is the layout described in docs/gamepad.md.
func DefaultMapping() Mapping {
	var m Mapping

	m[ButtonA] = ActionClick
	m[ButtonB] = ActionRightClick
	m[ButtonX] = ActionInventory
	m[ButtonY] = ActionCharacter
	m[ButtonLB] = ActionCycleLeftSkill
	m[ButtonRB] = ActionCycleRightSkill
	m[ButtonLT] = ActionRightSkill
	m[ButtonRT] = ActionLeftSkill
	m[ButtonBack] = ActionAutomap
	m[ButtonStart] = ActionGameMenu
	m[ButtonL3] = ActionSkills
	m[ButtonR3] = ActionQuests
	m[ButtonDUp] = ActionPotion1
	m[ButtonDRight] = ActionPotion2
	m[ButtonDDown] = ActionPotion3
	m[ButtonDLeft] = ActionPotion4

	return m
}

// Set binds an action to a button.
func (m *Mapping) Set(b Button, a Action) {
	if b >= 0 && int(b) < NumButtons && a >= 0 && a < actionCount {
		m[b] = a
	}
}

// ButtonsFor lists the buttons that trigger an action.
func (m Mapping) ButtonsFor(a Action) []Button {
	var r []Button

	for i, act := range m {
		if act == a {
			r = append(r, Button(i))
		}
	}

	return r
}

// MarshalJSON writes {"A":"CLICK",...}.
func (m Mapping) MarshalJSON() ([]byte, error) {
	o := make(map[string]string, NumButtons)
	for i, a := range m {
		o[Button(i).String()] = a.String()
	}

	return json.Marshal(o)
}

// UnmarshalJSON reads the MarshalJSON form; buttons that are absent or name an
// unknown action keep their current value, so a partial file only overrides.
func (m *Mapping) UnmarshalJSON(data []byte) error {
	var o map[string]string
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}

	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		b, okB := ParseButton(k)
		a, okA := ParseAction(o[k])

		if okB && okA {
			m[b] = a
		}
	}

	return nil
}

// OptionKey is the configuration option key of the remap row of a button.
func OptionKey(b Button) string { return "pad." + strings.ToLower(b.String()) }

// MappingFromChoices builds a mapping from the stored choices: get returns the
// chosen action name of a button's option key ("" = default).
func MappingFromChoices(get func(optionKey string) string) Mapping {
	m := DefaultMapping()

	for _, b := range Buttons() {
		if a, ok := ParseAction(get(OptionKey(b))); ok {
			m[b] = a
		}
	}

	return m
}
