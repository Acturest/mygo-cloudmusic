package lyrics

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// A Line is a line of timed lyrics, with its translation if any, and
// when each of its words is sung.
type Line struct {
	At          time.Duration
	End         time.Duration
	Text        string
	Translation string
	Words       []Word
	// Timed says the words carry their own times, as NetEase's word-timed
	// lyrics give them; else they share the line's time evenly.
	Timed bool
	// Chorus says the line is one of the song's chorus, as Folia finds it:
	// the text that comes round most often in the lyrics.
	Chorus bool
}

// A Word is a word of a line, or a character of Chinese or Japanese,
// with when it is sung.
type Word struct {
	Text       string
	Start, End time.Duration
}

var lrcTime = regexp.MustCompile(`\[(\d+):(\d+)(?:[.:](\d+))?\]`)

// parseLRC reads lyrics in the LRC format, as NetEase serves them: a line
// may carry several times, and credits come as lines of their own.
func parseLRC(lrc string) []Line {
	var out []Line
	for _, raw := range strings.Split(lrc, "\n") {
		times := lrcTime.FindAllStringSubmatchIndex(raw, -1)
		if len(times) == 0 {
			continue
		}
		text := strings.TrimSpace(raw[times[len(times)-1][1]:])
		for _, m := range times {
			min, _ := strconv.Atoi(raw[m[2]:m[3]])
			sec, _ := strconv.Atoi(raw[m[4]:m[5]])
			var frac time.Duration
			if m[6] >= 0 {
				f := raw[m[6]:m[7]]
				n, _ := strconv.Atoi(f)
				frac = time.Duration(n) * time.Second
				for range len(f) {
					frac /= 10
				}
			}
			at := time.Duration(min)*time.Minute + time.Duration(sec)*time.Second + frac
			out = append(out, Line{At: at, Text: text})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

var (
	yrcLine = regexp.MustCompile(`^\[(\d+),(\d+)\]`)
	yrcWord = regexp.MustCompile(`\((\d+),(\d+),\d+\)([^(]*)`)
)

// parseYRC reads NetEase's word-timed lyrics: a line is
// "[start,length](start,length,0)word(start,length,0)word…", in
// milliseconds. Credits come as lines of JSON, which it leaves out.
func parseYRC(yrc string) []Line {
	ms := func(s string) time.Duration {
		n, _ := strconv.Atoi(s)
		return time.Duration(n) * time.Millisecond
	}
	var out []Line
	for _, raw := range strings.Split(yrc, "\n") {
		m := yrcLine.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		l := Line{At: ms(m[1]), Timed: true}
		l.End = l.At + ms(m[2])
		for _, w := range yrcWord.FindAllStringSubmatch(raw[len(m[0]):], -1) {
			start := ms(w[1])
			word := Word{Text: w[3], Start: start, End: start + ms(w[2])}
			if strings.TrimSpace(word.Text) == "" {
				// A space between words: it belongs to the word before.
				if n := len(l.Words); n > 0 {
					l.Words[n-1].Text += " "
				}
				continue
			}
			l.Words = append(l.Words, word)
		}
		for _, w := range l.Words {
			l.Text += w.Text
		}
		l.Text = strings.TrimSpace(l.Text)
		if l.Text != "" {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

// Build makes the lines to show from what NetEase serves: the
// word-timed lyrics where there are some and they keep the song's times,
// else the timed lines with each word's time estimated as Folia estimates
// it, and the translation of each.
func Build(lrc, translation, yrc string) []Line {
	plain := make([]Line, 0, 64)
	for _, l := range parseLRC(lrc) {
		if l.Text != "" {
			plain = append(plain, l)
		}
	}
	var lines []Line
	if yrc != "" {
		if timed := parseYRC(yrc); trustYRC(timed, plain) {
			lines = timed
		}
	}
	if len(lines) == 0 {
		lines = append(lines, plain...)
		for i := range lines {
			next, hasNext := time.Duration(0), i+1 < len(lines)
			if hasNext {
				next = lines[i+1].At
			}
			lines[i].End = lineEnd(lines[i].Text, lines[i].At, next, hasNext)
			lines[i].Words = SpreadWords(lines[i].Text, lines[i].At, lines[i].End)
		}
	}
	markChorus(lines, lrc)
	entries := make([]Line, 0, 64)
	for _, t := range parseLRC(translation) {
		if t.Text != "" {
			entries = append(entries, t)
		}
	}
	times := make([]time.Duration, len(lines))
	for i := range lines {
		times[i] = lines[i].At
	}
	for i, t := range pairTranslations(times, entries) {
		lines[i].Translation = t
	}
	return lines
}

// mergeLyrics pairs timed lines with their translations, without words'
// own times.
func mergeLyrics(lrc, translation string) []Line {
	return Build(lrc, translation, "")
}

// translationWindow is how far a translation may sit from the line it
// belongs to: word-timed lines start a little off the timed lines the
// translation follows.
const translationWindow = 1500 * time.Millisecond

// pairTranslations gives the lines at startTimes their translations, as
// Folia pairs them: closest first, each line and each translation used
// once, so that lines starting together take their own translations rather
// than both taking the first one's.
func pairTranslations(startTimes []time.Duration, entries []Line) []string {
	out := make([]string, len(startTimes))
	type pair struct {
		line, entry int
		gap         time.Duration
	}
	pairs := make([]pair, 0, len(startTimes)*2)
	for i, at := range startTimes {
		for j, e := range entries {
			gap := e.At - at
			if gap < 0 {
				gap = -gap
			}
			if gap < translationWindow {
				pairs = append(pairs, pair{i, j, gap})
			}
		}
	}
	sort.SliceStable(pairs, func(a, b int) bool {
		if pairs[a].gap != pairs[b].gap {
			return pairs[a].gap < pairs[b].gap
		}
		if pairs[a].line != pairs[b].line {
			return pairs[a].line < pairs[b].line
		}
		return pairs[a].entry < pairs[b].entry
	})
	lineTaken := make([]bool, len(startTimes))
	entryTaken := make([]bool, len(entries))
	for _, p := range pairs {
		if lineTaken[p.line] || entryTaken[p.entry] {
			continue
		}
		lineTaken[p.line], entryTaken[p.entry] = true, true
		out[p.line] = entries[p.entry].Text
	}
	return out
}

// lyricPunctuation is what is sung too fast to take time of its own.
const lyricPunctuation = "，。！？、：；\"'）"

// lineEnd is when a line sung from at ends, as Folia puts it: at the next
// line, but no later than a reading of it allows, so that a long break in
// the song stays a break instead of the line's words walking over it.
func lineEnd(text string, at, next time.Duration, hasNext bool) time.Duration {
	d := 5 * time.Second
	if hasNext {
		d = next - at
	}
	reading := time.Duration(float64(utf8.RuneCountInString(text)) * 0.5 * float64(time.Second))
	if d > reading+2*time.Second && d > 5*time.Second {
		d = min(d, reading+2*time.Second)
	}
	return at + d
}

// SpreadWords splits text into words, each Chinese, Japanese or Korean
// character one of them, and shares the time from start to end among them
// by their weights, as Folia spreads a line that has no word times of its
// own: a sung character or a word counts one, a word longer by its letters,
// and punctuation almost nothing. Singing takes nine tenths of the line,
// the rest being the pause before the next.
func SpreadWords(text string, start, end time.Duration) []Word {
	type token struct {
		text   string
		weight float64
	}
	var tokens []token
	var cur strings.Builder
	var total float64
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		w := 1 + float64(utf8.RuneCountInString(strings.TrimSpace(cur.String())))*0.15
		tokens = append(tokens, token{text: cur.String(), weight: w})
		total += w
		cur.Reset()
	}
	sung := false // the word before is Chinese or Japanese
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			if cur.Len() > 0 {
				cur.WriteRune(' ')
				flush()
			}
		case isCJK(r):
			flush()
			tokens = append(tokens, token{text: string(r), weight: 1})
			total++
			sung = true
		case sung && strings.ContainsRune(lyricPunctuation, r):
			// Punctuation after Chinese or Japanese is not sung long.
			flush()
			tokens = append(tokens, token{text: string(r)})
		default:
			cur.WriteRune(r)
			sung = false
		}
	}
	flush()
	if len(tokens) == 0 {
		return nil
	}
	span := end - start
	if span < 100*time.Millisecond {
		span = 100 * time.Millisecond
	}
	if total == 0 {
		total = 1
	}
	// Singing takes most of the line, not the pause before the next.
	per := time.Duration(float64(span) * 0.9 / total)
	out := make([]Word, len(tokens))
	at := start
	for i, t := range tokens {
		d := time.Duration(float64(per) * t.weight)
		out[i] = Word{Text: t.text, Start: at, End: at + max(d, 50*time.Millisecond)}
		if t.weight > 0 {
			at += d
		} else {
			at += 50 * time.Millisecond
		}
	}
	// A line that runs past its end is squeezed back into it, rather than
	// being sung into the next line's time.
	if last := out[len(out)-1]; last.End > end && last.End > start {
		scale := float64(end-start) / float64(last.End-start)
		for i := range out {
			out[i].Start = start + time.Duration(float64(out[i].Start-start)*scale)
			out[i].End = start + time.Duration(float64(out[i].End-start)*scale)
		}
	}
	return out
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

// Index returns the line sung at pos, or -1 before the first.
func Index(lines []Line, pos time.Duration) int {
	i := sort.Search(len(lines), func(i int) bool { return lines[i].At > pos })
	return i - 1
}

// A Quiet is a stretch of a song with nothing sung: its intro, an
// interlude, or its outro.
type Quiet struct {
	From, To time.Duration
}

// Quiets returns the stretches of a song length long, of at least least,
// in which none of lines is sung. Credits, as 作词 and 作曲, are not sung.
func Quiets(lines []Line, length, least time.Duration) []Quiet {
	var out []Quiet
	add := func(from, to time.Duration) {
		if to-from >= least {
			out = append(out, Quiet{from, to})
		}
	}
	at := time.Duration(0)
	sung := false
	for _, l := range lines {
		if IsCredit(l.Text) {
			continue
		}
		add(at, l.At)
		at, sung = max(at, l.End), true
	}
	if sung && length > 0 {
		add(at, length)
	}
	return out
}

// creditNames are what credit lines begin with.
var creditNames = []string{"作词", "作曲", "编曲", "制作", "监制", "混音", "录音", "母带", "和声", "吉他", "贝斯", "鼓", "键盘", "弦乐", "出品", "发行", "OP", "SP", "词", "曲", "Lyrics", "Music", "Composer", "Arranger", "Producer"}

// IsCredit reports whether a line names who made the song, as "作词 :
// 林夕", rather than being sung.
func IsCredit(text string) bool {
	i := strings.IndexAny(text, ":：")
	if i <= 0 || utf8.RuneCountInString(text[:i]) > 12 {
		return false
	}
	head := strings.TrimSpace(text[:i])
	for _, n := range creditNames {
		// A name of one character, as 词, is the whole of the head.
		if head == n || utf8.RuneCountInString(n) > 1 && strings.HasPrefix(head, n) {
			return true
		}
	}
	return false
}
