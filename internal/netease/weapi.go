package netease

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
)

// The web player's own API: its cookie, its csrf token and the encrypted
// parameters every browser copy of the player sends. Writes that must stick
// go through it: NetEase answers the eapi's playlist additions 200 and then
// drops them, and the browser's like call is the one the service credits.

const (
	weapiPresetKey = "0CoJUm6Qyw8W8jud"
	weapiIV        = "0102030405060708"
	weapiAlphabet  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	weapiAppVer    = "2.7.1.198277"
	weapiUA        = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// weapiModulus is the public key the player encrypts a request's key
	// with, as hex.
	weapiModulus = "e0b509f6259df8642dbc35662901477df22677ec152b5ff68ace615bb7b725152b3ab17a876aea8a5aa76d2e417629ec4ee341f56135fccf695280104e0312ecbda92557c93870114af6c9d05c4f7f0c3685b7a46bee255932575cce10b424d813cfe4875d3e82047b97ddef52741d546b8e289dc6935b3ece0462db0a22b8e7"
)

// weapiPost sends data through the web player's API at path (as
// "/api/radio/like") and returns what NetEase answered.
func weapiPost(path string, data map[string]any) (*apiError, error) {
	params, encSecKey, err := weapiEncode(data)
	if err != nil {
		return nil, err
	}
	form := url.Values{"params": {params}, "encSecKey": {encSecKey}}
	csrf := weapiCookieValue("__csrf")
	u := apiBase + strings.Replace(path, "/api/", "/weapi/", 1) + "?csrf_token=" + url.QueryEscape(csrf)
	req, err := http.NewRequest("POST", u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", weapiUA)
	req.Header.Set("Referer", apiBase+"/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", weapiCookies())
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var status apiError
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("netease: %s: %w", path, err)
	}
	return &status, nil
}

// weapiCookies is the cookie the web player sends with its calls: the
// client it is, then the session's cookies.
func weapiCookies() string {
	parts := []string{"os=pc", "appver=" + weapiAppVer}
	for _, c := range httpClient.Jar.Cookies(siteURL) {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

func weapiCookieValue(name string) string {
	for _, c := range httpClient.Jar.Cookies(siteURL) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func weapiEncode(data map[string]any) (string, string, error) {
	key, err := weapiKey()
	if err != nil {
		return "", "", err
	}
	return weapiEncrypt(data, key)
}

// weapiEncrypt seals data twice with AES-CBC, with the player's preset key
// and then key, and hands key over RSA-encrypted, as the browser does.
func weapiEncrypt(data map[string]any, key string) (string, string, error) {
	text, err := json.Marshal(data)
	if err != nil {
		return "", "", err
	}
	first, err := weapiAES(string(text), weapiPresetKey)
	if err != nil {
		return "", "", err
	}
	params, err := weapiAES(first, key)
	if err != nil {
		return "", "", err
	}
	return params, weapiRSA(key), nil
}

func weapiAES(text, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	pad := aes.BlockSize - len(text)%aes.BlockSize
	plain := append([]byte(text), bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, []byte(weapiIV)).CryptBlocks(out, plain)
	return base64.StdEncoding.EncodeToString(out), nil
}

func weapiRSA(key string) string {
	reversed := []byte(key)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	mod, _ := new(big.Int).SetString(weapiModulus, 16)
	out := new(big.Int).Exp(new(big.Int).SetBytes(reversed), big.NewInt(65537), mod).Text(16)
	if len(out) < 256 {
		out = strings.Repeat("0", 256-len(out)) + out
	}
	return out
}

func weapiKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = weapiAlphabet[int(b[i])%len(weapiAlphabet)]
	}
	return string(b), nil
}
