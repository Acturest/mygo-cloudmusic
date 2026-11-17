package app

import (
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/player"
)

// vinylMix is the surface noise's mix as the player takes it.
func (a *app) vinylMix() player.VinylMix {
	return player.VinylMix{
		Hiss:        a.settings.VinylHiss,
		Rumble:      a.settings.VinylRumble,
		Crackle:     a.settings.VinylCrackle,
		Hum:         a.settings.VinylHum,
		CrackleProb: a.settings.VinylCrackleProb,
	}
}

// vinylProgress is how far the song has played, where the stylus sits in
// the grooves.
func (a *app) vinylProgress() float32 {
	tr := a.pl.Current()
	if tr == nil {
		return 0
	}
	d := tr.Length()
	if d <= 0 {
		return 0
	}
	return clamp01(float32(a.pl.Position().Seconds() / d.Seconds()))
}

// vinylItem adds the surface noise's mix to a song's menu: it opens the
// 配比 window, where each layer is pushed on its own.
func (a *app) vinylItem(m *ui.Menu) {
	if m.Item("配比").Chosen() {
		a.openVinylMix()
	}
}

// openVinylMix shows the 配比 window, opening it the first time.
func (a *app) openVinylMix() {
	if a.vinylWin != nil {
		a.vinylWin.Show()
		a.vinylWin.Focus()
		return
	}
	a.vinylWin = mygo.NewWindow(mygo.WindowOptions{
		Title:           "配比",
		Width:           420,
		Height:          560,
		MinWidth:        360,
		MinHeight:       420,
		StateKey:        "vinyl-mix",
		DisableMaximize: true,
		DisableMinimize: true,
		AutoHideMenuBar: true,
		BackgroundColor: "light-dark(#f4f4f6, #1c1c1f)",
		Content:         ui.View(a.vinylMixView),
	})
	a.vinylWin.OnClosed(func() { a.vinylWin = nil })
}

// vinylMixEdited keeps the mix, applies it to the sound, and refreshes the
// window that shows it.
func (a *app) vinylMixEdited() {
	a.settings.save()
	a.applyVinylMix()
}

func (a *app) vinylMixView(c *ui.Context) {
	t := *c.Theme()
	t.Font = themeFont(a.settings.Font)
	p := paletteFor(&t)
	t.Accent, t.AccentText = p.accent, p.onAccent
	c.SetTheme(&t)
	c.Root().Background(p.sidebar)
	s := &a.settings
	ui.Scroll(c).Fill().Padding(20, 22, 22, 22).Gap(10).Children(func() {
		ui.Text(c, "配比").FontSize(18).FontWeight(700)
		ui.Text(c, "黑胶底噪的四层配比与爆裂概率：100% 是出厂配比，0 关闭该层。").FontSize(12).TextColor(p.muted).LineHeight(1.45)
		ui.Box(c).Height(4)

		ui.Column(c).Radius(12).Background(p.bg).Border(1, p.hairline).Dividers(1, p.hairline).Children(func() {
			row := func(label, key string, value *float64, max float64) {
				ui.Column(c).Padding(12, 16).Gap(7).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						ui.Text(c, label).FontSize(13.5).FontWeight(500)
						ui.Spacer(c)
						ui.Text(c, pctLabel(*value)).FontSize(12).TextColor(p.muted).FontFeatures("tnum")
					})
					sl := a.effectSlider(c, key, value, 0, max, label, p)
					if sl.Changed() {
						a.vinylMixEdited()
					}
				})
			}
			row("嘶声", "mix-hiss", &s.VinylHiss, 1.5)
			row("转盘隆隆", "mix-rumble", &s.VinylRumble, 1.5)
			row("爆裂", "mix-crackle", &s.VinylCrackle, 1.5)
			row("爆裂概率", "mix-prob", &s.VinylCrackleProb, 1)
			row("交流声", "mix-hum", &s.VinylHum, 1.5)
		})

		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c.Key("vinyl-mix-reset"), "恢复出厂配比").Clicked() {
				s.VinylHiss, s.VinylRumble, s.VinylCrackle, s.VinylHum = 1, 1, 1, 1
				s.VinylCrackleProb = 1
				a.vinylMixEdited()
				if a.vinylWin != nil {
					a.vinylWin.Invalidate()
				}
			}
			ui.Text(c, "0 即关闭：四层都不响时，底噪完全安静。").FontSize(11.5).TextColor(p.faint)
		})
	})
}

