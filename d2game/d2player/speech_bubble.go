package d2player

import (
	"image/color"
	"strings"

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

	notice     *d2ui.Label
	noticeLeft float64
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
}

// Notice displays a one line message near the top of the screen.
func (s *SpeechBubble) Notice(text string, seconds float64) {
	s.notice = s.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	s.notice.SetText(text)
	s.notice.Color[0] = speechSpeaker
	s.noticeLeft = seconds
}

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
	}

	if s.noticeLeft > 0 {
		s.noticeLeft -= elapsed
	}
}

// Render draws the bubble.
func (s *SpeechBubble) Render(target d2interface.Surface) {
	if s.noticeLeft > 0 && s.notice != nil {
		w, _ := s.notice.GetTextMetrics(s.notice.GetText())
		bw := w + 2*speechPadding

		target.PushTranslation((800-bw)/2, noticeTop)
		target.DrawRect(bw, speechLineHeight+speechPadding, speechBackground)
		target.Pop()

		s.notice.SetPosition((800-w)/2, noticeTop+speechLineHeight)
		s.notice.Render(target)
	}

	if s.left <= 0 || len(s.lines) == 0 {
		return
	}

	height := (len(s.lines)+1)*speechLineHeight + 2*speechPadding
	left, top := (800-speechWidth)/2, speechBottom-height

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
