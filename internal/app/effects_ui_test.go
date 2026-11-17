package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The presets and the two custom slots apply their curves.
func TestEffectsPanelChips(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	tt := ui.NewTester(a.effectsView, 880, 720)
	tt.Frame()

	if err := tt.Click("Lo-Fi"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := a.settings.Effects.Preset; got != "lofi" {
		t.Errorf("after Lo-Fi: %q", got)
	}

	if err := tt.Click("自定义 1"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := a.settings.Effects.Preset; got != "custom1" {
		t.Errorf("after custom 1: %q", got)
	}
	if len(a.settings.EffectSlots) < 1 || a.settings.EffectSlots[0].Gains != a.settings.Effects.Gains {
		t.Errorf("custom 1 kept %+v", a.settings.EffectSlots)
	}
}
