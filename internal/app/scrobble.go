package app

import (
	"time"

	"netease-music-mygo/internal/netease"
)

// scrobble reports a play to NetEase; the tests hear the reports instead of
// sending them.
var scrobble = netease.Scrobble

// reportPlay tells NetEase what played, so the listen counts in the
// listener's record of what they play (听歌排行). The player calls it with a
// track and how much of it was heard, when the track ended or another one
// started. Only songs count, and only while the listener has listening
// reports on and is signed in.
func (a *app) reportPlay(t netease.Track, played time.Duration) {
	if t.ID == 0 || t.ProgramID != 0 || !a.settings.Scrobble || a.profile == nil {
		return
	}
	// The report waits on the network, which nothing on screen waits for:
	// it goes on its own, and whatever comes of it is not the listener's
	// business.
	go func() { _ = scrobble(t.ID, 0, "list", played) }()
}
