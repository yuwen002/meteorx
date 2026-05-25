package tenant

import (
	"github.com/go-chi/chi/v5"
	"meteorx/internal/modules/tenant/handler"
)

func Routes(h *handler.TenantHandler) chi.Router {
	r := chi.NewRouter()

	// 租户入驻，通常是公开接口
	r.Post("/register", h.Register)

	// 其他需要管理的接口
	r.Route("/{id}", func(r chi.Router) {
		// 这里未来可以加中间件：r.Use(auth.AdminOnly)
		r.Get("/", h.GetByID)
	})

	return r
}
