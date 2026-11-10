package app

import (
	"runtime/debug"
	"time"

	"netease-music-mygo/internal/netease"
)

// heapLimit is the soft limit the runtime works to keep the heap under, its
// own and the covers'. The app's use is far below it: the limit is there so
// that a page loading a lot makes the runtime give memory back to Windows as
// it goes, rather than grow to the session's peak and keep it.
const heapLimit = 96 << 20

// gcPercent keeps the heap tight: playing a song decodes it a frame at a
// time, and go-mp3 makes about two megabytes of garbage a second doing it.
// At the default the runtime lets the heap drift to twice what the app can
// reach on a page, which is 20 MB the task manager shows. Holding it to a
// quarter costs the collector a pass a second, which does not show on the
// clock.
const gcPercent = 25

// releaseEvery is how often the pages left behind give their memory back at
// most: pages change on clicks, and giving the heap back pauses a few
// milliseconds while Windows gets it.
const releaseEvery = 10 * time.Second

// releaseMemory gives back what the page left behind held: the covers it
// showed, and the heap the runtime grew to load them. Windows trims a
// working set when it needs the memory, so what the app hands back is what
// the task manager shows it taking.
func (a *app) releaseMemory() {
	if a.art != nil {
		a.art.shrink()
	}
	// Without a window there is no working set to give back, and the tests
	// open pages by the hundred.
	if a.win == nil || time.Since(a.releasedAt) < releaseEvery {
		return
	}
	a.releasedAt = time.Now()
	// The pause is a few milliseconds, and not on this thread.
	go debug.FreeOSMemory()
}

// clearCaches drops what the pages loaded: the playlists, artists, albums,
// users, podcasts and searches they showed, and every cover. Nothing is
// showing them while light mode has the window away, and the page that comes
// back loads its own again, so the heap goes back to what it was before any
// page was opened.
func (a *app) clearCaches() {
	a.playlists = map[int64]*netease.Playlist{}
	a.loadingPl = map[int64]bool{}
	a.plErr = map[int64]string{}
	a.artists = map[int64]*artistData{}
	a.albums = map[int64]*albumData{}
	a.users = map[int64]*userData{}
	a.podcasts = map[int64]*podcastData{}
	a.searches = map[route]*searchData{}
	if a.art != nil {
		a.art.clear()
	}
}

// releaseAll gives back everything the session grew, as light mode does when
// the window goes away: the pages' caches, and the heap they left. It runs
// where releaseMemory would not, without its pause's wait, because the window
// is away and nothing shows the stop.
func (a *app) releaseAll() {
	a.clearCaches()
	a.releasedAt = time.Now()
	debug.FreeOSMemory()
}
