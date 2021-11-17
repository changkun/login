// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package login_test

import (
	"testing"

	"changkun.de/x/login"
)

func TestLogin(t *testing.T) {
	t.Skip("Run this test manually by using a valid JWT.")
	err := login.Check("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJjaGFuZ2t1biIsImV4cCI6MTY0MjUyMDYwMiwianRpIjoiekttd3BWMjljS2thaUFjU0p5OGZnSyIsImlhdCI6MTYzNzMzNjYwMiwiaXNzIjoibG9naW4uY2hhbmdrdW4uZGUiLCJuYmYiOjE2MzczMzY2MDIsInN1YiI6ImxvZ2luIn0.PpRZRph9inNHSGevAJ4G-RSw-rwvjRXMufusBUYtW30")
	if err != nil {
		t.Fatalf("expect to be valid, but failed: %v", err)
	}
}
