package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/seiortech/mahakam"
	"github.com/seiortech/mahakam/extensions"
)

type response struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	cache := extensions.NewMapCache()

	mux := http.NewServeMux()

	mux.HandleFunc("/cache/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		if c, ok := cache.Get(name); ok {
			log.Println("Cache hit for ", name)

			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(w).Encode(c); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}

			return
		} else {
			log.Println("Cache miss for ", name)

			data := response{
				Id:   int(time.Now().UnixMilli()),
				Name: name,
			}

			if err := cache.Set(name, data); err != nil {
				http.Error(w, "Failed to set cache", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(w).Encode(data); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		}

	})

	// cache using middleware
	mux.HandleFunc("/cache-middleware/{name}", extensions.CacheMiddleware(cache, func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		data := response{
			Id:   int(time.Now().UnixMilli()),
			Name: name,
		}

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}))

	if err := mahakam.NewServer(":8080", mux).ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
