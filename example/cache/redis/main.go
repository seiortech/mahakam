package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seiortech/mahakam"
	"github.com/seiortech/mahakam/extensions"
)

type response struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	cache, err := extensions.NewRedisCache(redisClient)
	if err != nil {
		log.Fatalf("Failed to create Redis cache: %v", err)
	}

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
		}

		log.Println("Cache miss for", name)
		data := response{
			Id:   int(time.Now().UnixMilli()),
			Name: name,
		}

		var cacheData []byte
		if cacheData, err = json.Marshal(data); err != nil {
			log.Println("Failed to marshal data:", err)
			http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
			return
		}

		if err := cache.Set(name, cacheData); err != nil {
			log.Println("Failed to set cache:", err)
			http.Error(w, "Failed to set cache", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
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
