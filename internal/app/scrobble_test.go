package app

import (
	"errors"
	"testing"
	"time"

	"netease-music-mygo/internal/netease"
)

// scrobbleSent is one log the play log sent.
type scrobbleSent struct {
	start    bool
	id       int64
	sourceID int64
	source   string
	played   time.Duration
}

// stubPlayLog makes the play log's sending heard here, and quick: no gap
// between logs, no listen skipped, no backoff.
func stubPlayLog(t *testing.T) (chan scrobbleSent, *func() error) {
	t.Helper()
	sent := make(chan scrobbleSent, 8)
	oldStart, oldEnd := startPlay, playEnd
	oldGap, oldSkip, oldBackoff := scrobbleGap, scrobbleSkip, scrobbleBackoff
	scrobbleGap, scrobbleSkip, scrobbleBackoff = 0, 0, 0
	startPlay = func(id, sourceID int64, source string) error {
		sent <- scrobbleSent{start: true, id: id, sourceID: sourceID, source: source}
		return nil
	}
	refuse := func() error { return nil }
	playEnd = func(id, sourceID int64, source string, played time.Duration) error {
		sent <- scrobbleSent{id: id, sourceID: sourceID, source: source, played: played}
		return refuse()
	}
	t.Cleanup(func() {
		startPlay, playEnd = oldStart, oldEnd
		scrobbleGap, scrobbleSkip, scrobbleBackoff = oldGap, oldSkip, oldBackoff
	})
	return sent, &refuse
}

// waitLogs waits until the play log sent what it had, or failed to.
func waitLogs(t *testing.T, a *app) {
	t.Helper()
	if a.scrobbles == nil {
		return
	}
	a.scrobbles.mu.Lock()
	done := a.scrobbles.done
	a.scrobbles.mu.Unlock()
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the play log did not settle")
	}
}

func expectLog(t *testing.T, sent chan scrobbleSent) scrobbleSent {
	t.Helper()
	select {
	case got := <-sent:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("no log")
		return scrobbleSent{}
	}
}

func expectNoLog(t *testing.T, sent chan scrobbleSent) {
	t.Helper()
	select {
	case got := <-sent:
		t.Fatalf("logged %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

// reportPlay sends a listen to NetEase only when the listener turned
// listening reports on, is signed in, and what played is a song: a podcast
// has no listen count to add to.
func TestReportPlay(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 9}
	sent, _ := stubPlayLog(t)

	tr := netease.Track{ID: 7, Name: "宝船", Duration: 200000}

	a.reportPlay(tr, 150*time.Second)
	expectNoLog(t, sent)

	a.settings.Scrobble = true
	a.reportPlay(tr, 150*time.Second)
	got := expectLog(t, sent)
	if got.start || got.id != 7 || got.played != 150*time.Second || got.source != "list" || got.sourceID != 0 {
		t.Fatalf("logged %+v", got)
	}

	a.reportPlay(netease.Track{ID: 13, ProgramID: 3, Duration: 600000}, 300*time.Second)
	expectNoLog(t, sent)

	a.reportPlay(netease.Track{ID: 14, Duration: 20000}, 20*time.Second)
	expectNoLog(t, sent)

	a.profile = nil
	a.reportPlay(tr, 150*time.Second)
	expectNoLog(t, sent)
}

// A listen is never longer than the song: NetEase reads a report of a song
// that outlasted itself as one nobody made.
func TestReportPlayCapsTheLength(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 9}
	a.settings.Scrobble = true
	sent, _ := stubPlayLog(t)

	a.reportPlay(netease.Track{ID: 7, Duration: 200000}, 300*time.Second)
	waitLogs(t, a)
	if got := expectLog(t, sent); got.played != 200*time.Second {
		t.Fatalf("logged %+v", got)
	}
}

// Some listens go unreported, as a listener's do.
func TestReportPlayLeavesSomeListensOut(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 9}
	a.settings.Scrobble = true
	sent, _ := stubPlayLog(t)
	scrobbleSkip = 1

	a.reportPlay(netease.Track{ID: 7, Duration: 200000}, 150*time.Second)
	waitLogs(t, a)
	expectNoLog(t, sent)
}

// reportStart logs that a song started, then reportPlay what it played:
// NetEase counts a listen better when its two logs come as a listener's do.
func TestReportStartAndPlayAreTwoLogs(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 9}
	a.settings.Scrobble = true
	sent, _ := stubPlayLog(t)

	tr := netease.Track{ID: 7, Duration: 200000}
	a.reportStart(tr)
	first := expectLog(t, sent)
	if !first.start || first.id != 7 || first.played != 0 || first.source != "list" {
		t.Fatalf("started with %+v", first)
	}

	a.reportPlay(tr, 150*time.Second)
	waitLogs(t, a)
	second := expectLog(t, sent)
	if second.start || second.id != 7 || second.played != 150*time.Second {
		t.Fatalf("played %+v", second)
	}

	a.reportStart(netease.Track{ID: 13, ProgramID: 3, Duration: 600000})
	expectNoLog(t, sent)
}

// A failed log is not tried again, and the ones waiting go unsent: knocking
// again on what refused a log is what the client never does.
func TestScrobbleLogBacksOffAfterAFailure(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 9}
	a.settings.Scrobble = true
	sent, refuse := stubPlayLog(t)
	scrobbleBackoff = time.Hour
	*refuse = func() error { return errors.New("refused") }

	tr := netease.Track{ID: 7, Duration: 200000}
	a.reportPlay(tr, 150*time.Second)
	waitLogs(t, a)
	if got := expectLog(t, sent); got.played != 150*time.Second {
		t.Fatalf("logged %+v", got)
	}

	a.reportPlay(tr, 150*time.Second)
	waitLogs(t, a)
	expectNoLog(t, sent)
}
