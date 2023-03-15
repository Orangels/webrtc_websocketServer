// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"net/http"

	"golang.org/x/exp/slog"
)

var addr = flag.String("addr", ":8989", "http service address")
var certFile = flag.String("cert", "/home/ubuntu/cert/cert.pem", "cert file")
var keyFile = flag.String("key", "/home/ubuntu/cert/key.pem", "key file")

func main() {
	flag.Parse()

	mgr := GetManager()
	mgr.Run()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	
	http.HandleFunc("/webrtc", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Upgrade to websocket failed!", err)
			return
		}

		channel := NewChannel(conn, mgr)
		channel.Startup()
	})

	slog.Info(fmt.Sprintf("WebRTC Server started, listening on %s...", *addr))
	err := http.ListenAndServeTLS(*addr, *certFile, *keyFile, nil)
	if err != nil {
		slog.Error("ListenAndServeTLS failed!", err)
	}
}
