// Copyright (c) 2021 Changkun Ou <hi@changkun.de>. All Rights Reserved.
// Unauthorized using, copying, modifying and distributing, via any
// medium is strictly prohibited.

package main

import (
	"errors"
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetPrefix("login: ")
	log.SetFlags(0)

	http.Handle("/", logging(http.HandlerFunc(homefunc)))
	http.Handle("/auth", logging(http.HandlerFunc(authfunc)))
	http.Handle("/verify", logging(http.HandlerFunc(verifyfunc)))
	http.Handle("/test", logging(http.HandlerFunc(testfunc)))
	http.Handle("/sdk.js", logging(http.HandlerFunc(sdkfunc)))

	port := os.Getenv("LOGIN_PORT")
	if port == "" {
		port = ":8080"
	}
	log.Printf("serving at %s...\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server is closed with error: %s", err)
	}
	log.Println("login server is down, bye!")
}
