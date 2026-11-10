package app

import (
	"testing"
	"time"

	"netease-music-mygo/internal/netease"
	"github.com/egoist/mygo/ui"
)

// An instrumental has no lyrics, so the stage shows its own message in a box
// as large as the window. That box must let the pointer through, or the strip
// above it cannot be dragged and its buttons cannot be pressed.
func TestStageOfAnInstrumentalTakesNoPointer(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	tr := netease.Track{ID: 7, Name: "海屿你", Duration: 200000}
	a.pl.Cue([]netease.Track{tr})
	a.nowPlaying, a.npPane = true, paneStage
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Advance(time.Second)
	// The chevron only shows while the pointer is in the window.
	tt.Move(600, 8)
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("关闭"); err != nil {
		t.Fatal(err)
	}
	if a.nowPlaying {
		t.Fatalf("the close button did not take the click")
	}
}
