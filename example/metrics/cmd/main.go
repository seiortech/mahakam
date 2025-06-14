package main

import (
	"log"
	"net/http"

	"github.com/seiortech/mahakam"
	"github.com/seiortech/mahakam/extensions"
	"github.com/seiortech/mahakam/middleware"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("foo"))
	})

	mux.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bar"))
	})

	metrics := extensions.NewMetrics()
	if err := metrics.Register(mux); err != nil {
		log.Fatalln("Failed to register metrics:", err)
	}

	metrics.StartUptimeTracking()

	s := mahakam.NewServer("0.0.0.0:8080", mux)
	s.Use(middleware.Logger)
	s.Use(metrics.Middleware)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalln("Failed to start server:", err)
	}
}
