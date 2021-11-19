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
	"time"

	"changkun.de/x/login/internal/uuid"
	"github.com/golang-jwt/jwt"
)

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

		w.WriteHeader(http.StatusBadRequest)
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

	// Checking credentials.
	ok := a.check(lo.Username, lo.Password)
	if !ok {
		err = errors.New("incorrect username or password")
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
	w.Header().Set("Set-Cookie", fmt.Sprintf("auth=%s; Max-Age=%d", token, 60*60*24*60)) // 3 months

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

	if !a.checkUser(claims.Audience) {
		err = fmt.Errorf("invalid username: %s", claims.Audience)
		return
	}

	// Everything is OK!
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
			if ok && t.Valid && a.checkUser(claims.Audience) {
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
