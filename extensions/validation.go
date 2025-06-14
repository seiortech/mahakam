package extensions

import (
	"context"
	"encoding/json"
	"net/http"
)

type contextKey string

const (
	BodyKey contextKey = "body"
)

type ValidationError struct {
	Message string         `json:"message"`
	Code    int            `json:"code"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func (e ValidationError) Error() string {
	return e.Message
}

func (e ValidationError) JSON() ([]byte, error) {
	return json.Marshal(e)
}

type Validation interface {
	Validate() error
}

func ValidationMiddleware[T Validation](next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data T
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			panic(err)
		}

		if err := data.Validate(); err != nil {
			panic(err)
		}

		next(w, r.WithContext(context.WithValue(r.Context(), BodyKey, data)))
	}
}
