package app

import "netease-music-mygo/internal/player"

// Light mode is the window away with the music playing on in the
// notification area, as Clash Verge's light mode is: no frame is built,
// laid out or drawn, so what the views animated stops and the CPU and GPU
// work of drawing goes with it. The queue plays on repeat meanwhile, so
// the music does not fall silent, and the icon brings the window back as
// the listener left it: the same page, the repeat mode it had.

// setLite turns light mode on or off.
func (a *app) setLite(on bool) {
	if on == a.lite {
		return
	}
	if on {
		a.liteRepeat = a.pl.Repeat()
		a.pl.SetRepeat(player.RepeatAll)
		a.lite = true
		if a.win != nil {
			a.win.Hide()
		}
		// Away for a while: the covers shown, and the heap, go back.
		a.releaseMemory()
	} else {
		a.lite = false
		a.pl.SetRepeat(a.liteRepeat)
		a.showMain()
	}
	a.refreshTrayMenu()
	a.redraw()
}

// liteLabel names the light mode item of the icon's menu, as it does what
// the window now lacks.
func (a *app) liteLabel() string {
	if a.lite {
		return "退出轻量模式"
	}
	return "轻量模式"
}

// showWindow brings the window back from light mode, or forward where it
// is already showing: the icon's left click, its 打开 item, and the app
// brought forward all come here, so light mode ends wherever the window
// does come back.
func (a *app) showWindow() {
	if a.lite {
		a.setLite(false)
		return
	}
	a.showMain()
}
