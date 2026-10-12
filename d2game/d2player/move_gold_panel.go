package d2player

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// Positions come from the original's dialog (move_gold_layout.go, UI_OpenPanelByIndex 0x44f7b0). The values
// below are the layout of GoldDialog for the drop kind with the real picture size filled in at Load.
const (
	moveGoldActionLabelOffsetY = goldDlgTextRowAdvance
)

const goldValueFilter = "0123456789"

// NewMoveGoldPanel creates a new move gold panel
func NewMoveGoldPanel(asset *d2asset.AssetManager,
	ui *d2ui.UIManager,
	gold int,
	l d2util.LogLevel,
) *MoveGoldPanel {
	originX := 0
	originY := 0

	mgp := &MoveGoldPanel{
		asset:     asset,
		uiManager: ui,
		originX:   originX,
		originY:   originY,
		gold:      gold,
	}

	mgp.Logger = d2util.NewLogger()
	mgp.Logger.SetLevel(l)
	mgp.Logger.SetPrefix(logPrefix)

	return mgp
}

// MoveGoldPanel represents the move gold panel
type MoveGoldPanel struct {
	asset        *d2asset.AssetManager
	uiManager    *d2ui.UIManager
	panel        *d2ui.Sprite
	onCloseCb    func()
	panelGroup   *d2ui.WidgetGroup
	gold         int
	actionLabel1 *d2ui.Label
	actionLabel2 *d2ui.Label
	value        *d2ui.TextBox
	ok, cancel   *d2ui.Button

	// kind and limit describe the open dialog (UI_OpenPanelByIndex 0x44f7b0): the kind picks the text and the
	// request, the limit is the purse the amount is taken from. onConfirm gets the amount of a pressed OK.
	kind      d2inventory.GoldDialogKind
	limit     int
	onConfirm func(kind d2inventory.GoldDialogKind, amount int)

	originX int
	originY int
	isOpen  bool

	*d2util.Logger
}

// Load the data for the move gold panel
func (s *MoveGoldPanel) Load() {
	var err error

	s.panelGroup = s.uiManager.NewWidgetGroup(d2ui.RenderPriorityInventory)

	s.panel, err = s.uiManager.NewSprite(d2resource.MoveGoldDialog, d2resource.PaletteSky)
	if err != nil {
		s.Error(err.Error())
	}

	fw, fh := s.panel.GetCurrentFrameSize()
	lay := GoldDialog(GoldDlgDrop, fw, fh)

	// pictures stand on their bottom edge
	s.panel.SetPosition(lay.Frame.X, lay.Frame.Y+lay.Frame.H)
	s.panelGroup.AddWidget(s.panel)

	closeButton := s.uiManager.NewButton(d2ui.ButtonTypeSquareClose, "")
	okButton := s.uiManager.NewButton(d2ui.ButtonTypeSquareOk, "")

	// the exe sizes both buttons from their picture frames (UI_ButtonImpl_Construct 0x4b77b0) and stands them on
	// their bottom edge, so the top depends on the real frame height
	var sz GoldDlgSizes

	sz.OkW, sz.OkH = okButton.GetSize()
	sz.CancelW, sz.CancelH = closeButton.GetSize()
	lay = GoldDialogSized(GoldDlgDrop, fw, fh, sz)

	s.ok, s.cancel = okButton, closeButton
	closeButton.SetVisible(false)
	closeButton.SetPosition(lay.Cancel.X, lay.Cancel.Y)
	closeButton.OnActivated(func() { s.Close() })
	s.panelGroup.AddWidget(closeButton)

	okButton.SetVisible(false)
	okButton.SetPosition(lay.Ok.X, lay.Ok.Y)
	okButton.OnActivated(func() { s.action() })
	s.panelGroup.AddWidget(okButton)

	s.value = s.uiManager.NewTextbox()
	s.value.SetFilter(goldValueFilter)
	s.value.SetText("")
	s.value.Activate()
	s.value.SetNumberOnly(s.gold)
	s.value.SetPosition(lay.Edit.X, lay.Edit.Y+lay.Edit.H)
	s.panelGroup.AddWidget(s.value)

	s.actionLabel1 = s.uiManager.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	s.actionLabel1.Alignment = d2ui.HorizontalAlignCenter
	// VERIFIED (UI_DialogMeasureEntryWidthSimple 0x4b4cd0): a dialog line starts at nX + (W - textW + 1)/2 + 1, i.e.
	// it is centred on the dialog picture one pixel to the right of its middle
	labelX := lay.Frame.X + lay.Frame.W/2 + 1
	labelY := lay.TextBaseline0 - 16
	s.actionLabel1.SetPosition(labelX, labelY)
	s.panelGroup.AddWidget(s.actionLabel1)

	s.actionLabel2 = s.uiManager.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	s.actionLabel2.Alignment = d2ui.HorizontalAlignCenter
	s.actionLabel2.SetPosition(labelX, labelY+moveGoldActionLabelOffsetY)
	s.panelGroup.AddWidget(s.actionLabel2)

	increase := s.uiManager.NewButton(d2ui.ButtonTypeUpArrow, d2resource.PaletteSky)
	increase.SetPosition(lay.Spinner.X, lay.Spinner.Y+15)
	increase.SetVisible(false)
	increase.OnActivated(func() { s.increase() })
	s.panelGroup.AddWidget(increase)

	decrease := s.uiManager.NewButton(d2ui.ButtonTypeDownArrow, d2resource.PaletteSky)
	decrease.SetPosition(lay.Spinner.X, lay.Spinner.Y+30)
	decrease.SetVisible(false)
	decrease.OnActivated(func() { s.decrease() })
	s.panelGroup.AddWidget(decrease)

	s.setActionText()

	s.panelGroup.SetVisible(false)
}

