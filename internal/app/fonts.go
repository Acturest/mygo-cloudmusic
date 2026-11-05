package app

import (
	_ "embed"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The app's own font, which the listener may set in the settings: 獅尾四季春
// 加糖SC (Swei Spring Sugar CJK SC), a serif after Noto Serif CJK, under the
// SIL Open Font License (fonts/OFL.txt). The file here is a subset of it
// made for this app — the characters of GB 2312 and JIS X 0208, kana, Latin
// and punctuation, without hinting, a quarter of its size (fonts/README.md).
// Whatever it lacks falls back to the system's fonts.
//
//go:embed fonts/SweiSpringSugarCJKsc-Regular.ttf
var sweiSpring []byte

// fontSweiSpring is the settings' name for that font.
const fontSweiSpring = "swei-spring"

// sweiSpringFamily is the family the font is registered under.
const sweiSpringFamily = "Swei Spring Sugar CJK SC"

var (
	registerSweiOnce sync.Once
	registerSweiErr  error
)

// themeFont is the family text is set in, "" for the system's: the app's own
// font is registered the first time it is asked for.
func themeFont(s string) string {
	if s != fontSweiSpring {
		return ""
	}
	registerSweiOnce.Do(func() { registerSweiErr = ui.RegisterFont(sweiSpring, sweiSpringFamily) })
	if registerSweiErr != nil {
		return ""
	}
	return sweiSpringFamily
}
