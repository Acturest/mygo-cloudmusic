package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// userRecord reads a listener's record from NetEase; tests replace it.
var userRecord = netease.UserRecord

// The two rankings of a record: the songs played this week, and the songs
// played of all time.
const (
	recordWeek = iota
	recordAll
)

// recordData is a listener's record as the app has it: what they played this
// week and of all time, most played first.
type recordData struct {
	week, all []netease.PlayRecord
	loading   int // how many of the two are on their way
	tab       int // which of them the page lists: recordWeek or recordAll

	// Shown records whether a ranking has loaded once, which tells a list
	// that is legitimately empty from one still on its way.
	weekShown, allShown bool
	weekErr, allErr     string
}

// ranking is the ranking chosen by the page, with the reason it is missing
// if it failed.
func (d *recordData) ranking() (recs []netease.PlayRecord, err string) {
	if d.tab == recordAll {
		return d.all, d.allErr
	}
	return d.week, d.weekErr
}

// loadRecord asks for both rankings of a listener's record at once: they are
// separate requests, and the page can show either as soon as it comes.
func (a *app) loadRecord(uid int64) {
	d := a.records[uid]
	if d == nil {
		d = &recordData{}
		a.records[uid] = d
	}
	if d.loading > 0 {
		return
	}
	d.loading, d.weekErr, d.allErr = 2, "", ""
	go func() {
		recs, err := userRecord(uid, true)
		a.post(func() {
			d.loading--
			if err != nil {
				d.weekErr = "最近一周的听歌记录加载失败"
				return
			}
			d.week, d.weekShown = recs, true
		})
	}()
	go func() {
		recs, err := userRecord(uid, false)
		a.post(func() {
			d.loading--
			if err != nil {
				d.allErr = "所有时间的听歌记录加载失败"
				return
			}
			d.all, d.allShown = recs, true
		})
	}()
}

// recordTracks is a ranking's songs, in the order they are played.
func recordTracks(recs []netease.PlayRecord) []netease.Track {
	tracks := make([]netease.Track, len(recs))
	for i := range recs {
		tracks[i] = recs[i].Track
	}
	return tracks
}

// recordPage lists the songs of a listener's record: the ones they played
// this week, or those of all time, each with its play count.
func (a *app) recordPage(c *ui.Context) {
	id := a.route.id
	d := a.records[id]
	if d == nil {
		return
	}
	if d.loading > 0 && !d.weekShown && !d.allShown {
		s := a.pageShimmer(c)
		a.trackSkeleton(c, s, true, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		return
	}
	recs, err := d.ranking()
	a.recordRows(c, recs, func(c *ui.Context) { a.recordHeader(c, d) }, func(c *ui.Context) {
		if len(recs) == 0 {
			a.recordEmpty(c, d, err)
		}
	})
}

// recordCard is the record's card among the listener's playlists: one in the
// accent color, as NetEase has no cover for a record.
func (a *app) recordCard(c *ui.Context, uid int64) {
	p := a.pal
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).Justify(ui.Start).FocusRing(false).Label("听歌排行")
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(10)
		pose := ui.Pose{}
		if hovered {
			pose = ui.Pose{Y: -3}
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.AnimatePose(pose, ui.Snappy)
		art.Children(func() {
			a.recordCover(c, 10, 44)
		})
		ui.Text(c, "听歌排行").FontSize(13).FontWeight(500).MaxLines(2).LineHeight(1.35)
		ui.Text(c, "最近一周 · 所有时间").FontSize(12).TextColor(p.muted).SingleLine().Margin(-6, 0, 0, 0)
	})
	if card.Clicked() {
		a.open(route{page: pageRecord, id: uid})
	}
}

// recordCover paints a record's cover: the accent color in a gradient with a
// chart on it.
func (a *app) recordCover(c *ui.Context, radius, icon float32) {
	dark := ui.RGB(0, 0, 0)
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Radius(radius).Center().
		LinearGradient(ui.LinearGradient{Angle: 135, Oklab: true, From: a.pal.accent.Mix(dark, 0.25), To: a.pal.accent.Mix(dark, 0.72)}).
		Children(func() {
			ui.Icon(c, iconCharts).Size(icon, icon).TextColor(white.Alpha(0.9))
		})
}

// recordHeader is the top of a record: its cover, the ranking chosen, how
// many songs it holds and how often they were played.
func (a *app) recordHeader(c *ui.Context, d *recordData) {
	p := a.pal
	recs, _ := d.ranking()
	tracks := recordTracks(recs)
	total := 0
	for _, rec := range recs {
		total += rec.PlayCount
	}
	ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 26, 0).Children(func() {
		ui.Box(c).Size(212, 212).Radius(14).Shadow(0, 18, 40, -10, p.cardShade).Children(func() {
			a.recordCover(c, 14, 88)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, "听歌排行").FontSize(12).FontWeight(600).TextColor(p.accent)
			title := "最近一周"
			if d.tab == recordAll {
				title = "所有时间"
			}
			ui.Text(c, title).FontSize(30).FontWeight(800).LetterSpacing(-0.6).MaxLines(2).LineHeight(1.2)
			ui.Text(c, fmt.Sprintf("%d 首 · 共 %s 次播放", len(recs), plays(float64(total)))).FontSize(13).TextColor(p.muted)
			ui.Row(c).Children(func() {
				segmented(c.Key("tab"), &d.tab, "最近一周", "所有时间")
			})
			ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				if pillButton(c.Key("play"), iconPlay, "播放", p.accent, p.onAccent).Clicked() {
					if a.pl.Shuffle() {
						a.pl.ToggleShuffle()
					}
					a.pl.PlayQueue(tracks, firstPlayable(tracks))
				}
				if pillButton(c.Key("shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
					if !a.pl.Shuffle() {
						a.pl.ToggleShuffle()
					}
					a.pl.PlayQueue(tracks, firstPlayable(tracks))
				}
			})
		})
	})
}

