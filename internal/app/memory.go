package app

import (
	"runtime/debug"
	"time"
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
