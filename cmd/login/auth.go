// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package main

import (
	"log"
	"os"
)

var (
	loginUsername string
	loginPassword string
)

func init() {
	loginUsername = os.Getenv("LOGIN_USERNAME")
	if loginUsername == "" {
		log.Fatal("LOGIN_USERNAME is required")
	}
	loginPassword = os.Getenv("LOGIN_PASSWORD")
	if loginPassword == "" {
		log.Fatal("LOGIN_PASSWORD is required")
	}
}

func check(u, p string) bool {
	if u == "" || p == "" {
		return false
	}
	return u == loginUsername && p == loginPassword
}

func checkUser(u string) bool {
	return u == loginUsername
}