// recordEmpty says a ranking has nothing in it, or why it could not be read,
// and offers to read it again.
func (a *app) recordEmpty(c *ui.Context, d *recordData, err string) {
	msg := "最近一周还没有听歌记录"
	if d.tab == recordAll {
		msg = "还没有听歌记录"
	}
	if err != "" {
		msg = err
	}
	ui.Column(c).Padding(40, 0).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, msg).FontSize(13).TextColor(a.pal.muted)
		if err != "" && ui.Button(c.Key("retry"), "重试").Clicked() {
			a.loadRecord(a.route.id)
		}
	})
}

// recordRows lists the songs of a record under its header, which scrolls with
// them, each with the number of times it was played. A click plays the
// ranking from the song clicked.
func (a *app) recordRows(c *ui.Context, recs []netease.PlayRecord, header func(c *ui.Context), footer ...func(c *ui.Context)) {
	p := a.pal
	tracks := recordTracks(recs)
	cur := a.pl.Current()
	w, _ := c.Size()
	wide := w-sidebarWidth > 820
	top, bottom := a.insets(c)
	a.underTop, a.underBottom = true, true
	rows := len(recs) + 1 + len(footer)
	ui.List(c.Key(a.route), &a.trackList, rows, func(i int) {
		if i == 0 {
			ui.Column(c).Children(func() {
				a.startMark(c)
				header(c)
			})
			return
		}
		if i == rows-1 {
			defer a.endMark(c)
		}
		if i > len(recs) {
			footer[i-len(recs)-1](c)
			return
		}
		i--
		rec := &recs[i]
		tr := &rec.Track
		playing := cur != nil && cur.ID == tr.ID
		height, pad, gap, cover := a.listRow()
		row := ui.ButtonBase(c.Key(tr.ID)).Height(height).PaddingX(pad).Gap(gap).Radius(10).AlignItems(ui.Center).FocusRing(false).Label(tr.Name)
		hovered := row.Hovered()
		row.Background(row.AnimateColor("bg", a.rowBackground(i, playing, hovered), ui.Animation{Duration: 100 * time.Millisecond}))
		if !tr.Playable() {
			row.Opacity(0.42)
		}
		row.Children(func() {
			ui.Box(c).Width(26).Center().Children(func() {
				switch {
				case playing:
					a.nowPlayingBars(c)
				case hovered && tr.Playable():
					ui.Icon(c, iconPlay).Size(13, 13).TextColor(p.text)
				default:
					ui.Textf(c, "%d", i+1).FontSize(13).FontFeatures("tnum").TextColor(p.faint)
				}
			})
			if cover {
				a.cover(c, tr.Album.PicURL, 96, 38, 6)
			}
			ui.Column(c).Grow(3).Basis(0).MinWidth(0).Gap(3).Children(func() {
				name := p.text
				if playing {
					name = p.accent
				}
				spans := []ui.Span{{Text: tr.Name, Weight: 500, Color: name}}
				if len(tr.Alias) > 0 {
					spans = append(spans, ui.Span{Text: "（" + tr.Alias[0] + "）", Color: p.faint})
				}
				ui.RichText(c, spans...).FontSize(14).SingleLine()
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					if !tr.Playable() {
						ui.Text(c, "VIP").FontSize(9).FontWeight(800).TextColor(p.accent).Padding(0, 4).Radius(3).Border(1, p.accent.Alpha(0.6))
					}
					a.artistLinks(c, tr, p.muted, p.text).FontSize(12.5).SingleLine().Shrink(1).MinWidth(0)
					if tr.Reason != "" {
						ui.Text(c, tr.Reason).FontSize(10.5).FontWeight(600).TextColor(p.accent).Padding(1, 6).Radius(999).Background(p.accent.Alpha(0.1)).SingleLine()
					}
				})
			})
			if wide {
				album := ui.RichText(c).Grow(2).Basis(0).MinWidth(0).FontSize(12.5).TextColor(p.muted).SingleLine()
				album.Children(func() {
					if tr.Album.ID == 0 {
						ui.Text(c, tr.Album.Name)
					} else {
						a.link(c.Key("album"), tr.Album.Name, p.muted, p.text, route{page: pageAlbum, id: tr.Album.ID})
					}
				})
			}
			ui.Textf(c, "%d 次", rec.PlayCount).FontSize(12.5).FontFeatures("tnum").TextColor(p.muted).Width(62).TextAlign(ui.End)
			ui.Box(c).Size(32, 32).Center().Children(func() {
				if a.liked[tr.ID] || hovered {
					a.likeButton(c.Key("like"), tr, 15, p.muted)
				}
			})
			ui.Text(c, clock(tr.Length())).FontSize(12.5).FontFeatures("tnum").TextColor(p.faint).Width(44).TextAlign(ui.End)
		})
		if row.Clicked() && tr.Playable() {
			a.pl.PlayQueue(tracks, i)
		}
	}).Grow(1).Padding(top+pageGap, pagePad-12, bottom+32, pagePad-12).ScrollbarInsets(top, 0, bottom, 0)
}
