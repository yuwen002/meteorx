package bootstrap

import (
	"net/http"

	"meteorx/internal/modules/auth"

	"github.com/go-chi/chi/v5"
)

func (a *Application) registerRoutes() {

	// health check
	a.Router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// auth module
	a.Router.Mount("/auth", auth.Routes(a.DB))
}

func httpListen(r *chi.Mux) error {
	return http.ListenAndServe(":8080", r)
}
