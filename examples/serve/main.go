package main

import (
	"net/http"
)

func main() {
	http.Handle("/", &noCache{Handler: http.FileServer(http.Dir("."))})
	http.ListenAndServe(":8080", nil)
}

type noCache struct {
	http.Handler
}

func (h *noCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	h.Handler.ServeHTTP(w, r)
}
