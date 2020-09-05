package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	http.Handle("/upload/", &uploader{})
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":8080", nil)
}

type uploader struct{}

func (*uploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("receiving upload")
	if r.Method != "PUT" {
		log.Println(r.Method, "denied")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	f, err := os.Create("data/saved.pcd")
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	io.Copy(f, r.Body)
	log.Println("uploaded")
}
