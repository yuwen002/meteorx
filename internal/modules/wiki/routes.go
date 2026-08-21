package wiki

import (
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func InitModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewWikiRepository(db)
	svc := service.NewWikiService(repo)
	h := handler.NewWikiHandler(svc)

	r.Route("/wiki", func(r chi.Router) {
		r.Get("/stats", h.GetStats)

		r.Route("/spaces", func(r chi.Router) {
			r.Get("/", h.ListSpaces)
			r.Post("/", h.CreateSpace)
			r.Get("/{id}", h.GetSpace)
			r.Put("/{id}", h.UpdateSpace)
			r.Delete("/{id}", h.DeleteSpace)

			r.Route("/{spaceId}/nodes", func(r chi.Router) {
				r.Get("/tree", h.GetNodeTree)
				r.Post("/", h.CreateNode)
				r.Get("/{id}", h.GetNode)
				r.Put("/{id}", h.UpdateNode)
				r.Delete("/{id}", h.DeleteNode)
			})

			r.Route("/{spaceId}/members", func(r chi.Router) {
				r.Get("/", h.ListMembers)
				r.Post("/", h.AddMember)
				r.Delete("/{userId}", h.RemoveMember)
			})
		})

		r.Route("/documents", func(r chi.Router) {
			r.Post("/nodes/{nodeId}", h.CreateDocument)
			r.Get("/nodes/{nodeId}", h.GetDocument)
			r.Put("/{id}", h.UpdateDocument)
			r.Delete("/{id}", h.DeleteDocument)

			r.Get("/{documentId}/revisions", h.ListRevisions)
			r.Get("/{documentId}/revisions/{version}", h.GetRevision)
			r.Post("/{documentId}/revisions/{version}/restore", h.RestoreRevision)
		})
	})
}