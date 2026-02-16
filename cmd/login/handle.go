// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"changkun.de/x/login/internal/uuid"
	"github.com/golang-jwt/jwt"
)

var errUnauthorized = errors.New("request unauthorized")

// blocklist holds the ip that should be blocked for further requests.
//
// This map may keep grow without releasing memory because of
// continuously attempts. we also do not persist this type of block info
// to the disk, which means if we reboot the service then all the blocker
// are gone and they can attack the server again.
// We clear the map very month.
var blocklist sync.Map // map[string]*blockinfo{}

func init() {
	t := time.NewTicker(time.Hour * 24 * 30)
	go func() {
		for range t.C {
			blocklist.Range(func(k, v interface{}) bool {
				blocklist.Delete(k)
				return true
			})
		}
	}()
}

type blockinfo struct {
	failCount int64
	lastFail  atomic.Value // time.Time
	blockTime atomic.Value // time.Duration
}

const maxFailureAttempts = 10

// loginForm is a login credentials
type loginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Redirect string `json:"redirect"`
}

var hmacSecret []byte

func init() {
	secret := os.Getenv("LOGIN_SECRET")
	if secret == "" {
		log.Fatal("LOGIN_SECRET is missing.")
	}

	hmacSecret = []byte(secret)
}

func authfunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Cache-Control", "max-age=0")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		return
	}

	var err error
	defer func() {
		if err == nil {
			return
		}

		if errors.Is(err, errUnauthorized) {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		log.Println(err)
	}()
	if r.Method != http.MethodPost {
		err = errors.New("unsupported method")
		return
	}

	// Load login body.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed to read request body: %w", err)
		return
	}
	lo := &loginForm{}
	err = json.Unmarshal(b, lo)
	if err != nil {
		err = fmt.Errorf("failed to parse request body: %w", err)
		return
	}

	// check if the IP failure attempts are too much
	// if so, direct abort the request without checking credentials
	ip := readIP(r)
	if i, ok := blocklist.Load(ip); ok {
		info := i.(*blockinfo)
		count := atomic.LoadInt64(&info.failCount)
		if count > maxFailureAttempts {
			// if the ip is under block, then directly abort
			last := info.lastFail.Load().(time.Time)
			bloc := info.blockTime.Load().(time.Duration)

			if time.Now().UTC().Sub(last.Add(bloc)) < 0 {
				log.Printf("block ip %v, too much failure attempts. Block time: %v, release until: %v\n",
					ip, bloc, last.Add(bloc))
				err = fmt.Errorf("%w: too much failure attempts", errUnauthorized)
				return
			}

			// clear the failcount, but increase the next block time
			atomic.StoreInt64(&info.failCount, 0)
			info.blockTime.Store(bloc * 2)
		}
	}

	defer func() {
		if !errors.Is(err, errUnauthorized) {
			return
		}

		if i, ok := blocklist.Load(ip); !ok {
			info := &blockinfo{
				failCount: 1,
			}
			info.lastFail.Store(time.Now().UTC())
			info.blockTime.Store(time.Second * 10)

			blocklist.Store(ip, info)
		} else {
			info := i.(*blockinfo)
			atomic.AddInt64(&info.failCount, 1)
			info.lastFail.Store(time.Now().UTC())
		}
	}()

	// Checking credentials.
	ok := check(lo.Username, lo.Password)
	if !ok {
		err = errUnauthorized
		return
	}

	// Prepare login jwt token.
	now := time.Now().UTC()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Id:        uuid.Must(uuid.NewShort()),
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		ExpiresAt: now.Add(60 * 24 * time.Hour).Unix(),
		Audience:  lo.Username,
		Issuer:    "login.changkun.de",
		Subject:   "login",
	}).SignedString(hmacSecret)
	if err != nil {
		err = fmt.Errorf("failed to create login token: %w", err)
		return
	}

	// The credentials are valid, jwt token is also ready. Now let's
	// determine where should we redirect the user to. We expect the
	// request also provide the redirect location, otherwise we use
	// https://changkun.de as default location.

	u, err := url.Parse(lo.Redirect)
	if err != nil || lo.Redirect == "" {
		log.Println("missing redirect, use changkun.de instead.")
		u = &url.URL{
			Scheme: "https",
			Host:   "changkun.de",
		}
	}

	// Set the cookie if possible.
	w.Header().Set("Set-Cookie", fmt.Sprintf("auth=%s; Domain=changkun.de; Path=/; Max-Age=%d; SameSite=Lax", token, 60*60*24*60)) // 3 months

	// And supply the token to the redirected location, so that we could
	// handle CORS cases if the auth is from a different domain.
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	log.Println("redirecting to:", u.String())

	b, _ = json.Marshal(struct {
		Redirect string `json:"redirect"`
		Token    string `json:"token"`
	}{
		Redirect: u.String(),
		Token:    token,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func verifyfunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Cache-Control", "max-age=0")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		return
	}

	var err error
	defer func() {
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}()
	if r.Method != http.MethodPost {
		err = errors.New("unsupported method")
		return
	}

	// We assume verifying token is provided in the request body.
	type body struct {
		Token string `json:"token"`
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed to read request body: %w", err)
		return
	}
	data := &body{}
	err = json.Unmarshal(b, data)
	if err != nil {
		err = fmt.Errorf("failed to parse request body: %w", err)
		return
	}

	// Parse the provided jwt token and see if it is valid for parse
	t, err := jwt.ParseWithClaims(data.Token, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return hmacSecret, nil
	})
	if err != nil {
		err = fmt.Errorf("parse token failed: %w", err)
		return
	}

	// Checking validity of the token.
	claims, ok := t.Claims.(*jwt.StandardClaims)
	if !ok {
		err = fmt.Errorf("unsupported claims format")
		return
	}
	if !t.Valid {
		err = fmt.Errorf("invalid claims: %w", claims.Valid())
		return
	}

	if !checkUser(claims.Audience) {
		err = fmt.Errorf("invalid username: %s", claims.Audience)
		return
	}

	// Everything is OK!
	b, _ = json.Marshal(struct {
		Username string `json:"username"`
	}{Username: claims.Audience})
	w.Write(b)
}

