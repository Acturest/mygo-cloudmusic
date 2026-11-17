package app

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// The surface noise's mix is in the song's menu in Now Playing.
func TestVinylMenuHasMix(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	a.pl.Cue([]netease.Track{{ID: 1, Name: "一", Duration: 200000}})
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if err := tt.Click("更多"); err != nil {
		t.Fatal(err)
	}
	if menu := tt.Menu(); !slices.Contains(menu, "配比") {
		t.Fatalf("the menu shows %q", menu)
	}
	tt.CloseMenu()
}
