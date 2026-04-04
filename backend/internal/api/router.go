package api

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// NewRouter creates the chi router with all API endpoints.
func NewRouter(h *Handler, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(CORS)
	r.Use(RequestLogger(logger))

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth
		r.Post("/auth/validate", h.ValidateAuth)

		// Repos
		r.Get("/repos", h.ListRepos)
		r.Get("/repos/{id}/settings", h.GetRepoSettings)
		r.Put("/repos/{id}/settings", h.UpdateRepoSettings)

		// Groups
		r.Get("/groups", h.ListGroups)
		r.Post("/groups", h.CreateGroup)
		r.Delete("/groups/{id}", h.DeleteGroup)
		r.Get("/groups/{id}/metrics", h.GetGroupMetrics)
		r.Get("/groups/{id}/trends", h.GetGroupTrends)
		r.Get("/groups/{id}/settings", h.GetGroupSettings)
		r.Put("/groups/{id}/settings", h.UpdateGroupSettings)
		r.Get("/groups/{id}/export", h.ExportCSV)
		r.Get("/groups/{id}/badge", h.GetBadge)
		r.Get("/groups/{id}/pulls", h.ListGroupPulls)

		// Sync
		r.Post("/groups/{id}/sync", h.StartSync)
		r.Get("/jobs/{id}", h.GetJob)
		r.Post("/jobs/{id}/cancel", h.CancelJob)
	})

	return r
}
