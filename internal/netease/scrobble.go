package netease

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The scrobble goes to the clientlog host, as the desktop client's play log
// does. Posting the same logs to music.163.com is answered 200 and dropped
// before it reaches the record, which is why this speaks to that host as the
// macOS client: its address, user agent and app version are what the log is
// checked against.
const (
	scrobbleDomain = "https://clientlog.music.163.com"
	scrobbleAppVer = "3.1.10.5100"
	scrobbleOSVer  = "15.5"
	scrobbleUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
)

// Scrobble tells NetEase that a track played, so that it counts in the
// listener's record of what they play. source names where it played from, as
// the scrobble API's play sources do: "list", "album", "artist", "recommend",
// "search" or "download". sourceID is the playlist, album or artist it came
// from, 0 for a source without one.
//
// The report is two logs, as the client sends them: that playback started,
// then that it ended after so many seconds. NetEase answers 200 whether or not
// the listen reaches the record, and the counts catch up later.
func Scrobble(id, sourceID int64, source string, played time.Duration) error {
	if id == 0 {
		return fmt.Errorf("netease: scrobble without a song")
	}
	if source == "" {
		source = "list"
	}
	// The report keeps whole seconds, as the players it copies do.
	secs := int64(played / time.Second)
	if secs < 0 {
		secs = 0
	}
	// The log says which list, album or artist it came from as a page the
	// player was on: "id=" and the source's id, 0 when it had none.
	content := fmt.Sprintf("id=%d", sourceID)
	start := map[string]any{
		"action": "startplay",
		"json": map[string]any{
			"id":          id,
			"type":        "song",
			"mainsite":    "1",
			"mainsiteWeb": "1",
			"content":     content,
		},
	}
	play := map[string]any{
		"action": "play",
		"json": map[string]any{
			"download":    0,
			"end":         "playend",
			"id":          id,
			"sourceId":    sourceID,
			"time":        secs,
			"type":        "song",
			"wifi":        0,
			"source":      source,
			"mainsite":    "1",
			"mainsiteWeb": "1",
			"content":     content,
		},
	}
	if err := scrobbleLog(start); err != nil {
		return err
	}
	return scrobbleLog(play)
}

// scrobbleLog sends one log entry to the clientlog host through the eapi,
// with the headers of the macOS client that host expects.
func scrobbleLog(entry map[string]any) error {
	logs, err := json.Marshal([]map[string]any{entry})
	if err != nil {
		return err
	}
	data := map[string]any{"logs": string(logs)}
	header := map[string]string{
		"os":          "osx",
		"appver":      scrobbleAppVer,
		"osver":       scrobbleOSVer,
		"channel":     "netease",
		"deviceId":    device(),
		"versioncode": "140",
		"mobilename":  "",
		"buildver":    fmt.Sprint(time.Now().Unix()),
		"resolution":  "1920x1080",
		"requestId":   fmt.Sprintf("%d_%04d", time.Now().UnixMilli(), time.Now().Nanosecond()%1000),
		"__csrf":      "",
	}
	for _, c := range httpClient.Jar.Cookies(siteURL) {
		switch c.Name {
		case "MUSIC_U", "NMTID":
			header[c.Name] = c.Value
		case "__csrf":
			header["__csrf"] = c.Value
		}
	}
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
