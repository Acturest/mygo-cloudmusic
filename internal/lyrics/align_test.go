package lyrics

import (
	"testing"
	"time"
)

func TestTrustYRCKeepsAlignedLines(t *testing.T) {
	plain := parseLRC("[00:10.00]一\n[00:20.00]二\n[00:30.00]三\n[00:40.00]四\n")
	timed := parseYRC("[10000,2000](10000,2000,0)一\n[20300,2000](20300,2000,0)二\n[30000,2000](30000,2000,0)三\n[40500,2000](40500,2000,0)四\n")
	if !trustYRC(timed, plain) {
		t.Fatalf("lines half a second apart were not trusted")
	}
}

func TestTrustYRCDropsLinesThatLostTheSong(t *testing.T) {
	// As 場末 - DADARAY's: a break in the song is missing from the word
	// times, so every line after it is early, by more and more.
	plain := parseLRC("[00:10.00]一\n[00:20.00]二\n[00:30.00]三\n[00:90.00]四\n[01:40.00]五\n")
	timed := parseYRC("[10000,2000](10000,2000,0)一\n[20000,2000](20000,2000,0)二\n[30000,2000](30000,2000,0)三\n[60000,2000](60000,2000,0)四\n[90000,2000](90000,2000,0)五\n")
	if trustYRC(timed, plain) {
		t.Fatalf("lines losing a break were trusted")
	}
}

func TestTrustYRCDropsLinesThatLostTheSongWithoutMatchingWords(t *testing.T) {
	// The word-timed line is cut short of the timed line's words, so the
	// two cannot be paired by them; they are still the same lines in order.
	plain := parseLRC("[00:10.00]あなたから\n[00:20.00]終電まで\n[00:30.00]悲しみが\n[00:40.00]走り出す\n")
	timed := parseYRC("[10000,2000](10000,2000,0)あ\n[20000,2000](20000,2000,0)終\n[30000,2000](30000,2000,0)悲\n[60000,2000](60000,2000,0)走\n")
	if trustYRC(timed, plain) {
		t.Fatalf("lines losing a break were trusted")
	}
}

func TestTrustYRCWithoutEnoughToCompare(t *testing.T) {
	plain := parseLRC("[00:10.00]一\n")
	timed := parseYRC("[10000,2000](10000,2000,0)一\n")
	if !trustYRC(timed, plain) {
		t.Fatalf("a song with too few lines should keep its word times")
	}
}

func TestBuildFallsBackWhenWordTimesAreOff(t *testing.T) {
	lrc := "[00:10.00]从不 主动 示弱\n[03:02.95]あなたから\n[03:05.70]終電まで\n[03:09.20]悲しみが\n[03:12.52]もう遠く\n"
	yrc := "[10000,3000](10000,900,0)从(10900,900,0)不(11800,900,0)\n" +
		"[154630,6300](154630,1260,0)あなたから\n" +
		"[160960,270](160960,67,0)終電まで\n" +
		"[161460,3130](161460,780,0)悲しみが\n" +
		"[164590,3320](164590,830,0)もう遠く\n"
	lines := Build(lrc, "", yrc)
	if len(lines) != 5 || lines[0].Timed {
		t.Fatalf("lines %+v", lines)
	}
	// The timed lyrics' time is the one kept: 3:02.95, as the translation
	// and the ears have it, and not the word-timed 2:34.63.
	if lines[1].At != 182950*time.Millisecond {
		t.Errorf("line at %v, want 3:02.95", lines[1].At)
	}
	if words := lines[1].Words; len(words) != 5 || words[0].Start != 182950*time.Millisecond {
		t.Errorf("words %+v", words)
	}
}

func TestLineEndKeepsALongBreak(t *testing.T) {
	// 場末 - DADARAY: 2:29.03 必要以上の乾杯, then nothing till 3:02.95.
	end := lineEnd("必要以上の乾杯", 149030*time.Millisecond, 182950*time.Millisecond, true)
	if end != 154530*time.Millisecond {
		t.Errorf("line ends at %v, want 2:34.53", end)
	}
	if end := lineEnd("一", 10*time.Second, 0, false); end != 15*time.Second {
		t.Errorf("last line ends at %v, want 15s", end)
	}
}

func TestBuildKeepsWordTimesThatAgree(t *testing.T) {
	lrc := "[00:10.00]从不\n[00:20.00]主动\n[00:30.00]示弱\n[00:40.00]曾经\n"
	yrc := "[10010,900](10010,450,0)从(10460,450,0)不\n" +
		"[20000,900](20000,450,0)主(20450,450,0)动\n" +
		"[30000,900](30000,450,0)示(30450,450,0)弱\n" +
		"[40000,900](40000,450,0)曾(40450,450,0)经\n"
	lines := Build(lrc, "", yrc)
	if len(lines) != 4 || !lines[0].Timed || lines[0].At != 10010*time.Millisecond {
		t.Fatalf("lines %+v", lines)
	}
}

func TestSpreadWordsGivesPunctuationLittleTime(t *testing.T) {
	s := time.Second
	w := SpreadWords("悲しみが、", 0, 4*s)
	if len(w) != 5 {
		t.Fatalf("words %+v", w)
	}
	if last := w[4]; last.Text != "、" || last.End-last.Start > 60*time.Millisecond {
		t.Errorf("punctuation took %v", last.End-last.Start)
	}
	if w[0].End-w[0].Start <= 0 || w[0].Start != 0 {
		t.Errorf("first word %+v", w[0])
	}
	if joined := wordsText(w); joined != "悲しみが、" {
		t.Errorf("words join to %q", joined)
	}
}

func wordsText(words []Word) string {
	text := ""
	for _, w := range words {
		text += w.Text
	}
	return text
}
func TestPairTranslationsGivesLinesTheirOwn(t *testing.T) {
	// Two lines a moment apart, each with the translation that follows it:
	// the second no longer has to do without.
	times := []time.Duration{10 * time.Second, 10400 * time.Millisecond}
	entries := parseLRC("[00:10.10]one\n[00:10.50]two\n")
	got := pairTranslations(times, entries)
	if got[0] != "one" || got[1] != "two" {
		t.Fatalf("translations were %q, want one and two", got)
	}
}

func TestPairTranslationsUsesEachTranslationOnce(t *testing.T) {
	// Lines starting together take the closer translation, and the other
	// line waits for one of its own rather than taking a neighbour's.
	times := []time.Duration{10 * time.Second, 10050 * time.Millisecond}
	entries := parseLRC("[00:10.02]shared\n")
	got := pairTranslations(times, entries)
	if got[0] != "shared" || got[1] != "" {
		t.Fatalf("translations were %q, want shared and nothing", got)
	}
}

func TestPairTranslationsLetsFarLinesWait(t *testing.T) {
	// A line whose translation is further off than the window stays
	// without one, as a translation of another line does.
	times := []time.Duration{10 * time.Second, 60 * time.Second}
	entries := parseLRC("[00:15.00]elsewhere\n")
	got := pairTranslations(times, entries)
	if got[0] != "" || got[1] != "" {
		t.Fatalf("translations were %q, want none", got)
	}
}