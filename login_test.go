// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package login_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"changkun.de/x/login"
)

func TestLogin(t *testing.T) {
	t.Skip("Run this test manually by using a valid JWT.")

	_, err := login.Verify("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJjaGFuZ2t1biIsImV4cCI6MTY0MjUyMDYwMiwianRpIjoiekttd3BWMjljS2thaUFjU0p5OGZnSyIsImlhdCI6MTYzNzMzNjYwMiwiaXNzIjoibG9naW4uY2hhbmdrdW4uZGUiLCJuYmYiOjE2MzczMzY2MDIsInN1YiI6ImxvZ2luIn0.PpRZRph9inNHSGevAJ4G-RSw-rwvjRXMufusBUYtW30")
	if err != nil {
		t.Fatalf("expect to be valid, but failed: %v", err)
	}
}

func TestLogin2(t *testing.T) {
	t.Skip("Run this test manually by using a valid credentials.")

	token, err := login.RequestToken("changkun", "changkun")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/"+"?token="+token, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	u, err := login.HandleAuth(rr, req)
	if err != nil {
		t.Fatal(err)
	}
	if u != "changkun" {
		t.Fatalf("user is not changkun!")
	}

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
