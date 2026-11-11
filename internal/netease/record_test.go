package netease

import (
	"encoding/json"
	"testing"
)

func TestRecordSongReadsBothShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "web",
			body: `{"playCount":5,"score":80,"song":{"id":1,"name":"雨","artists":[{"id":2,"name":"歌手"}],"album":{"id":3,"name":"专辑","picUrl":"p"},"duration":215000}}`,
		},
		{
			name: "app",
			body: `{"playCount":5,"score":80,"song":{"id":1,"name":"雨","ar":[{"id":2,"name":"歌手"}],"al":{"id":3,"name":"专辑","picUrl":"p"},"dt":215000}}`,
		},
	}
	for _, c := range cases {
		var line recordLine
		if err := json.Unmarshal([]byte(c.body), &line); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		tr := line.Song.Track
		if line.PlayCount != 5 || line.Score != 80 {
			t.Errorf("%s: play count %d, score %d", c.name, line.PlayCount, line.Score)
		}
		if tr.ID != 1 || tr.Name != "雨" || tr.ArtistNames() != "歌手" || tr.Album.ID != 3 || tr.Album.Name != "专辑" || tr.Duration != 215000 {
			t.Errorf("%s: track %+v", c.name, tr)
		}
	}
}
