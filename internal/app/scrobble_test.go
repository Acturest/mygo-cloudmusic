package app

import (
	"testing"
	"time"

	"netease-music-mygo/internal/netease"
)

// reportPlay sends a listen to NetEase only when the listener turned
// listening reports on, is signed in, and what played is a song: a podcast
// has no listen count to add to.
func TestReportPlay(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 9}

	reports := make(chan struct {
		id     int64
		played time.Duration
	}, 4)
	old := scrobble
	scrobble = func(id, sourceID int64, source string, played time.Duration) error {
		reports <- struct {
			id     int64
			played time.Duration
		}{id, played}
		return nil
	}
	defer func() { scrobble = old }()

	tr := netease.Track{ID: 7, Name: "宝船", Duration: 200000}

	a.reportPlay(tr, 150*time.Second)
	select {
	case got := <-reports:
		t.Fatalf("reported while the setting is off: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}

	a.settings.Scrobble = true
	a.reportPlay(tr, 150*time.Second)
	select {
	case got := <-reports:
		if got.id != 7 || got.played != 150*time.Second {
			t.Fatalf("reported %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("no report")
	}

	a.reportPlay(netease.Track{ID: 13, ProgramID: 3}, 600*time.Second)
	select {
	case got := <-reports:
		t.Fatalf("reported a podcast: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}

	a.profile = nil
	a.reportPlay(tr, 150*time.Second)
	select {
	case got := <-reports:
		t.Fatalf("reported signed out: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}
