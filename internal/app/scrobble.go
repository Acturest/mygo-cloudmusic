package app

import (
	"math/rand/v2"
	"sync"
	"time"

	"netease-music-mygo/internal/netease"
)

// startPlay and playEnd send the client's two play logs; the tests hear them
// instead of sending them.
var (
	startPlay = netease.StartPlay
	playEnd   = netease.PlayEnd
)

// How the play log keeps out of NetEase's risk control, which cleans out what
// does not look like a listener's own playing. They are variables, as the
// tests set them.
var (
	// scrobbleSkip is the share of listens left unreported: a listener does
	// not report every song they hear.
	scrobbleSkip = 0.15
	// scrobbleGap is the least time between two logs: the client logs as it
	// plays, never in a burst.
	scrobbleGap = 3 * time.Second
	// scrobbleBackoff is how long the log holds back once one failed, rather
	// than trying again at once.
	scrobbleBackoff = 10 * time.Minute
	// scrobbleMinLength is the shortest song reported: a shorter one never
	// played long enough to count.
	scrobbleMinLength = 30 * time.Second
)

// play plays tracks at start, from source, whose id is sourceID: the page
// they were picked on, so that the play log says where the listen came from.
func (a *app) play(tracks []netease.Track, start int, source string, sourceID int64) {
	a.pl.PlayQueueFrom(tracks, start, source, sourceID)
}

// routeSource names the page the listener is on as the play log's sources
// do, with the id of its playlist, album or artist, 0 for a page without one.
func (a *app) routeSource() (string, int64) {
	switch a.route.page {
	case pagePlaylist:
		return "list", a.route.id
	case pageAlbum:
		return "album", a.route.id
	case pageArtist:
		return "artist", a.route.id
	case pageDaily, pageFM:
		return "recommend", 0
	case pageSearch:
		return "search", 0
	}
	return "list", 0
}

// reportStart tells NetEase that a track started playing, so that the play
// log comes as the client's does: what started, then later how it ended.
func (a *app) reportStart(t netease.Track) {
	if !a.scrobbleable(t) {
		return
	}
	source, id := a.pl.Origin()
	a.logScrobble(scrobbleJob{start: true, track: t, source: source, id: id})
}

// reportPlay tells NetEase what played, so the listen counts in the
// listener's record of what they play (听歌排行). The player calls it with a
// track and how much of it was heard, when the track ended or another one
// started. Only songs count, and only while the listener has listening
// reports on and is signed in.
func (a *app) reportPlay(t netease.Track, played time.Duration) {
	if !a.scrobbleable(t) {
		return
	}
	// Some listens go unreported: a report of every song heard is not what
	// a listener's log looks like.
	if rand.Float64() < scrobbleSkip {
		return
	}
	// Never longer than the song: a report of a song longer than it played
	// reads as one nobody made.
	if d := t.Length(); played > d {
		played = d
	}
	source, id := a.pl.Origin()
	a.logScrobble(scrobbleJob{track: t, source: source, id: id, played: played})
}

// scrobbleable says whether t is a song the play log reports: a song, not an
// episode, long enough to count, while listening reports are on and the
// listener is signed in.
func (a *app) scrobbleable(t netease.Track) bool {
	return t.ID != 0 && t.ProgramID == 0 && a.settings.Scrobble && a.profile != nil &&
		t.Length() >= scrobbleMinLength
}

// scrobbleJob is one log to send: that a track started, or that it played
// for played.
type scrobbleJob struct {
	start  bool
	track  netease.Track
	source string
	id     int64
	played time.Duration
}

// scrobbleLog sends the play logs one at a time, on a goroutine of its own:
// NetEase counts them better when they come as a listener's playing does,
// spread out and never at once.
type scrobbleLog struct {
	mu      sync.Mutex
	jobs    []scrobbleJob
	sending bool
	last    time.Time // when the last log went
	pause   time.Time // when sending may go on after a failure
	done    chan struct{} // closed once the worker emptied the queue
}

// logScrobble queues one log, and starts sending if nothing is.
func (a *app) logScrobble(j scrobbleJob) {
	if a.scrobbles == nil {
		a.scrobbles = &scrobbleLog{}
	}
	l := a.scrobbles
	l.mu.Lock()
	l.jobs = append(l.jobs, j)
	if l.sending {
		l.mu.Unlock()
		return
	}
	l.sending = true
	l.done = make(chan struct{})
	done := l.done
	l.mu.Unlock()
	go func() {
		defer close(done)
		a.sendScrobbles()
	}()
}

// sendScrobbles sends what is queued, one log at a time. A log that fails is
// not tried again: the ones waiting go unsent, and the log holds back, rather
// than knocking again on what refused it.
func (a *app) sendScrobbles() {
	l := a.scrobbles
	for {
		l.mu.Lock()
		if len(l.jobs) == 0 || time.Now().Before(l.pause) {
			l.jobs = nil
			l.sending = false
			l.mu.Unlock()
			return
		}
		j := l.jobs[0]
		l.jobs = l.jobs[1:]
		wait := scrobbleGap - time.Since(l.last)
		l.mu.Unlock()
		if wait > 0 {
			// Spread the logs out: a machine that sends them at a fixed
			// beat is not one a listener plays on.
			wait += time.Duration(rand.Float64() * float64(scrobbleGap))
			time.Sleep(wait)
		}
		var err error
		if j.start {
			err = startPlay(j.track.ID, j.id, j.source)
		} else {
			err = playEnd(j.track.ID, j.id, j.source, j.played)
		}
		l.mu.Lock()
		l.last = time.Now()
		if err != nil {
			l.pause = l.last.Add(scrobbleBackoff)
		}
		l.mu.Unlock()
	}
}
