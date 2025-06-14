package extensions

import (
	"context"
	"encoding/json"
	"net/http"
)

type contextKey string

const (
	BodyKey contextKey = "body" // BodyKey is the context key used to store the body of the request.
)

type ValidationError struct {
	Message string         `json:"message"`
	Code    int            `json:"code"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new ValidationError with the given message and code.
func (e ValidationError) JSON() ([]byte, error) {
	return json.Marshal(e)
}

// Validation is an interface that requires a Validate method for validating request data using ValidationMiddleware.
type Validation interface {
	Validate() error
}

// ValidationMiddleware is a middleware that validates the request body against the Validation interface.
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
