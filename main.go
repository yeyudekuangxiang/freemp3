package main

import (
	"net/http"
)

func main() {
	http.ListenAndServe(":8022", http.FileServer(http.Dir("./")))
}
