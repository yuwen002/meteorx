package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/admin", func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, req *http.Request) {
					rc := chi.RouteContext(req.Context())
					fmt.Println("RoutePatterns:", rc.RoutePatterns)
					pattern := rc.RoutePatterns[len(rc.RoutePatterns)-1]
					fmt.Println("Last pattern:", pattern)
					fmt.Println("Trimmed:", strings.TrimPrefix(pattern, "/api/v1/"))
				})
			})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}