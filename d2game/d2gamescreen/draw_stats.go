package d2gamescreen

import (
	"os"
	"time"
)

const drawStatsInterval = 2 * time.Second

// logDrawStats writes a DRAWSTATS line (what the last frame drew) every two seconds while OD2_DRAWSTATS is
// set, plus the first frame of every level, so screenshot scenarios can assert the scene content.
func (v *Game) logDrawStats() {
	if os.Getenv("OD2_DRAWSTATS") == "" || v.localPlayer == nil {
		return
	}

	id := v.currentLevel()
	now := time.Now()

	if id == v.statsLevel && now.Sub(v.statsAt) < drawStatsInterval {
		return
	}

	st := v.mapRenderer.DrawStats()
	if st.Floors == 0 && st.Walls == 0 && st.Entities == 0 {
		return // nothing drawn yet (level still loading)
	}

	v.statsLevel, v.statsAt = id, now
	v.Infof("DRAWSTATS level=%d %s", id, st)
}
