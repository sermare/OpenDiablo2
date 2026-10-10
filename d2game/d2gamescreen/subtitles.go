package d2gamescreen

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// subtitleLine formats what an NPC says as one log line, for players who
// cannot hear the voice (or want a transcript): the speaker, the message id
// the quest system sent and the text of its string table entry.
func subtitleLine(speaker string, msg int, text string) string {
	text = strings.Join(strings.Fields(text), " ") // the entries carry line breaks

	if text == "" {
		return fmt.Sprintf("SUBTITLE %s [message %d]: (no text)", speaker, msg)
	}

	return fmt.Sprintf("SUBTITLE %s [message %d]: %s", speaker, msg, text)
}

// logSubtitle writes the subtitle line when the speech subtitle log option
// (Options -> Accessibility, or OD2_SUBTITLE_LOG=1) is on.
func (v *Game) logSubtitle(speaker string, msg int, text string) {
	if !d2player.SubtitleLogOn() {
		return
	}

	v.Info(subtitleLine(speaker, msg, text))
}
