package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"netease-music-mygo/internal/player"
)

// The zero value reads as the neutral chain, and out of range values are
// brought back into it.
func TestEffectSettingsNormalized(t *testing.T) {
	neutral := neutralEffectSettings()
	if got := (effectSettings{}).normalized(); got != neutral {
		t.Errorf("zero value = %+v, want %+v", got, neutral)
	}
	in := effectSettings{Gains: [10]float64{20, -20, 0, 0, 0, 0, 0, 0, 0, 0}, Highpass: 5000, Lowpass: 100, Width: 3, Space: 2}
	got := in.normalized()
	if got.Gains[0] != 12 || got.Gains[1] != -12 {
		t.Errorf("gains %v", got.Gains)
	}
	if got.Highpass != 2000 || got.Lowpass != 500 {
		t.Errorf("cutoffs %v %v", got.Highpass, got.Lowpass)
	}
	if got.Width != 2 || got.Space != 1 {
		t.Errorf("width %v space %v", got.Width, got.Space)
	}
}

// Applying a preset makes its curve and chain, and finds it again.
func TestAudioPresetApply(t *testing.T) {
	var s effectSettings
	for _, p := range audioEffectPresets {
		s.applyAudioPreset(p)
		if s.Preset != p.id {
			t.Fatalf("%s: preset %q", p.id, s.Preset)
		}
		if s.Gains != p.gains {
			t.Errorf("%s: gains %v", p.id, s.Gains)
		}
		if got := s.effectSettingsForPreset(); got != p.id {
			t.Errorf("%s: found again as %q", p.id, got)
		}
	}
	s.Gains[0] = 5
	if got := s.effectSettingsForPreset(); got != "" {
		t.Errorf("edited curve still matches %q", got)
	}
}

// The chain reaches the player only while its switch is on.
func TestEffectSettingsPlayer(t *testing.T) {
	s := neutralEffectSettings()
	s.applyAudioPreset(audioEffectPresets[1]) // lofi
	if got := s.playerEffects(false); got.Enabled {
		t.Error("disabled chain enabled")
	}
	got := s.playerEffects(true)
	want := player.Effects{Enabled: true, Gains: s.Gains, Highpass: s.Highpass, Lowpass: s.Lowpass,
		Drive: s.Drive, Crush: s.Crush, Wow: s.Wow, Noise: s.Noise, Width: s.Width, Space: s.Space, Punch: s.Punch}
	if got != want {
		t.Errorf("converted %+v, want %+v", got, want)
	}
}

// The effects are kept with the other settings.
func TestEffectSettingsPersist(t *testing.T) {
	s := defaultSettings()
	s.EffectsEnabled = true
	s.Effects.applyAudioPreset(audioEffectPresets[2]) // radio
	s.path = t.TempDir() + "/settings.json"
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "effectsEnabled") {
		t.Errorf("saved %s", b)
	}
	var got settings
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	got.Effects = got.Effects.normalized()
	if got.Effects.Gains != s.Effects.Gains || got.Effects.Preset != "radio" {
		t.Errorf("read back %+v", got.Effects)
	}
}

// The cover style and the record's noise mix are kept with the other
// settings.
func TestVinylSettingsPersist(t *testing.T) {
	s := defaultSettings()
	s.Cover = "vinyl"
	s.VinylHiss = 0.4
	s.VinylRumble = 0.6
	s.VinylCrackle = 0.8
	s.VinylHum = 0.2
	s.VinylCrackleProb = 0.3
	s.path = t.TempDir() + "/settings.json"
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	var got settings
	b, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Cover != "vinyl" || got.VinylHiss != 0.4 || got.VinylRumble != 0.6 || got.VinylCrackle != 0.8 || got.VinylHum != 0.2 || got.VinylCrackleProb != 0.3 {
		t.Errorf("read back cover %q mix %+v", got.Cover, got)
	}
}
