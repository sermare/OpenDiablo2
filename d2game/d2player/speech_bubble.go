package d2player

import (
	"image/color"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const (
	speechMaxChars   = 58
	speechLineHeight = 18
	speechPadding    = 8
	speechWidth      = 520
	speechBottom     = 470 // bottom edge on the 800x600 screen, above the control panel
	noticeTop        = 70
	noticeWidth      = 300
)

//nolint:gochecknoglobals // colours
var (
	speechBackground = color.RGBA{R: 0, G: 0, B: 0, A: 190}
	speechSpeaker    = color.RGBA{R: 255, G: 215, B: 0, A: 255}
)

// SpeechBubble shows the text of an NPC's spoken line (a subtitle) and short
// notices such as "quest log updated". Both disappear on their own.
type SpeechBubble struct {
	ui *d2ui.UIManager

	speaker string
	name    *d2ui.Label
	lines   []*d2ui.Label
	left    float64 // seconds left of the speech
	age     float64 // seconds since Show

	notice     *d2ui.Label
	noticeLeft float64
}

// NoticePlacement is the top left corner and the width of the notice box for a notice text of textW pixels
// on a screen of the given logical size: centred on the screen, noticeTop pixels from the screen top.
func NoticePlacement(screen d2display.Size, textW int) (x, y, boxW int) {
	boxW = textW + 2*speechPadding

	return (screen.W - boxW) / 2, noticeTop, boxW
}

// SpeechPlacement is the top left corner and the size of the subtitle box for n text lines, in column space
// (the 800x600 interface column, which AnchorBottom centres on the screen and stands on its bottom edge). The
// box is centred on the column and ends speechBottom pixels from the column top, i.e. 130 above the HUD bar, so on a
// wider or taller screen it stays centred over the bar and clear of it. OD2 design values, not from the exe (the
// original has no subtitle box).
func SpeechPlacement(n int) (x, y, w, h int) {
	h = (n+1)*speechLineHeight + 2*speechPadding

	return (d2display.BaseW - speechWidth) / 2, speechBottom - h, speechWidth, h
}

// NewSpeechBubble creates an empty bubble.
func NewSpeechBubble(ui *d2ui.UIManager) *SpeechBubble {
	return &SpeechBubble{ui: ui}
}

// Show displays text spoken by speaker for the given number of seconds.
func (s *SpeechBubble) Show(speaker, text string, seconds float64) {
	s.speaker = speaker
	s.lines = s.lines[:0]
	s.name = s.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	s.name.SetText(speaker)
	s.name.Color[0] = speechSpeaker

	lines := d2util.SplitIntoLinesWithMaxWidth(strings.ReplaceAll(text, "\n", " "), speechMaxChars)
	for _, l := range lines {
		label := s.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		label.SetText(l)
		s.lines = append(s.lines, label)
	}

	s.left = seconds
	s.age = 0
}

// Notice displays a one line message near the top of the screen.
func (s *SpeechBubble) Notice(text string, seconds float64) {
	s.notice = s.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	s.notice.SetText(text)
	s.notice.Color[0] = speechSpeaker
	s.noticeLeft = seconds
}

// Clear removes the speech at once (the hero left the NPC: another level).
func (s *SpeechBubble) Clear() { s.left = 0 }

// speechSkipGrace is how long after a line starts a skip key is swallowed without cutting it: the exe's
// speech key handler (table 0x721940, handler 0x49de80) compares GetTickCount with the tick stored when the
// line opened (0x7b72b4) and only acts after 0x64 = 100 ms (VERIFIED).
const speechSkipGrace = 0.1

// Active reports whether a line is on screen.
func (s *SpeechBubble) Active() bool { return s.left > 0 }

// Skippable reports whether a skip key would cut the line now (it is on screen for 100 ms or more).
func (s *SpeechBubble) Skippable() bool { return s.left > 0 && s.age >= speechSkipGrace }

// Text returns the speech currently shown ("" when none).
func (s *SpeechBubble) Text() string {
	if s.left <= 0 {
		return ""
	}

	parts := make([]string, 0, len(s.lines))
	for _, l := range s.lines {
		parts = append(parts, l.GetText())
	}

	return strings.Join(parts, " ")
}

// Advance counts the display times down.
func (s *SpeechBubble) Advance(elapsed float64) {
	if s.left > 0 {
		s.left -= elapsed
		s.age += elapsed
	}

	if s.noticeLeft > 0 {
		s.noticeLeft -= elapsed
	}
}

// Render draws the bubble.
func (s *SpeechBubble) Render(target d2interface.Surface) {
	if s.noticeLeft > 0 && s.notice != nil {
		w, _ := s.notice.GetTextMetrics(s.notice.GetText())
		x, y, bw := NoticePlacement(d2display.Get(), w)

		// the notice stands at the top of the screen: leave the interface column the app translated into
		ox, oy := d2display.ColumnOrigin()
		target.PushTranslation(x-ox, y-oy)
		target.DrawRect(bw, speechLineHeight+speechPadding, speechBackground)
		target.Pop()

		s.notice.SetPosition(x-ox+speechPadding, y-oy+speechLineHeight)
		s.notice.Render(target)
	}

	if s.left <= 0 || len(s.lines) == 0 {
		return
	}

	left, top, _, height := SpeechPlacement(len(s.lines))

	target.PushTranslation(left, top)
	target.DrawRect(speechWidth, height, speechBackground)
	target.Pop()

	s.name.SetPosition(left+speechPadding, top+speechPadding+speechLineHeight)
	s.name.Render(target)

	for i, l := range s.lines {
		l.SetPosition(left+speechPadding, top+speechPadding+(i+2)*speechLineHeight)
		l.Render(target)
	}
}
