package app

import (
	"testing"

	"netease-music-mygo/internal/netease"
)

// The app keeps what the pages just left behind loaded, not every page a
// session went through: a long session would hold every playlist's songs.
func TestPagesKept(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	for i := int64(1); i <= pagesKept+3; i++ {
		a.playlists[i] = &netease.Playlist{ID: i}
	}
	for i := int64(1); i <= pagesKept+3; i++ {
		a.open(route{page: pagePlaylist, id: i})
	}
	if len(a.playlists) != pagesKept {
		t.Fatalf("%d playlists kept, want %d", len(a.playlists), pagesKept)
	}
	for i := int64(1); i <= 3; i++ {
		if a.playlists[i] != nil {
			t.Errorf("playlist %d kept, though it is the oldest", i)
		}
	}
	if a.playlists[pagesKept+3] == nil {
		t.Errorf("playlist %d dropped, though its page is the one open", pagesKept+3)
	}

	// A page whose load is on its way keeps its place: its songs would
	// land with no page to belong to.
	b := newTestApp()
	b.settings = defaultSettings()
	for i := int64(1); i <= pagesKept+1; i++ {
		b.playlists[i] = &netease.Playlist{ID: i}
	}
	b.loadingPl[1] = true
	for i := int64(1); i <= pagesKept+1; i++ {
		b.open(route{page: pagePlaylist, id: i})
	}
	if b.playlists[1] == nil {
		t.Fatalf("playlist 1 dropped while its load was on its way")
	}

	// Once it lands, the pages before it go as before.
	delete(b.loadingPl, 1)
	b.playlists[pagesKept+2] = &netease.Playlist{ID: pagesKept + 2}
	b.open(route{page: pagePlaylist, id: pagesKept + 2})
	if len(b.playlists) != pagesKept {
		t.Fatalf("%d playlists kept once the load landed, want %d", len(b.playlists), pagesKept)
	}
	if b.playlists[1] != nil {
		t.Errorf("playlist 1 kept once its load landed")
	}
}
