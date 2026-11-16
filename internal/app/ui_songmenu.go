package app

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
	"netease-music-mygo/internal/netease"
)

// copyText puts text on the clipboard; tests replace it.
var copyText = func(text string) error { return mygo.Clipboard.Write(transfer.TextData(text)) }

// songMenu is three dots that open the menu of tr, with extra items, if
// any, after the ones that copy it.
func (a *app) songMenu(c *ui.Context, tr *netease.Track, size float32, color ui.Color, extra ...func(m *ui.Menu)) ui.Element {
	b := iconButton(c, iconMore, size, color, "更多")
	b.Menu(a.songItems(tr, 0, extra...))
	return b
}

// songItems builds the menu of what can be done with tr, behind its dots
// or a right click on its row: copying its title, its title with its
// artists, or the link to its page, opening its artists' or album's, and,
// signed in, adding it to a playlist or taking it out of pid, the playlist
// it is listed in, if set.
func (a *app) songItems(tr *netease.Track, pid int64, extra ...func(m *ui.Menu)) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if tr.ProgramID != 0 {
			a.programItems(m, tr)
			return
		}
		if m.Item("复制歌名").Chosen() {
			copyText(tr.Name)
		}
		if m.Item("复制歌名和歌手").Chosen() {
			copyText(tr.Name + " - " + tr.ArtistNames())
		}
		m.Separator()
		if m.Item("复制链接").Chosen() {
			copyText(tr.URL())
		}
		for _, add := range extra {
			add(m)
		}
		// Its artists' and album's pages, those NetEase has.
		linked := tr.Album.ID != 0
		for _, ar := range tr.Artists {
			linked = linked || ar.ID != 0
		}
		if linked {
			m.Separator()
		}
		for _, ar := range tr.Artists {
			if ar.ID != 0 && m.Item("查看歌手："+ar.Name).Chosen() {
				a.openArtist(ar.ID)
			}
		}
		if tr.Album.ID != 0 && m.Item("查看专辑："+tr.Album.Name).Chosen() {
			a.openPage(route{page: pageAlbum, id: tr.Album.ID})
		}
		a.playlistItems(m, tr, pid)
	}
}

// lyricsItem adds the lyrics of what plays to a song's menu, on the lyrics
// pane: copying all of them, with their translations.
func (a *app) lyricsItem(m *ui.Menu) {
	if a.npPane != paneLyrics {
		return
	}
	lines, loaded := a.pl.Lyrics()
	if !loaded || len(lines) == 0 {
		return
	}
	if m.Item("复制当前歌词").Chosen() {
		copyText(lyricsText(lines))
	}
}

// lyricsText is lines as text: each line, and its translation under it if
// it has one.
func lyricsText(lines []lyrics.Line) string {
	var b strings.Builder
	for _, l := range lines {
		if l.Text == "" {
			continue
		}
		b.WriteString(l.Text)
		b.WriteByte('\n')
		if l.Translation != "" {
			b.WriteString(l.Translation)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// lyricLineText is one line as text: its words, and its translation under
// them if it has one.
func lyricLineText(l lyrics.Line) string {
	if l.Translation != "" {
		return l.Text + "\n" + l.Translation
	}
	return l.Text
}

// programItems builds the menu of an episode: copying its title or the
// link to its page, or opening its podcast's.
func (a *app) programItems(m *ui.Menu, tr *netease.Track) {
	if m.Item("复制节目名").Chosen() {
		copyText(tr.Name)
	}
	if m.Item("复制链接").Chosen() {
		copyText(tr.URL())
	}
	m.Separator()
	if m.Item("查看播客").Chosen() {
		a.openPage(route{page: pagePodcast, id: tr.PodcastID})
	}
}
