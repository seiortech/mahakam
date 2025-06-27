package main

import (
	"net/http"

	"github.com/seiortech/mahakam"
	"github.com/seiortech/mahakam/middleware"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from /foo"))
	})

	mux.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from /bar"))
	})

	cors := middleware.NewCORSMiddleware(&middleware.DefaultCORSMiddlewareOption)
	cors.AddOrigin("localhost:3000", "mohamadrishwan.me").AddMethod("GET", "POST")

	server := mahakam.NewServer(":8080", mux)
	server.Use(middleware.Logger, cors.Middleware)
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
