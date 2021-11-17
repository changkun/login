// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"go.etcd.io/bbolt"
)

var a *accounts

func init() {
	var err error
	a, err = newDB()
	if err != nil {
		log.Fatal(err)
	}
}

type accounts struct {
	db *bbolt.DB
}

func newDB() (*accounts, error) {
	a := &accounts{}

	addr := os.Getenv("LOGIN_DB")
	if addr == "" {
		addr = "db/login.db"
	}

	var err error
	a.db, err = bbolt.Open(addr, 0666, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("cannot open login.db: %w", err)
	}

	return a, nil
}

func (a *accounts) check(u, p string) bool {
	if u == "" || p == "" {
		return false
	}

	var pb string
	err := a.db.View(func(t *bbolt.Tx) error {
		pb = string(t.Bucket([]byte("accounts")).Get([]byte(u)))
		return nil
	})
	if err != nil {
		return false
	}

	return pb == p
}

func (a accounts) checkUser(u string) bool {
	var pb string
	err := a.db.View(func(t *bbolt.Tx) error {
		pb = string(t.Bucket([]byte("accounts")).Get([]byte(u)))
		return nil
	})
	if err != nil {
		return false
	}

	return pb != ""
}
