package main

// credit: whoever did http://play.golang.org/p/ABoHSHoTmu

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Authorization struct {
	Username, Password, Realm, NONCE, QOP, Opaque, Algorithm string
}

func GetAuthorization(username, password string, resp *http.Response) *Authorization {
	header := resp.Header.Get("www-authenticate")
	parts := strings.SplitN(header, " ", 2)
	parts = strings.Split(parts[1], ",")
	opts := make(map[string]string)

	for _, part := range parts {
		vals := strings.SplitN(strings.Trim(part, " "), "=", 2)
		key := vals[0]
		val := strings.Trim(vals[1], "\",")
		opts[key] = val
	}

	auth := Authorization{
		username, password,
		opts["realm"], opts["nonce"], opts["qop"], opts["opaque"], opts["algorithm"],
	}
	return &auth
}

func SetDigestAuth(r *http.Request, username, password string, resp *http.Response, nc int) {
	auth := GetAuthorization(username, password, resp)
	ApplyDigestAuth(r, auth, nc)
}

func ApplyDigestAuth(r *http.Request, auth *Authorization, nc int) {
	auth_str := GetAuthString(auth, r.URL, r.Method, nc)
	r.Header.Add("Authorization", auth_str)
}

func GetAuthString(auth *Authorization, url *url.URL, method string, nc int) string {
	a1 := auth.Username + ":" + auth.Realm + ":" + auth.Password
	h := md5.New()
	io.WriteString(h, a1)
	ha1 := hex.EncodeToString(h.Sum(nil))

	h = md5.New()
	a2 := method + ":" + url.Path
	io.WriteString(h, a2)
	ha2 := hex.EncodeToString(h.Sum(nil))

	nc_str := fmt.Sprintf("%08x", nc)
	hnc := "MTM3MDgw"

	respdig := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, auth.NONCE, nc_str, hnc, auth.QOP, ha2)
	h = md5.New()
	io.WriteString(h, respdig)
	respdig = hex.EncodeToString(h.Sum(nil))

	base := "username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\", response=\"%s\""
	base = fmt.Sprintf(base, auth.Username, auth.Realm, auth.NONCE, url.Path, respdig)
	if auth.Opaque != "" {
		base += fmt.Sprintf(", opaque=\"%s\"", auth.Opaque)
	}
	if auth.QOP != "" {
		base += fmt.Sprintf(", qop=\"%s\", nc=%s, cnonce=\"%s\"", auth.QOP, nc_str, hnc)
	}
	if auth.Algorithm != "" {
		base += fmt.Sprintf(", algorithm=\"%s\"", auth.Algorithm)
	}

	// r.Header.Add("Authorization", "Digest " +base)
	return "Digest " + base
}

// func main() {
//     auth := Authorization{
//         "Joe", "Schmoe", "RETS Server",
//         "0decbedc0f7828a0f8a0f0ea4a2107e3", "auth",
//         "a0531450f6a92cda6b30ae8e4de1cd2b", "",
//     }
//     myURL := url.URL{"http", "", nil, "www.dis.com:6103", "/rets/login", "", ""}
//     fmt.Println(GetAuthString(&auth, &myURL, "POST", 3))
// }
