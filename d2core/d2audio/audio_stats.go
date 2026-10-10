package d2audio

import (
	"fmt"
	"sort"
	"strings"
)

// Sound categories of the in-game audio audit (OD2_VERIFY_SOUND scenarios).
const (
	CatMusic    = "music"
	CatAmbience = "ambience"
	CatEvent    = "event"
	CatFootstep = "footstep"
	CatSwing    = "swing"
	CatHit      = "hit"
	CatSkill    = "skill"
	CatMonster  = "monster"
	CatSpeech   = "speech"
	CatUI       = "ui"
	CatOther    = "other"
)

// Category classifies a Sounds.txt handle for the audit counters. The handle
// prefixes are the naming of the rows themselves (a heuristic: the original
// has no such category, so this is for reporting only).
func Category(handle string, music, loop bool) string {
	h := strings.ToLower(handle)

	switch {
	case music:
		return CatMusic
	case strings.Contains(h, "_walk_") || strings.Contains(h, "_run_"):
		return CatFootstep
	case strings.HasPrefix(h, "weapon_"):
		return CatSwing
	case strings.HasPrefix(h, "cursor_") || strings.HasPrefix(h, "button_") || strings.HasPrefix(h, "menu_"):
		return CatUI
	case strings.Contains(h, "_hit") || strings.HasPrefix(h, "hit_") || strings.Contains(h, "impact"):
		return CatHit
	case strings.Contains(h, "_cast") || strings.Contains(h, "missile") || strings.Contains(h, "_bolt") ||
		strings.Contains(h, "firebolt"):
		return CatSkill
	case strings.Contains(h, "greeting") || strings.Contains(h, "_bark") || strings.Contains(h, "_speech"):
		return CatSpeech
	case strings.HasPrefix(h, "scene_") || (loop && strings.HasPrefix(h, "ambient")):
		return CatAmbience
	case strings.Contains(h, "_attack") || strings.Contains(h, "_death") || strings.Contains(h, "_gethit") ||
		strings.Contains(h, "_taunt"):
		return CatMonster
	case strings.HasPrefix(h, "event_") || strings.HasPrefix(h, "amb"):
		return CatEvent
	}

	return CatOther
}

// AudioStats counts the sounds that reached a player (a started channel) per
// category, and the loudest volume (0..1, after master volume, ducking and
// distance) each category was ever given. Collected in muted runs too.
type AudioStats struct {
	Started map[string]int
	Peak    map[string]float64
	// Flow is the bytes the audio device pulled per category and FlowPeak the
	// loudest sample (0..32767) among them; both stay zero in muted runs.
	Flow     map[string]int64
	FlowPeak map[string]int
	// ProviderPlays counts every effect the provider started, DirectPlays those
	// that bypassed the voice bank (UI clicks, menu sounds).
	ProviderPlays, DirectPlays int
	// Music and Sfx are the master volumes last seen.
	Music, Sfx float64
}

// String is the AUDIOSTAT summary line body.
func (a AudioStats) String() string {
	cats := make([]string, 0, len(a.Started))
	for c := range a.Started {
		cats = append(cats, c)
	}

	sort.Strings(cats)

	parts := make([]string, 0, len(cats))
	for _, c := range cats {
		parts = append(parts, fmt.Sprintf("%s=%d(vol %.3f flow %dB amp %d)", c, a.Started[c], a.Peak[c], a.Flow[c], a.FlowPeak[c]))
	}

	return fmt.Sprintf("music_master=%.2f sfx_master=%.2f provider_plays=%d direct_plays=%d %s", a.Music, a.Sfx,
		a.ProviderPlays, a.DirectPlays, strings.Join(parts, " "))
}

// statPlayer wraps a player to count starts and record the peak volume.
type statPlayer struct {
	inner interface {
		Play()
		Stop()
		SetPan(pan float64)
		SetVolume(volume float64)
		IsPlaying() bool
	}
	e   *SoundEngine
	cat string
}

func (p *statPlayer) Play() {
	p.e.audit.Started[p.cat]++
	p.e.live[p] = struct{}{}
	p.inner.Play()
}

// flow reads the device counters of the wrapped player, if it keeps any.
func (p *statPlayer) flow() (int64, int) {
	if f, ok := p.inner.(interface{ Flow() (int64, int) }); ok {
		return f.Flow()
	}

	return 0, 0
}

func (p *statPlayer) retire() {
	if _, ok := p.e.live[p]; !ok {
		return
	}

	delete(p.e.live, p)
	p.e.addFlow(p.cat, p)
}

func (p *statPlayer) Stop()               { p.inner.Stop(); p.retire() }
func (p *statPlayer) SetPan(v float64)    { p.inner.SetPan(v) }
func (p *statPlayer) IsPlaying() bool     { return p.inner.IsPlaying() }
func (p *statPlayer) SetVolume(v float64) { p.e.notePeak(p.cat, v); p.inner.SetVolume(v) }

func (s *SoundEngine) notePeak(cat string, v float64) {
	if v > s.audit.Peak[cat] {
		s.audit.Peak[cat] = v
	}
}

func (s *SoundEngine) addFlow(cat string, p *statPlayer) {
	b, pk := p.flow()
	s.audit.Flow[cat] += b

	if pk > s.audit.FlowPeak[cat] {
		s.audit.FlowPeak[cat] = pk
	}
}

// AudioStats returns a copy of the audit counters, including the sounds still playing.
func (s *SoundEngine) AudioStats() AudioStats {
	a := AudioStats{Started: map[string]int{}, Peak: map[string]float64{}, Flow: map[string]int64{},
		FlowPeak: map[string]int{}, Music: s.bgmVol, Sfx: s.sfxVol}

	for k, v := range s.audit.Started {
		a.Started[k] = v
	}

	for k, v := range s.audit.Peak {
		a.Peak[k] = v
	}

	for k, v := range s.audit.Flow {
		a.Flow[k] = v
	}

	for k, v := range s.audit.FlowPeak {
		a.FlowPeak[k] = v
	}

	for p := range s.live {
		b, pk := p.flow()
		a.Flow[p.cat] += b

		if pk > a.FlowPeak[p.cat] {
			a.FlowPeak[p.cat] = pk
		}
	}

	if pp, ok := s.provider.(interface{ Plays() int }); ok {
		a.ProviderPlays = pp.Plays()
		banked := 0

		for _, n := range a.Started {
			banked += n
		}

		if a.DirectPlays = a.ProviderPlays - banked; a.DirectPlays < 0 {
			a.DirectPlays = 0
		}
	}

	return a
}
