// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package login

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

// Endpoint is the login verify endpoint.
var Endpoint = "https://login.changkun.de/verify"

var (
	ErrBadRequest   = errors.New("bad request")
	ErrLoginInvalid = errors.New("invalid login")
)

// Check checks if the given login token is valid or not.
func Check(token string) error {
	b, _ := json.Marshal(struct {
		Token string `json:"token"`
	}{
		Token: token,
	})
	br := bytes.NewReader(b)

	resp, err := http.DefaultClient.Post(Endpoint, "application/json", br)
	if err != nil {
		return ErrBadRequest
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ErrLoginInvalid
	}

	return nil
}
