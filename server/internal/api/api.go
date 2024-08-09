package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kevinfjq/go-react-ask-app/server/internal/store/pgstore"
	"net/http"
)

type apiHandler struct {
	q *pgstore.Queries
	r *chi.Mux
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q: q,
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger)

	r.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {

		})
	})
	a.r = r
	return a
}