// SetOnConfirm sets the receiver of the amount of a pressed OK.
func (s *MoveGoldPanel) SetOnConfirm(cb func(kind d2inventory.GoldDialogKind, amount int)) {
	s.onConfirm = cb
}

// Amount is the amount typed in the edit box, held to the digits and the limit of the edit box.
func (s *MoveGoldPanel) Amount() int {
	return d2inventory.ClampGoldInput(strings.TrimSpace(s.value.GetText()), s.limit)
}

func (s *MoveGoldPanel) action() {
	amount := s.Amount()

	if d2inventory.GoldDialogRequest(s.kind, amount) && s.onConfirm != nil {
		s.onConfirm(s.kind, amount)
	}

	s.Close()
}

func (s *MoveGoldPanel) setAmount(v int) {
	if v < 0 {
		v = 0
	}

	if v > s.limit {
		v = s.limit
	}

	s.value.SetText(strconv.Itoa(v))
}

func (s *MoveGoldPanel) increase() { s.setAmount(s.Amount() + 1) }

func (s *MoveGoldPanel) decrease() { s.setAmount(s.Amount() - 1) }

// Texts of the four dialog kinds (string ids 0xfc1, 0xfce, 0xfd1, 0xfd2) with their English fallbacks.
var goldDialogTexts = map[d2inventory.GoldDialogKind][2]string{ //nolint:gochecknoglobals // static
	d2inventory.GoldDialogDrop:     {"strDropGoldHowMuch", "How much Gold would you like to drop?"},
	d2inventory.GoldDialogTrade:    {"strOfferGoldHowMuch", "How much Gold would you like to offer?"},
	d2inventory.GoldDialogDeposit:  {"strBankGoldDeposit", "How much Gold would you like to deposit?"},
	d2inventory.GoldDialogWithdraw: {"strBankGoldWithdraw", "How much Gold would you like to withdraw?"},
}

// GoldDialogText is the question of a dialog kind.
func GoldDialogText(tr func(string) string, kind d2inventory.GoldDialogKind) string {
	e := goldDialogTexts[kind]
	if s := tr(e[0]); s != "" && s != e[0] {
		return s
	}

	return e[1]
}

func (s *MoveGoldPanel) setActionText() {
	lines := d2util.SplitIntoLinesWithMaxWidth(GoldDialogText(func(k string) string { return s.asset.TranslateString(k) }, s.kind), 20)
	for len(lines) < 2 {
		lines = append(lines, "")
	}

	s.actionLabel1.SetText(d2ui.ColorTokenize(lines[0], d2ui.ColorTokenGold))
	s.actionLabel2.SetText(d2ui.ColorTokenize(lines[1], d2ui.ColorTokenGold))
}

// IsOpen returns true if the move gold panel is opened
func (s *MoveGoldPanel) IsOpen() bool {
	return s.isOpen
}

// Toggle toggles the visibility of the move gold panel
func (s *MoveGoldPanel) Toggle() {
	if s.isOpen {
		s.Close()
	} else {
		s.Open()
	}
}

// Open opens the move gold panel as the drop dialog of the carried gold.
func (s *MoveGoldPanel) Open() {
	s.OpenKind(d2inventory.GoldDialogDrop, s.gold, 0)
}

// OpenKind opens the dialog of a kind (UI_OpenPanelByIndex 0x44f7b0): limit is the purse it takes from
// (d2inventory.GoldDialogLimit); the deposit opens with the whole limit typed in (d2inventory.GoldDialogPrefill).
func (s *MoveGoldPanel) OpenKind(kind d2inventory.GoldDialogKind, limit, _ int) {
	s.kind, s.limit = kind, limit
	s.value.SetNumberOnly(limit)

	if d2inventory.GoldDialogPrefill(kind) {
		s.setAmount(limit)
	} else {
		s.value.SetText("")
	}

	s.setActionText()
	s.isOpen = true
	s.panelGroup.SetVisible(true)
}

// Close closed the move gold panel
func (s *MoveGoldPanel) Close() {
	s.isOpen = false
	s.panelGroup.SetVisible(false)

	if s.onCloseCb != nil {
		s.onCloseCb()
	}
}

// SetOnCloseCb the callback run on closing the HeroStatsPanel
func (s *MoveGoldPanel) SetOnCloseCb(cb func()) {
	s.onCloseCb = cb
}
