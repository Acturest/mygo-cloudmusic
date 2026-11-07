package app

import (
	"testing"

	"netease-music-mygo/internal/player"
)

// Light mode plays the queue on repeat, and gives back the listener's own
// repeat mode with the window.
func TestLiteModeRestoresRepeat(t *testing.T) {
	a := newTestApp()
	a.pl.SetRepeat(player.RepeatOne)
	a.setLite(true)
	if !a.lite || a.pl.Repeat() != player.RepeatAll {
		t.Fatalf("light mode: lite %v, repeat %v, want the list repeating", a.lite, a.pl.Repeat())
	}
	a.setLite(false)
	if a.lite || a.pl.Repeat() != player.RepeatOne {
		t.Fatalf("back: lite %v, repeat %v, want the mode it had", a.lite, a.pl.Repeat())
	}
}