// applyVinylMix gives the player the mix of surface noise under the
// music, as the 配比 panel keeps it.
func (a *app) applyVinylMix() {
	if a.pl == nil {
		return
	}
	if a.settings.VinylOn {
		a.pl.SetVinylMix(a.vinylMix())
	} else {
		a.pl.SetVinylMix(player.VinylMix{})
	}
}

// setCover gives the app a cover style, turning the record's surface noise
// on as the cover takes the record's form.
func (a *app) setCover(style string) {
	a.settings.Cover = style
	// The record's noise goes with its cover: on with it, and off again
	// as the cover goes back to the plain sleeve.
	a.settings.VinylOn = style == "vinyl"
	a.settings.save()
	a.applyVinylMix()
	a.redraw()
	if a.settingsWin != nil {
		a.settingsWin.Invalidate()
	}
}

// vinylState is the record's turn and the tonearm's swing, kept between
// the frames that show them.
type vinylState struct {
	last  time.Time
	angle float32
	arm   float32 // 0 parked, 1 on the record
}

// vinylStep advances the record and the arm to now, and asks for the
// frames that show them: the record turns at 33 1/3 while it plays, and
// the arm swings down onto it, or lifts off as it pauses.
func (a *app) vinylStep(c *ui.Context, playing bool) (angle, arm float32) {
	v := &a.vinyl
	now := c.Now()
	if v.last.IsZero() {
		v.last = now
	}
	dt := float32(now.Sub(v.last).Seconds())
	v.last = now
	if dt < 0 {
		dt = 0
	}
	if dt > 0.25 {
		dt = 0.25
	}
	target := float32(0)
	if playing {
		target = 1
		// One turn every 19.8 s: about 3 RPM, 16.4° in 0.9 s.
		v.angle = float32(math.Mod(float64(v.angle+dt*360/19.8), 360))
	}
	if math.Abs(float64(v.arm-target)) > 0.0005 {
		v.arm += (target - v.arm) * min(1, dt*8)
		if math.Abs(float64(v.arm-target)) < 0.003 {
			v.arm = target
		}
	}
	if playing || math.Abs(float64(v.arm-target)) > 0.0005 {
		c.After(16 * time.Millisecond)
	}
	return v.angle, v.arm
}

// vinylRecord is the cover as a record: a disc of grooves with the cover
// as its label, turning as it plays, and a tonearm over it that swings
// down on play and lifts off on pause.
func (a *app) vinylRecord(c *ui.Context, bmp *ui.Bitmap, angle, arm, progress float32) ui.Element {
	return ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		drawVinylCover(p, r, bmp, angle, arm, progress)
	})
}

