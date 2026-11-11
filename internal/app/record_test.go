package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// A record of a listener's plays: the songs of the week, and those of all
// time, most played first.
func testRecord() *recordData {
	return &recordData{
		weekShown: true, allShown: true,
		week: []netease.PlayRecord{
			{PlayCount: 12, Track: netease.Track{ID: 1, Name: "雨", Artists: []netease.Artist{{ID: 7, Name: "歌手"}}, Album: netease.Album{ID: 3, Name: "专辑"}, Duration: 215000}},
			{PlayCount: 4, Track: netease.Track{ID: 2, Name: "影", Artists: []netease.Artist{{ID: 7, Name: "歌手"}}, Album: netease.Album{ID: 3, Name: "专辑"}, Duration: 180000}},
		},
		all: []netease.PlayRecord{
			{PlayCount: 80, Track: netease.Track{ID: 3, Name: "光", Artists: []netease.Artist{{ID: 10, Name: "另一个歌手"}}, Album: netease.Album{ID: 4, Name: "另一张专辑"}, Duration: 200000}},
		},
	}
}

// The record's card leads the playlists of the listener's own page, and its
// page lists what was played this week, and of all time, with the number of
// times each song was played.
func TestRecordPage(t *testing.T) {
	ars, pods := followedArtists, subscribedPodcasts
	defer func() { followedArtists, subscribedPodcasts = ars, pods }()
	followedArtists = func() ([]netease.ArtistProfile, error) { return nil, nil }
	subscribedPodcasts = func() ([]netease.Podcast, error) { return nil, nil }

	a := ownPageApp()
	a.records[9] = testRecord()
	a.open(route{page: pageUser, id: 9})
	tt := ui.NewTester(a.view, 1200, 1200)
	tt.Frame()
	if err := tt.Click("听歌排行"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.route.page != pageRecord || a.route.id != 9 {
		t.Fatalf("route = %+v, want the record of user 9", a.route)
	}
	if !tt.HasText("12 次") || !tt.HasText("4 次") {
		t.Fatal("the week's plays are not counted")
	}
	if !tt.HasText("2 首 · 共 16 次播放") {
		t.Fatal("the week's total is not there")
	}
	if err := tt.Click("所有时间"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if a.records[9].tab != recordAll {
		t.Fatalf("tab = %d, want all time", a.records[9].tab)
	}
	if !tt.HasText("80 次") || !tt.HasText("1 首 · 共 80 次播放") {
		t.Fatal("the all time plays are not listed")
	}
}

// A record that could not be read says so, and offers to read it again.
func TestRecordPageError(t *testing.T) {
	a := newTestApp()
	a.records[9] = &recordData{weekShown: true, weekErr: "最近一周的听歌记录加载失败"}
	a.open(route{page: pageRecord, id: 9})
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if !tt.HasText("最近一周的听歌记录加载失败") {
		t.Fatal("the failure is not said")
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

// The record's page opens on what was played this week.
func TestRecordPageStartsOnTheWeek(t *testing.T) {
	a := newTestApp()
	a.records[9] = testRecord()
	a.open(route{page: pageRecord, id: 9})
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if a.records[9].tab != recordWeek {
		t.Fatalf("tab = %d, want the week", a.records[9].tab)
	}
	if !tt.HasText("最近一周") || !tt.HasText("12 次") {
		t.Fatal("the week is not listed")
	}
}

// With NM_FRAMES set, saves both rankings of a record, in the light and the
// dark.
func TestRecordFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	for _, dark := range []bool{false, true} {
		for tab := range 2 {
			a := newTestApp()
			a.records[9] = testRecord()
			a.records[9].tab = tab
			a.open(route{page: pageRecord, id: 9})
			tt := ui.NewTester(a.view, 1200, 800)
			tt.SetDark(dark)
			tt.Advance(time.Second)
			tt.Frame()
			name := fmt.Sprintf("record-%d.png", tab)
			if dark {
				name = fmt.Sprintf("record-%d-dark.png", tab)
			}
			f, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
}
