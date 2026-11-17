package app

import (
	"math"

	"netease-music-mygo/internal/player"
)

// effectSettings is the audio effect chain's settings, as the effects
// window and settings.json keep them: Folia's ten bands and its effect
// chain. Its zero value means unset, read as the neutral chain.
type effectSettings struct {
	Gains    [10]float64 `json:"gains"`
	Highpass float64     `json:"highpass,omitempty"`
	Lowpass  float64     `json:"lowpass,omitempty"`
	Drive    float64     `json:"drive,omitempty"`
	Crush    float64     `json:"crush,omitempty"`
	Wow      float64     `json:"wow,omitempty"`
	Noise    float64     `json:"noise,omitempty"`
	Width    float64     `json:"width,omitempty"`
	Space    float64     `json:"space,omitempty"`
	Punch    float64     `json:"punch,omitempty"`
	Preset   string      `json:"preset,omitempty"`
}

// neutralEffectSettings is the chain doing nothing: flat bands, the
// cutoffs at their ends, stereo as it came.
func neutralEffectSettings() effectSettings {
	return effectSettings{Highpass: 20, Lowpass: 20000, Width: 1, Preset: "flat"}
}

// audioEffectPreset is one of Folia's sound presets: an equalizer curve and
// a whole effect chain state.
type audioEffectPreset struct {
	id      string
	label   string
	gains   [10]float64
	effects effectSettings
}

// audioEffectPresets are Folia's built-in presets, in its order.
var audioEffectPresets = []audioEffectPreset{
	{"flat", "水平", [10]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, effectSettings{Highpass: 20, Lowpass: 20000, Width: 1}},
	{"lofi", "Lo-Fi", [10]float64{-4, -2, 0, 2, 3, 2, 0, -3, -7, -10}, effectSettings{Highpass: 90, Lowpass: 6000, Drive: 0.28, Crush: 0.45, Wow: 0.32, Noise: 0.22, Width: 0.7, Space: 0.16, Punch: 0.4}},
	{"radio", "收音机", [10]float64{-12, -8, -3, 2, 4, 4, 2, -3, -8, -12}, effectSettings{Highpass: 420, Lowpass: 3800, Drive: 0.42, Crush: 0.3, Wow: 0.1, Noise: 0.2, Width: 0.12, Punch: 0.6}},
	{"hall", "大厅", [10]float64{1, 1, 0, 0, 0, 0, 1, 1, 1, 0}, effectSettings{Highpass: 20, Lowpass: 16000, Width: 1.3, Space: 0.55, Punch: 0.15}},
	{"vocal", "人声", [10]float64{-2, -1, 0, 2, 4, 4, 3, 1, 0, -1}, effectSettings{Highpass: 110, Lowpass: 20000, Width: 0.9, Space: 0.08, Punch: 0.35}},
	{"bass", "低音增强", [10]float64{6, 5, 3, 1, 0, -1, -1, 0, 1, 2}, effectSettings{Highpass: 20, Lowpass: 20000, Drive: 0.1, Width: 1.15, Punch: 0.45}},
}

// applyAudioPreset makes the settings the preset's curve and chain.
func (s *effectSettings) applyAudioPreset(p audioEffectPreset) {
	e := p.effects
	e.Gains = p.gains
	e.Preset = p.id
	*s = e.normalized()
}

// effectSettingsForPreset finds the preset whose curve and chain are
// these, or empty.
func (s effectSettings) effectSettingsForPreset() string {
	for _, p := range audioEffectPresets {
		e := p.effects
		e.Gains = p.gains
		if s.equal(e.normalized()) {
			return p.id
		}
	}
	return ""
}

func (s effectSettings) equal(o effectSettings) bool {
	if s.Gains != o.Gains {
		return false
	}
	return s.Highpass == o.Highpass && s.Lowpass == o.Lowpass && s.Drive == o.Drive &&
		s.Crush == o.Crush && s.Wow == o.Wow && s.Noise == o.Noise &&
		s.Width == o.Width && s.Space == o.Space && s.Punch == o.Punch
}

// normalized keeps the chain in range, and reads the zero value as neutral.
func (s effectSettings) normalized() effectSettings {
	if s == (effectSettings{}) {
		return neutralEffectSettings()
	}
	for i := range s.Gains {
		s.Gains[i] = clampFloat(s.Gains[i], -12, 12)
	}
	if s.Highpass <= 0 {
		s.Highpass = 20
	}
	if s.Lowpass <= 0 {
		s.Lowpass = 20000
	}
	s.Highpass = clampFloat(s.Highpass, 20, 2000)
	s.Lowpass = clampFloat(s.Lowpass, 500, 20000)
	s.Drive = clampFloat(s.Drive, 0, 1)
	s.Crush = clampFloat(s.Crush, 0, 1)
	s.Wow = clampFloat(s.Wow, 0, 1)
	s.Noise = clampFloat(s.Noise, 0, 1)
	s.Width = clampFloat(s.Width, 0, 2)
	s.Space = clampFloat(s.Space, 0, 1)
	s.Punch = clampFloat(s.Punch, 0, 1)
	return s
}

func clampFloat(v, min, max float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return min
	}
	return math.Min(max, math.Max(min, v))
}

// playerEffects converts the settings to what the player applies.
func (s effectSettings) playerEffects(enabled bool) player.Effects {
	return player.Effects{
		Enabled:  enabled,
		Gains:    s.Gains,
		Highpass: s.Highpass,
		Lowpass:  s.Lowpass,
		Drive:    s.Drive,
		Crush:    s.Crush,
		Wow:      s.Wow,
		Noise:    s.Noise,
		Width:    s.Width,
		Space:    s.Space,
		Punch:    s.Punch,
	}
}

// effectsPlayerSettings is what the player is given now.
func (a *app) applyEffects() {
	if a.pl != nil {
		a.pl.SetEffects(a.settings.Effects.playerEffects(a.settings.EffectsEnabled))
	}
}

// effectSliderPositions are the log positions the cutoffs' sliders bind to,
// 0 to 1000, kept while the effects window is open.
func effectLogPosition(hz, min, max float64) float64 {
	hz = clampFloat(hz, min, max)
	return (math.Log(hz) - math.Log(min)) / (math.Log(max) - math.Log(min)) * 1000
}

func effectLogValue(pos, min, max float64) float64 {
	pos = clampFloat(pos, 0, 1000)
	return math.Exp(math.Log(min) + pos/1000*(math.Log(max)-math.Log(min)))
}
