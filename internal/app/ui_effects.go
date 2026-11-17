package app

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// openEffects shows the audio effects window, opening it the first time.
func (a *app) openEffects() {
	if a.effectsWin != nil {
		a.effectsWin.Show()
		a.effectsWin.Focus()
		return
	}
	a.syncEffectPositions()
	a.effectsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:                "音频效果器",
		Width:                880,
		Height:               720,
		MinWidth:             760,
		MinHeight:            560,
		StateKey:             "effects",
		DisableMaximize:      true,
		DisableMinimize:      true,
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 19},
		AutoHideMenuBar:      true,
		BackgroundColor:      "light-dark(#f4f4f6, #1c1c1f)",
		Content:              ui.View(a.effectsView),
	})
	a.effectsWin.OnClosed(func() { a.effectsWin = nil })
}

// syncEffectPositions reads the cutoffs into the log positions their
// sliders bind to.
func (a *app) syncEffectPositions() {
	s := &a.settings
	a.effectsHPos = effectLogPosition(s.Effects.Highpass, 20, 2000)
	a.effectsLPos = effectLogPosition(s.Effects.Lowpass, 500, 20000)
}

// effectsEdited keeps what the effects window changed, applies it to the
// sound, and refreshes the windows that show it.
func (a *app) effectsEdited() {
	s := &a.settings
	s.Effects = s.Effects.normalized()
	switch {
	case s.Effects.Preset == "custom1" || s.Effects.Preset == "custom2":
		idx := int(s.Effects.Preset[len(s.Effects.Preset)-1] - '1')
		slot := s.Effects
		slot.Preset = ""
		for len(s.EffectSlots) <= idx {
			s.EffectSlots = append(s.EffectSlots, effectSettings{})
		}
		s.EffectSlots[idx] = slot
	default:
		if id := s.Effects.effectSettingsForPreset(); id != "" {
			s.Effects.Preset = id
		} else {
			s.Effects.Preset = ""
		}
	}
	a.applyEffects()
	a.settings.save()
	if a.effectsWin != nil {
		a.effectsWin.Invalidate()
	}
	if a.settingsWin != nil {
		a.settingsWin.Invalidate()
	}
	a.redraw()
}

func hzLabel(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1f kHz", v/1000)
	}
	return fmt.Sprintf("%.0f Hz", v)
}

func pctLabel(v float64) string {
	return fmt.Sprintf("%.0f%%", v*100)
}

func bandLabel(i int) string {
	return [...]string{"31", "62", "125", "250", "500", "1k", "2k", "4k", "8k", "16k"}[i]
}

// effectEqSlider is one band's vertical slider: a track through zero, and
// a round knob, as the effects panel shows them.
func (a *app) effectEqSlider(c *ui.Context, key string, value *float64, pal palette) ui.Element {
	const knob = float32(12)
	s := ui.SliderBase(c.Key(key), value, -12, 12).Vertical().Size(24, 200).Step(0.1).FocusRing(false).Label("频段增益")
	s.Draw(func(p *ui.Painter, r ui.Rect) {
		x := r.X + r.W/2
		top, foot := r.Y+knob/2, r.Y+r.H-knob/2
		zero := r.Y + r.H/2
		v := clamp01(float32((*value + 12) / 24))
		y := foot - (foot-top)*v
		p.Fill(ui.Rect{X: x - 1, Y: top, W: 2, H: foot - top}, pal.hairline, 1)
		if y != zero {
			y0, y1 := zero, y
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			p.Fill(ui.Rect{X: x - 1, Y: y0, W: 2, H: y1 - y0}, pal.accent, 1)
		}
		p.Fill(ui.Rect{X: r.X + 2, Y: zero - 0.5, W: r.W - 4, H: 1}, pal.faint, 0.5)
		k := ui.Rect{X: x - knob/2, Y: y - knob/2, W: knob, H: knob}
		p.Shadow(k, knob/2, 0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.3))
		p.Fill(k, pal.text, knob/2)
	})
	return s
}

