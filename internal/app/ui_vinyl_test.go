package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The record turns and the arm comes down while playing, and the arm
// lifts off again once paused.
func TestVinylTurnsAndArms(t *testing.T) {
	a := newTestApp()
	playing := true
	var angle, arm float32
	tt := ui.NewTester(func(c *ui.Context) {
		angle, arm = a.vinylStep(c, playing)
	}, 100, 100)
	tt.Frame()
	tt.Advance(700 * time.Millisecond)
	if angle == 0 {
		t.Error("the record did not turn")
	}
	if arm < 0.5 {
		t.Errorf("the arm is at %v after playing", arm)
	}

	playing = false
	tt.Frame()
	tt.Advance(2 * time.Second)
	if arm > 0.1 {
		t.Errorf("the arm stayed at %v after pausing", arm)
	}
}

// The record's surface noise is off by default and comes on with the
// record cover; the listener can turn it off again.
func TestVinylCoverTurnsNoiseOn(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	if a.settings.VinylOn {
		t.Fatal("surface noise on by default")
	}
	a.setCover("vinyl")
	if a.settings.Cover != "vinyl" || !a.settings.VinylOn {
		t.Errorf("after the record cover: %q, noise %v", a.settings.Cover, a.settings.VinylOn)
	}
	a.setCover("")
	if a.settings.Cover != "" || a.settings.VinylOn {
		t.Errorf("back to the plain cover: %q, noise %v", a.settings.Cover, a.settings.VinylOn)
	}
	a.setCover("vinyl")
	if !a.settings.VinylOn {
		t.Error("the record cover did not turn the noise back on")
	}
}
