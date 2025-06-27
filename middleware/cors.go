package middleware

import (
	"net/http"
	"strings"
)

// CORSOption defines the configuration options for CORS middleware.
type CORSOption struct {
	AllowOrigin      []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	MaxAge           string
	AllowCredentials bool
	AllowAllOrigins  bool
}

// DefaultCORSOption provides default values for CORS options.
// Don't use this for production, as it allows all origins and methods.
var DefaultCORSMiddlewareOption = CORSOption{
	AllowOrigin:      []string{"*"},
	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
	ExposeHeaders:    []string{"Content-Length"},
	MaxAge:           "3600",
	AllowCredentials: false,
	AllowAllOrigins:  true,
}

// CORS is a middleware that handles Cross-Origin Resource Sharing (CORS) headers.
type CORS struct {
	*CORSOption
}

// NewCORSOption creates a new CORSOption with the provided parameters. if option is nil, it uses the default options.
// Don't use default options for production, as it allows all origins and methods.
func NewCORSMiddleware(option *CORSOption) *CORS {
	if option == nil {
		option = &DefaultCORSMiddlewareOption
	}

	return &CORS{
		CORSOption: option,
	}
}

func (c *CORS) isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range c.AllowOrigin {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

func (c *CORS) validateAndSetOrigin(w http.ResponseWriter, origin string) bool {
	if origin == "" {
		return true
	}

	if c.AllowAllOrigins && !c.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return true
	}

	if c.isOriginAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		return true
	}

	return false
}

// handlePreflight processes preflight OPTIONS requests
func (c *CORS) handlePreflight(w http.ResponseWriter, r *http.Request) {
	reqMethod := r.Header.Get("Access-Control-Request-Method")
	methodAllowed := false
	for _, method := range c.AllowMethods {
		if method == reqMethod {
			methodAllowed = true
			break
		}
	}

	reqHeaders := r.Header.Get("Access-Control-Request-Headers")
	headersAllowed := true
	if reqHeaders != "" {
		requestedHeaders := strings.Split(reqHeaders, ",")
		for _, header := range requestedHeaders {
			header = strings.TrimSpace(header)
			headerAllowed := false
			for _, allowedHeader := range c.AllowHeaders {
				if strings.EqualFold(allowedHeader, header) {
					headerAllowed = true
					break
				}
			}
			if !headerAllowed {
				headersAllowed = false
				break
			}
		}
	}

	if !methodAllowed || !headersAllowed {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.AllowMethods, ", "))
	if len(c.AllowHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.AllowHeaders, ", "))
	}
	if c.MaxAge != "" {
		w.Header().Set("Access-Control-Max-Age", c.MaxAge)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Middleware returns an HTTP middleware function that handles CORS headers
func (c *CORS) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		isPreflight := r.Method == "OPTIONS" && origin != "" &&
			r.Header.Get("Access-Control-Request-Method") != ""

		if !c.validateAndSetOrigin(w, origin) && origin != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if isPreflight {
			c.handlePreflight(w, r)
			return
		}

		if origin != "" {
			if len(c.ExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.ExposeHeaders, ", "))
			}

			if c.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		next(w, r)
	}
}

// / AddOrigin adds a specific origin to the allowed origins list.
func (c *CORS) AddOrigin(origins ...string) *CORS {
	for _, origin := range origins {
		if !c.isOriginAllowed(origin) {
			c.AllowOrigin = append(c.AllowOrigin, origin)
		}
	}

	return c
}

// AddMethod adds a specific method to the allowed methods list.
func (c *CORS) AddMethod(methods ...string) *CORS {
	for _, method := range methods {
		methodExists := false
		for _, allowedMethod := range c.AllowMethods {
			if allowedMethod == method {
				methodExists = true
				break
			}
		}

		if !methodExists {
			c.AllowMethods = append(c.AllowMethods, method)
		}
	}

	return c
}

// AddHeader adds a specific header to the allowed headers list.
func (c *CORS) AddHeader(headers ...string) *CORS {
	for _, header := range headers {
		headerExists := false
		for _, allowedHeader := range c.AllowHeaders {
			if strings.EqualFold(allowedHeader, header) {
				headerExists = true
				break
			}
		}

		if !headerExists {
			c.AllowHeaders = append(c.AllowHeaders, header)
		}
	}

	return c
}

// CORSMiddleware returns an HTTP middleware function that handles CORS headers with default options.
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	cors := NewCORSMiddleware(nil)
	return cors.Middleware(next)
}
