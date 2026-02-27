package gateway

import (
	"log"
	"net/http"
)

func StartHTTP() {
	http.HandleFunc("/ws", wsHandler)

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	log.Println("HTTP on :8080")
	http.ListenAndServe(":8080", nil)
}