// effectSlider is a slider with a drawn track, its filled part and knob,
// for the effect chain's controls.
func (a *app) effectSlider(c *ui.Context, key string, value *float64, lo, hi float64, label string, pal palette) ui.Element {
	s := ui.SliderBase(c.Key(key), value, lo, hi).Grow(1).Height(28).FocusRing(false).Label(label)
	s.Draw(func(p *ui.Painter, r ui.Rect) {
		knob := float32(16)
		y := r.Y + r.H/2
		v := clamp01(float32((*value - lo) / (hi - lo)))
		p.Fill(ui.Rect{X: r.X, Y: y - 2, W: r.W, H: 4}, pal.hairline, 2)
		p.Fill(ui.Rect{X: r.X, Y: y - 2, W: max(knob/2, r.W*v), H: 4}, pal.accent, 2)
		k := ui.Rect{X: r.X + r.W*v - knob/2, Y: y - knob/2, W: knob, H: knob}
		p.Shadow(k, knob/2, 0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.25))
		p.Fill(k, pal.text, knob/2)
	})
	return s
}

func (a *app) effectsView(c *ui.Context) {
	t := *c.Theme()
	t.Font = themeFont(a.settings.Font)
	p := paletteFor(&t)
	t.Accent, t.AccentText = p.accent, p.onAccent
	c.SetTheme(&t)
	c.Root().Background(p.sidebar)
	c.OnShortcut(ui.Cmd, ui.KeyW, func() {
		if a.effectsWin != nil {
			a.effectsWin.Close()
		}
	})
	s := &a.settings
	eff := &s.Effects

	applyPreset := func(ps audioEffectPreset) {
		eff.applyAudioPreset(ps)
		a.syncEffectPositions()
		a.effectsEdited()
	}
	slotAt := func(i int) effectSettings {
		if i < len(s.EffectSlots) {
			return s.EffectSlots[i]
		}
		return effectSettings{}
	}
	storeSlot := func(i int, v effectSettings) {
		for len(s.EffectSlots) <= i {
			s.EffectSlots = append(s.EffectSlots, effectSettings{})
		}
		s.EffectSlots[i] = v
	}
	useSlot := func(i int) {
		if slot := slotAt(i); slot != (effectSettings{}) {
			*eff = slot
		} else {
			slot := *eff
			slot.Preset = ""
			storeSlot(i, slot)
		}
		eff.Preset = "custom" + strconv.Itoa(i+1)
		a.syncEffectPositions()
		a.effectsEdited()
	}
	resetAll := func() {
		*eff = neutralEffectSettings()
		a.syncEffectPositions()
		a.effectsEdited()
	}

	// The top bar: the title drags the window, the switch and the cross do
	// not.
	left, right := float32(20), float32(20)
	if runtime.GOOS == "darwin" {
		left = 84 // the traffic lights
	} else {
		right = 64 // the window's own close button
	}
	ui.Row(c).Height(60).Padding(0, right, 0, left).Gap(12).AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, "音频效果器").FontSize(15).FontWeight(700)
			ui.Text(c, "频段均衡与音频效果").FontSize(11.5).TextColor(p.muted)
		})
		status := "音频处理已关闭"
		if s.EffectsEnabled {
			status = "音频处理已开启"
		}
		ui.Text(c, status).FontSize(12).TextColor(p.muted)
		if ui.Switch(c.Key("effects-master"), &s.EffectsEnabled).Label(status).Changed() {
			a.changed()
		}
	})

	ui.Scroll(c).Fill().Padding(20, 24, 24, 24).Gap(16).Children(func() {
		// Presets and custom slots, with reset at the far end.
		ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
			chip := func(id, label string, active bool, click func()) {
				b := ui.ButtonBase(c.Key("chip-"+id)).Padding(7, 11).Radius(8).FocusRing(false).Label(label)
				bg, fg := p.hover, p.muted
				if active {
					bg, fg = p.press, p.text
				} else if b.Hovered() {
					bg, fg = p.press, p.text
				}
				b.Background(bg).Border(1, p.hairline)
				if active {
					b.Border(1, p.faint)
				}
				b.Children(func() {
					ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(fg)
				})
				if b.Clicked() {
					click()
				}
			}
			for _, ps := range audioEffectPresets {
				preset := ps
				chip(preset.id, preset.label, eff.Preset == preset.id, func() { applyPreset(preset) })
			}
			for i := 0; i < 2; i++ {
				idx := i
				chip("custom"+strconv.Itoa(i+1), "自定义 "+strconv.Itoa(i+1), eff.Preset == "custom"+strconv.Itoa(i+1), func() { useSlot(idx) })
			}
			ui.Spacer(c)
			if iconButton(c.Key("effects-reset"), iconReload, 16, p.muted, "重置").Clicked() {
				resetAll()
			}
		})

		ui.Text(c, "频段增益").FontSize(12).FontWeight(600).TextColor(p.muted).Margin(0, 2)

		// The ten bands: the scale at the left, then a column each.
		ui.Column(c).Radius(14).Background(p.bg).Border(1, p.hairline).Padding(14, 18).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Width(46).Height(244).Justify(ui.SpaceBetween).Children(func() {
					ui.Text(c, "+12 dB").FontSize(10.5).TextColor(p.faint)
					ui.Text(c, "0").FontSize(10.5).TextColor(p.faint)
					ui.Text(c, "−12 dB").FontSize(10.5).TextColor(p.faint)
				})
				for i := range eff.Gains {
					index := i
					ui.Column(c).Grow(1).Basis(0).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Text(c, fmt.Sprintf("%.0f", eff.Gains[index])).FontSize(10.5).TextColor(p.muted).FontFeatures("tnum")
						sl := a.effectEqSlider(c, "band-"+strconv.Itoa(index), &eff.Gains[index], p)
						if sl.Changed() {
							a.effectsEdited()
						}
						ui.Text(c, bandLabel(index)).FontSize(11).TextColor(p.muted)
					})
				}
				ui.Column(c).Width(22).Height(244).Justify(ui.End).Children(func() {
					ui.Text(c, "Hz").FontSize(10.5).TextColor(p.faint)
				})
			})
		})

		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "效果链").FontSize(12).FontWeight(600).TextColor(p.muted)
			ui.Spacer(c)
			ui.Text(c, "在频段之后处理").FontSize(11).TextColor(p.faint)
		})

		// Nine controls, row by row across three columns.
		ui.Column(c).Radius(14).Background(p.bg).Border(1, p.hairline).Padding(16, 18).Children(func() {
			cell := func(label, value string, noise bool, build func()) {
				ui.Column(c).Gap(8).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(p.text)
						if noise {
							ui.Text(c, "噪声").FontSize(9.5).Padding(1, 5).Radius(4).Background(p.hover).TextColor(p.muted)
						}
						ui.Spacer(c)
						ui.Text(c, value).FontSize(12).TextColor(p.muted).FontFeatures("tnum")
					})
					build()
				})
			}
			hp := func() {
				sl := a.effectSlider(c, "fx-highpass", &a.effectsHPos, 0, 1000, "低频截止", p)
				if sl.Changed() {
					eff.Highpass = effectLogValue(a.effectsHPos, 20, 2000)
					a.effectsEdited()
				}
			}
			lp := func() {
				sl := a.effectSlider(c, "fx-lowpass", &a.effectsLPos, 0, 1000, "高频截止", p)
				if sl.Changed() {
					eff.Lowpass = effectLogValue(a.effectsLPos, 500, 20000)
					a.effectsEdited()
				}
			}
			drive := func() {
				sl := a.effectSlider(c, "fx-drive", &eff.Drive, 0, 1, "饱和失真", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			crush := func() {
				sl := a.effectSlider(c, "fx-crush", &eff.Crush, 0, 1, "比特降质", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			wow := func() {
				sl := a.effectSlider(c, "fx-wow", &eff.Wow, 0, 1, "抖晃", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			noise := func() {
				sl := a.effectSlider(c, "fx-noise", &eff.Noise, 0, 1, "黑胶噪声", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			width := func() {
				sl := a.effectSlider(c, "fx-width", &eff.Width, 0, 2, "立体声宽度", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			space := func() {
				sl := a.effectSlider(c, "fx-space", &eff.Space, 0, 1, "空间混响", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			punch := func() {
				sl := a.effectSlider(c, "fx-punch", &eff.Punch, 0, 1, "动态压缩", p)
				if sl.Changed() {
					a.effectsEdited()
				}
			}
			ui.Row(c).Gap(28).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Gap(18).Children(func() {
					cell("低频截止", hzLabel(eff.Highpass), false, hp)
					cell("比特降质", pctLabel(eff.Crush), true, crush)
					cell("立体声宽度", pctLabel(eff.Width), false, width)
				})
				ui.Column(c).Grow(1).Basis(0).Gap(18).Children(func() {
					cell("高频截止", hzLabel(eff.Lowpass), false, lp)
					cell("抖晃", pctLabel(eff.Wow), false, wow)
					cell("空间混响", pctLabel(eff.Space), false, space)
				})
				ui.Column(c).Grow(1).Basis(0).Gap(18).Children(func() {
					cell("饱和失真", pctLabel(eff.Drive), false, drive)
					cell("黑胶噪声", pctLabel(eff.Noise), true, noise)
					cell("动态压缩", pctLabel(eff.Punch), false, punch)
				})
			})
		})
	})
}
