package router

import (
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/Aaron-json/WsSession/internal/controllers"
	"github.com/go-chi/chi/v5"
)

func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/new-session/{sessionName}", controllers.CreateNewSession)
	r.Get("/join-session/{sessionID}", controllers.JoinSession)

	// dev routes
	if os.Getenv("ENV") != "production" {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/*", func(w http.ResponseWriter, r *http.Request) {
			profile := chi.URLParam(r, "*")
			pprof.Handler(profile).ServeHTTP(w, r)
		})
	}
	return r
}
