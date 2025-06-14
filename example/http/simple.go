package main

import (
	"log"
	"net/http"

	"github.com/seiortech/mahakam"
	"github.com/seiortech/mahakam/extensions"
	"github.com/seiortech/mahakam/middleware"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	errorMap := make(map[string]any)
	if r.Email == "" {
		errorMap["email"] = "Email is required"
	}

	if r.Password == "" {
		errorMap["password"] = "Password is required"
	}

	if len(errorMap) > 0 {
		return extensions.ValidationError{
			Message: "Invalid request body",
			Code:    http.StatusBadRequest,
			Fields:  errorMap,
		}
	}

	return nil
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello, JSON!"}`))
	})

	mux.HandleFunc("POST /body", extensions.ValidationMiddleware[LoginRequest](func(w http.ResponseWriter, r *http.Request) {
		body, ok := r.Context().Value(extensions.BodyKey).(LoginRequest)
		if !ok {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Received body successfully", "email": "` + body.Email + `"}`))
	}))

	s := mahakam.NewServer("localhost:8080", mux)
	s.Use(middleware.Logger)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalln("Failed to start server:", err)
	}
}