func homefunc(w http.ResponseWriter, r *http.Request) {
	redirAddr := r.URL.Query().Get("redirect")
	if redirAddr == "" {
		log.Println("missing redirect address, use changkun.de instead.")
		redirAddr = "https://changkun.de"
	}

	// Fast path:
	// Check if cookie contains auth already. If so, check the validity
	// of the auth cookie, if everything went OK, let's do the redirect
	// directly without showing the login interface.
	c, err := r.Cookie("auth")
	if err == nil {
		// We found previous authentication token, let's check if
		// this is already logined credentials.
		var t *jwt.Token
		t, err = jwt.ParseWithClaims(c.Value, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return hmacSecret, nil
		})
		if err == nil {
			// Checking validity of the token.
			claims, ok := t.Claims.(*jwt.StandardClaims)
			if ok && t.Valid && checkUser(claims.Audience) {
				uu, err := url.Parse(redirAddr)
				if err == nil {
					q := uu.Query()
					q.Set("token", c.Value)
					uu.RawQuery = q.Encode()
					http.Redirect(w, r, uu.String(), http.StatusTemporaryRedirect)
					return
				}
			}
		}
	}

	loginTmpl.Execute(w, nil)
}
func testfunc(w http.ResponseWriter, r *http.Request) { testTmpl.Execute(w, nil) }

var (
	//go:embed login.html
	loginFile string
	loginTmpl = template.Must(template.New("login").Parse(loginFile))

	//go:embed test.html
	testFile string
	testTmpl = template.Must(template.New("test").Parse(testFile))
)
