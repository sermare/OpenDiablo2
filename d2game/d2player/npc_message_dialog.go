package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The modal message of the vendor window (UI_ApplyNpcTransactionResultA 0x4b1c50 / B 0x4b1ea0, VERIFIED): for every
// server reply code above 6 the exe destroys the pending dialog, then builds one dialog (UI_CreateDialog) with the
// message text as an entry of font 0x15 and a single button entry whose caption is string 0xd4b ("Continue"; the
// shop text calls it OK), whose callback (0x4b1520) closes it and returns the trade state to the shop. The dialog is
// modal: the shop ignores clicks until it is gone. The frame is the gold dialog's picture "dialogbackground"; the
// position is UNVERIFIED (the exe centres it on the NPC screen rect: UI_GetNpcInteractionScreenRect), the Go
// dialog uses the gold dialog's anchor.

const messageDialogChars = 24 // wrap width of the text in characters (U: the exe wraps by pixel width)

// MessageDialogRects is the pure layout of the message dialog.
type MessageDialogRects struct {
	Frame, Button UIRect
	TextX         int   // centre of the text lines (VERIFIED rule: DialogLineX)
	TextBaselines []int // one per line
}

// MessageDialog lays out a dialog for a frame of frameW x frameH, nLines text lines and a button of btnW x btnH.
func MessageDialog(frameW, frameH, nLines, btnW, btnH int) MessageDialogRects {
	g := GoldDialogSized(GoldDlgDrop, frameW, frameH, GoldDlgSizes{OkW: btnW, OkH: btnH})

	r := MessageDialogRects{
		Frame:  g.Frame,
		Button: UIRect{"msgdlg", "button", g.Frame.X + (frameW-btnW)/2, g.Ok.Y, btnW, btnH},
		TextX:  g.Frame.X + frameW/2 + 1,
	}

	for i := 0; i < nLines; i++ {
		r.TextBaselines = append(r.TextBaselines, g.TextBaseline0+i*g.RowStep)
	}

	return r
}

// NpcMessageDialog is the modal message with one button.
type NpcMessageDialog struct {
	asset *d2asset.AssetManager
	ui    *d2ui.UIManager

	group  *d2ui.WidgetGroup
	frame  *d2ui.Sprite
	button *d2ui.Button
	lines  []*d2ui.Label

	isOpen  bool
	text    string
	onClose func()
}

// NewNpcMessageDialog creates the dialog; Load builds its widgets.
func NewNpcMessageDialog(asset *d2asset.AssetManager, ui *d2ui.UIManager) *NpcMessageDialog {
	return &NpcMessageDialog{asset: asset, ui: ui}
}

const messageDialogMaxLines = 6

// Load builds the widgets.
func (m *NpcMessageDialog) Load() {
	m.group = m.ui.NewWidgetGroup(d2ui.RenderPriorityInventory)

	if sp, err := m.ui.NewSprite(d2resource.MoveGoldDialog, d2resource.PaletteSky); err == nil {
		m.frame = sp
		m.group.AddWidget(sp)
	}

	caption := m.asset.TranslateString("Continue") // string id 0xd4b (the key is U); the text is the same
	if caption == "" {
		caption = "Continue"
	}

	m.button = m.ui.NewButton(d2ui.ButtonTypeShort, caption)
	m.button.OnActivated(func() { m.Close() })
	m.group.AddWidget(m.button)

	for i := 0; i < messageDialogMaxLines; i++ {
		l := m.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.Alignment = d2ui.HorizontalAlignCenter
		m.lines = append(m.lines, l)
		m.group.AddWidget(l)
	}

	m.group.SetVisible(false)
}

// SetOnClose sets the callback run when the dialog is dismissed.
func (m *NpcMessageDialog) SetOnClose(cb func()) { m.onClose = cb }

// IsOpen reports whether the dialog is up (the shop is blocked meanwhile).
func (m *NpcMessageDialog) IsOpen() bool { return m != nil && m.isOpen }

// Text is the message shown ("" when closed).
func (m *NpcMessageDialog) Text() string {
	if !m.IsOpen() {
		return ""
	}

	return m.text
}

// Open shows the message.
func (m *NpcMessageDialog) Open(text string) {
	if m == nil || m.group == nil {
		return
	}

	m.isOpen, m.text = true, text
	lines := d2util.SplitIntoLinesWithMaxWidth(text, messageDialogChars)

	if len(lines) > messageDialogMaxLines {
		lines = lines[:messageDialogMaxLines]
	}

	fw, fh := 0, 0
	if m.frame != nil {
		fw, fh = m.frame.GetCurrentFrameSize()
	}

	bw, bh := m.button.GetSize()
	lay := MessageDialog(fw, fh, len(lines), bw, bh)

	if m.frame != nil {
		m.frame.SetPosition(lay.Frame.X, lay.Frame.Y+lay.Frame.H)
	}

	m.button.SetPosition(lay.Button.X, lay.Button.Y)

	for i, l := range m.lines {
		if i < len(lines) {
			l.SetText(d2ui.ColorTokenize(lines[i], d2ui.ColorTokenGold))
			l.SetPosition(lay.TextX, lay.TextBaselines[i])
			l.SetVisible(true)
		} else {
			l.SetText("")
			l.SetVisible(false)
		}
	}

	m.group.SetVisible(true)
}

// Close dismisses the dialog.
func (m *NpcMessageDialog) Close() {
	if !m.IsOpen() {
		return
	}

	m.isOpen, m.text = false, ""
	m.group.SetVisible(false)

	if m.onClose != nil {
		m.onClose()
	}
}

// Render draws nothing itself (the widget group is drawn by the UI manager); kept for symmetry with the panels.
func (m *NpcMessageDialog) Render(_ d2interface.Surface) {}
