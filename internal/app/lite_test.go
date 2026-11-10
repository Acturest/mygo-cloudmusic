package app

import (
	"testing"

	"netease-music-mygo/internal/netease"
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

// Light mode gives back what the pages loaded, so a session that opened many
// of them comes back to what a window just opened takes.
func TestLiteModeGivesBackThePagesCaches(t *testing.T) {
	a := newTestApp()
	a.playlists[1] = &netease.Playlist{ID: 1}
	a.artists[2] = &artistData{}
	a.searches[route{page: pageSearch}] = &searchData{}
	a.art.get("https://example.com/a.jpg", false)
	a.setLite(true)
	if len(a.playlists) != 0 || len(a.artists) != 0 || len(a.searches) != 0 {
		t.Fatalf("light mode kept the pages: %d playlists, %d artists, %d searches",
			len(a.playlists), len(a.artists), len(a.searches))
	}
	if a.art.size != 0 {
		t.Fatalf("light mode kept %d bytes of covers", a.art.size)
	}
}
