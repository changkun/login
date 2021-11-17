// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

// This program initializes a bbolt database for the void.
package main

import (
	"fmt"

	"go.etcd.io/bbolt"
)

func main() {
	db, err := bbolt.Open("login.db", 0666, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("accounts"))
		if err != nil {
			return fmt.Errorf("cannot create bucket: %s", err)
		}

		b.Put([]byte("changkun"), []byte("changkun"))
		return nil
	})
}