// drawVinylCover paints the record in r, the cover turning on it at angle,
// its tonearm at arm: 0 parked, 1 in the grooves, the stylus out at
// progress from the rim toward the label, as far as a single's grooves go.
func drawVinylCover(p *ui.Painter, r ui.Rect, bmp *ui.Bitmap, angle, arm, progress float32) {
	d := min(r.W, r.H)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	// The disc sits a little inside its box, so that the arm's pivot has
	// room outside it, as a turntable's does.
	H := d / 2
	R := H * 0.84
	disc := ui.Rect{X: cx - R, Y: cy - R, W: 2 * R, H: 2 * R}

	// The disc, its edge catching a little light.
	p.Shadow(disc, R, 0, d*0.035, d*0.09, -d*0.02, ui.RGBA(0, 0, 0, 0.55))
	p.Fill(disc, ui.Hex("#0c0c0e"), R)
	p.FillRadialGradient(disc, ui.RadialGradient{
		Stops:  []ui.GradientStop{{Color: ui.RGBA(255, 255, 255, 0.03), At: 0.2}, {Color: ui.RGBA(0, 0, 0, 0.35), At: 1}},
		Radius: R,
	}, R)

	// Grooves: rings from the label's edge to the rim.
	step := d * 0.008
	if d < 90 {
		step = d * 0.02
	}
	for i, rr := 0, R*0.66; rr < R*0.985; i, rr = i+1, rr+step {
		a := float32(0.035)
		if i%4 == 0 {
			a = 0.075
		}
		p.Stroke(ui.Rect{X: cx - rr, Y: cy - rr, W: 2 * rr, H: 2 * rr}, white.Alpha(a), rr, max(0.6, d*0.0016))
	}
	p.Stroke(disc, white.Alpha(0.12), R, 1)

	// The light on it, in two opposite fans, staying where the light is.
	p.Clip(disc, R, func() {
		p.FillConicGradient(disc, ui.ConicGradient{Angle: 20, Stops: []ui.GradientStop{
			{Color: white.Alpha(0), At: 0}, {Color: white.Alpha(0.11), At: 0.07}, {Color: white.Alpha(0), At: 0.19},
			{Color: white.Alpha(0), At: 0.49}, {Color: white.Alpha(0.10), At: 0.56}, {Color: white.Alpha(0), At: 0.68},
			{Color: white.Alpha(0), At: 1},
		}}, R)
	})

	// The label, the cover itself, turning with the record: a single's
	// picture disc keeps its grooves in a ring at the rim.
	lr := R * 0.63
	label := ui.Rect{X: cx - lr, Y: cy - lr, W: 2 * lr, H: 2 * lr}
	p.Fill(label, ui.Hex("#1a1a1d"), lr)
	if bmp != nil {
		p.Clip(label, lr, func() {
			p.Transform(ui.Rotation(angle), cx, cy, func() { p.Image(bmp, label, ui.Cover) })
		})
	}
	p.Stroke(label, ui.RGBA(255, 255, 255, 0.16), lr, max(1, d*0.002))
	hole := max(2.5, d*0.012)
	p.Fill(ui.Rect{X: cx - hole, Y: cy - hole, W: 2 * hole, H: 2 * hole}, ui.Hex("#0b0b0c"), hole)
	p.Stroke(ui.Rect{X: cx - hole, Y: cy - hole, W: 2 * hole, H: 2 * hole}, white.Alpha(0.25), hole, 1)

	// The arm turns about its pivot: parked just off the record, down onto
	// the grooves as it plays, and only a few degrees inward as the song
	// goes, as a single's does. Its length never changes, so it swings
	// rather than stretches.
	const armLen = 1.25 // of R
	pivot := ui.Point{X: cx + R*1.05, Y: cy - R*0.85}
	// The angles put the stylus at 0.950R at the start and 0.792R at the
	// end, as this record's grooves go.
	onDeg := 98.4 + (105.83-98.4)*progress
	deg := 90 + (onDeg-90)*arm
	rad := float64(deg) * math.Pi / 180
	dirX, dirY := float32(math.Cos(rad)), float32(math.Sin(rad))
	at := ui.Point{X: pivot.X + dirX*R*armLen, Y: pivot.Y + dirY*R*armLen}
	armW := max(1.6, d*0.025)
	p.StrokePolyline([][2]float32{{pivot.X + 1, pivot.Y + 2}, {at.X + 1, at.Y + 2}}, armW, ui.RGBA(0, 0, 0, 0.35))
	p.StrokePolyline([][2]float32{{pivot.X, pivot.Y}, {at.X, at.Y}}, armW, ui.Hex("#d9d9df"))
	// Its head, a short thick piece along the arm at the stylus.
	p.StrokePolyline([][2]float32{
		{at.X - dirX*d*0.035, at.Y - dirY*d*0.035},
		{at.X + dirX*d*0.055, at.Y + dirY*d*0.055},
	}, d*0.075, ui.Hex("#b9b9c2"))
	p.Fill(ui.Rect{X: at.X - d*0.02, Y: at.Y - d*0.02, W: d * 0.04, H: d * 0.04}, ui.Hex("#efeff4"), d*0.02)
	// The pivot and its counterweight.
	cw := ui.Point{X: pivot.X - dirX*d*0.07, Y: pivot.Y - dirY*d*0.07}
	p.Fill(ui.Rect{X: cw.X - d*0.05, Y: cw.Y - d*0.05, W: d * 0.10, H: d * 0.10}, ui.Hex("#2a2a30"), d*0.05)
	p.Fill(ui.Rect{X: pivot.X - d*0.05, Y: pivot.Y - d*0.05, W: d * 0.10, H: d * 0.10}, ui.Hex("#3a3a42"), d*0.05)
	p.Fill(ui.Rect{X: pivot.X - d*0.02, Y: pivot.Y - d*0.02, W: d * 0.04, H: d * 0.04}, ui.Hex("#c9c9d2"), d*0.02)
}
