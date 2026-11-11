package netease

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// PlayRecord is a song on a listener's record, with how often they played it.
type PlayRecord struct {
	PlayCount int
	// Score is how much of the song they listened to, by NetEase's count.
	Score int
	Track Track
}

// recordLine is a song on the record, before it is flattened into a PlayRecord.
type recordLine struct {
	PlayCount int        `json:"playCount"`
	Score     int        `json:"score"`
	Song      recordSong `json:"song"`
}

// recordSong is a song on the record, which comes in either the web's shape
// (artists, album, duration) or the app's (ar, al, dt). It reads both.
type recordSong struct {
	Track
}

func (s *recordSong) UnmarshalJSON(b []byte) error {
	var raw struct {
		Track
		Artists  []Artist `json:"artists"`
		Album    Album    `json:"album"`
		Duration int64    `json:"duration"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	s.Track = raw.Track
	if len(s.Artists) == 0 {
		s.Artists = raw.Artists
	}
	if s.Album.ID == 0 {
		s.Album = raw.Album
	}
	if s.Duration == 0 {
		s.Duration = raw.Duration
	}
	return nil
}

// UserRecord returns the songs a listener played most, this week or of all
// time, the most played first.
func UserRecord(uid int64, week bool) ([]PlayRecord, error) {
	var res struct {
		WeekData []recordLine `json:"weekData"`
		AllData  []recordLine `json:"allData"`
	}
	t := "0"
	if week {
		t = "1"
	}
	q := url.Values{"uid": {strconv.FormatInt(uid, 10)}, "type": {t}}
	// The web client's /user/record; NetEase answers it here, not at
	// /api/user/record (that path is a 404).
	if err := get("/api/v1/play/record", q, &res); err != nil {
		return nil, err
	}
	lines := res.AllData
	if week {
		lines = res.WeekData
	}
	records := make([]PlayRecord, len(lines))
	// They come without privileges, as the daily songs do.
	songs := make([]Track, len(lines))
	for i, l := range lines {
		records[i] = PlayRecord{PlayCount: l.PlayCount, Score: l.Score}
		songs[i] = l.Song.Track
	}
	detailPrivileges(songs)
	for i := range records {
		records[i].Track = songs[i]
	}
	return records, nil
}
