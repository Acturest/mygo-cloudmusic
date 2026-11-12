package netease

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The play log goes to the clientlog host, as the clients that still count do.
// Only the macOS client picture is credited there: the same log sent as the
// Windows, Android or iPhone client is answered 200 and dropped before it
// reaches "最近播放" or the listening record. So the log speaks as the macOS
// client while the session and the device it belongs to stay this app's.
const (
	scrobbleDomain = "https://clientlog.music.163.com"
	scrobbleOS     = "osx"
	scrobbleOSVer  = "15.5"
	scrobbleAppVer = "3.1.10.5100"
	// scrobbleUA is the macOS client's user agent; this path reads it against
	// the header, so the eapi's own is not used here.
	scrobbleUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// StartPlay tells NetEase that id started playing, from source, whose id is
// sourceID. The report is two logs, as the client sends them: that playback
// started, and later that it ended. NetEase answers 200 whether or not the
// listen reaches the record, and the counts catch up later.
//
// source names where it played from, as the play log's sources do: "list",
// "album", "artist", "recommend", "search" or "download". sourceID is the
// playlist, album or artist it came from, 0 for a source without one.
func StartPlay(id, sourceID int64, source string) error {
	if id == 0 {
		return fmt.Errorf("netease: startplay without a song")
	}
	return scrobbleLog(scrobbleEntry("startplay", id, sourceID, source, 0))
}

// PlayEnd tells NetEase that id played for played, from source, whose id is
// sourceID: the log the record counts, once the song ended or the listener
// left it.
func PlayEnd(id, sourceID int64, source string, played time.Duration) error {
	if id == 0 {
		return fmt.Errorf("netease: play without a song")
	}
	// The log keeps whole seconds, as the players it copies do, and never
	// longer than the song: a report of a song longer than it played reads
	// as one nobody made.
	secs := int64(played / time.Second)
	if secs < 0 {
		secs = 0
	}
	return scrobbleLog(scrobbleEntry("play", id, sourceID, source, secs))
}

// scrobbleEntry is one log entry: that playback started, or that it played
// for secs. The log says which list, album or artist the song came from as
// the page the player was on: "id=" and the source's id, 0 when it had none.
func scrobbleEntry(action string, id, sourceID int64, source string, secs int64) map[string]any {
	if source == "" {
		source = "list"
	}
	content := fmt.Sprintf("id=%d", sourceID)
	entry := map[string]any{
		"id":          id,
		"type":        "song",
		"mainsite":    "1",
		"mainsiteWeb": "1",
		"content":     content,
	}
	if action == "play" {
		entry["download"] = 0
		entry["end"] = "playend"
		entry["sourceId"] = sourceID
		entry["time"] = secs
		entry["wifi"] = 0
		entry["source"] = source
	}
	return map[string]any{"action": action, "json": entry}
}

// scrobbleHeader is the header the macOS client's logs carry: which client
// and device sent them, and the session's cookie. It goes in the encrypted
// envelope and, field by field, in the request's Cookie header.
func scrobbleHeader() map[string]string {
	header := map[string]string{
		"osver":       scrobbleOSVer,
		"deviceId":    device(),
		"os":          scrobbleOS,
		"appver":      scrobbleAppVer,
		"versioncode": "140",
		"mobilename":  "",
		"buildver":    fmt.Sprint(time.Now().Unix()),
		"resolution":  "1920x1080",
		"channel":     "netease",
		"requestId":   fmt.Sprintf("%d_%04d", time.Now().UnixMilli(), rand.Intn(10000)),
		"__csrf":      "",
	}
	for _, c := range httpClient.Jar.Cookies(siteURL) {
		switch c.Name {
		case "MUSIC_U", "MUSIC_A", "__csrf":
			header[c.Name] = c.Value
		}
	}
	return header
}

// scrobbleLog sends one log entry to the clientlog host through the eapi,
// with the header of the macOS client the log speaks as. Its time is the time
// it is sent: the client logs playback as it goes, not in hindsight.
func scrobbleLog(entry map[string]any) error {
	logs, err := json.Marshal([]map[string]any{entry})
	if err != nil {
		return err
	}
	data := map[string]any{"logs": string(logs)}
	header := scrobbleHeader()
	data["header"] = header
	data["e_r"] = false
	params, err := eapiEncrypt("/api/feedback/weblog", data)
	if err != nil {
		return err
	}
	// The path under /eapi/ drops the /api prefix, as the client's does.
	req, err := http.NewRequest("POST", scrobbleDomain+"/eapi/feedback/weblog",
		strings.NewReader(url.Values{"params": {params}}.Encode()))
	if err != nil {
		return err
	}
	var cookie []string
	for k, v := range header {
		cookie = append(cookie, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	req.Header.Set("Cookie", strings.Join(cookie, "; "))
	req.Header.Set("User-Agent", scrobbleUA)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Sent by hand, as the eapi's are: the jar would add the site's cookies.
	res, err := (&http.Client{Timeout: httpClient.Timeout}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var out apiError
	if err := json.Unmarshal(body, &out); err != nil {
		return err
	}
	if out.Code != 200 {
		msg := out.Message
		if msg == "" {
			msg = out.Msg
		}
		return fmt.Errorf("netease: scrobble: code %d %s", out.Code, msg)
	}
	return nil
}
