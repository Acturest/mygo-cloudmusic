package app

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
	"netease-music-mygo/internal/netease"
)

func TestSongMenuCopies(t *testing.T) {
	var copied string
	real := copyText
	copyText = func(s string) error { copied = s; return nil }
	defer func() { copyText = real }()

	a := newTestApp()
	a.settings = defaultSettings()
	tr := netease.Track{ID: 1859245776, Name: "宝船", Artists: []netease.Artist{{Name: "DOUDOU"}, {Name: "小明"}}, Duration: 200000}
	a.pl.Cue([]netease.Track{tr})
	a.pl.SetLyrics(nil)
	a.nowPlaying = true
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Advance(time.Second)
	tt.Frame()

	for _, c := range []struct{ item, want string }{
		{"复制歌名", "宝船"},
		{"复制歌名和歌手", "宝船 - DOUDOU, 小明"},
		{"复制链接", "https://music.163.com/song?id=1859245776"},
	} {
		if err := tt.Click("更多"); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(c.item); err != nil {
			t.Fatalf("%v; the menu shows %q", err, tt.Menu())
		}
		tt.Frame()
		if copied != c.want {
			t.Errorf("%s copied %q, want %q", c.item, copied, c.want)
		}
	}
}

// On the lyrics pane, the dots also copy the lyrics of what plays.
func TestLyricsMenuCopies(t *testing.T) {
	var copied string
	real := copyText
	copyText = func(s string) error { copied = s; return nil }
	defer func() { copyText = real }()

	a := newTestApp()
	a.settings = defaultSettings()
	a.pl.Cue([]netease.Track{{ID: 1, Name: "咲き誇れ", Duration: 200000}})
	a.pl.SetLyrics([]lyrics.Line{
		{At: 0, Text: "今すぐに", Translation: "现在立刻"},
		{At: time.Second, Text: "思うままに", Translation: "遵从内心的声音"},
	})
	a.nowPlaying = true
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()

	if err := tt.Click("更多"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("复制当前歌词"); err != nil {
		t.Fatalf("%v; the menu shows %q", err, tt.Menu())
	}
	tt.Frame()
	if want := "今すぐに\n现在立刻\n思うままに\n遵从内心的声音"; copied != want {
		t.Errorf("copied %q, want %q", copied, want)
	}
}

// A right click on a line copies its words, or them with the translation.
func TestLyricLineRightClickCopies(t *testing.T) {
	var copied string
	real := copyText
	copyText = func(s string) error { copied = s; return nil }
	defer func() { copyText = real }()

	a := newTestApp()
	a.settings = defaultSettings()
	a.pl.Cue([]netease.Track{{ID: 1, Name: "咲き誇れ", Duration: 200000}})
	a.pl.SetLyrics([]lyrics.Line{
		{At: 0, Text: "今すぐに", Translation: "现在立刻"},
		{At: time.Second, Text: "思うままに", Translation: "遵从内心的声音"},
	})
	a.nowPlaying = true
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()

	for _, c := range []struct{ item, want string }{
		{"复制这句歌词", "思うままに"},
		{"复制原文和翻译", "思うままに\n遵从内心的声音"},
	} {
		if err := tt.RightClick("思うままに"); err != nil {
			t.Fatal(err)
		}
		if got, want := tt.Menu(), []string{"复制这句歌词", "复制原文和翻译"}; !slices.Equal(got, want) {
			t.Fatalf("the menu shows %q, want %q", got, want)
		}
		if err := tt.ChooseMenuItem(c.item); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if copied != c.want {
			t.Errorf("%s copied %q, want %q", c.item, copied, c.want)
		}
	}
}

func TestSongMenuOnRightClick(t *testing.T) {
	var copied string
	real := copyText
	copyText = func(s string) error { copied = s; return nil }
	defer func() { copyText = real }()

	a := newTestApp()
	a.settings = defaultSettings()
	a.playlists[42] = &netease.Playlist{ID: 42, Name: "飙升榜", Tracks: []netease.Track{
		{ID: 1, Name: "一张相片", Artists: []netease.Artist{{Name: "林宥嘉"}}, Duration: 200000},
		{ID: 2, Name: "宝船", Artists: []netease.Artist{{Name: "DOUDOU"}}, Duration: 200000},
	}}
	a.route = route{page: pagePlaylist, id: 42}
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()

	if err := tt.RightClick("宝船"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"复制歌名", "复制歌名和歌手", "-", "复制链接"}; !slices.Equal(got, want) {
		t.Fatalf("the menu shows %q, want %q", got, want)
	}
	if err := tt.ChooseMenuItem("复制链接"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if want := "https://music.163.com/song?id=2"; copied != want {
		t.Errorf("copied %q, want %q", copied, want)
	}
	if a.pl.Current() != nil {
		t.Error("the right click played the song")
	}
}
