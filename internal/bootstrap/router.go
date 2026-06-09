package bootstrap

import (
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/auth"
	"meteorx/internal/modules/tenant"
	"meteorx/internal/modules/user"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func InitRouter(db *gorm.DB, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	SetupMiddleware(r)

	tokenHelper := jwt.NewTokenHelper(cfg.JWT)

	// 基础检查
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {

		// --- 分组一：公开接口 (Public) ---
		r.Group(func(r chi.Router) {
			// 1. 认证模块（登录、签发 Token）
			auth.InitModule(r, db, cfg.JWT)

			// 2. 租户公开接口（仅限注册）
			tenant.InitPublicModule(r, db)
		})

		// --- 分组二：受保护接口 (Protected) ---
		r.Group(func(r chi.Router) {
			// 【第一层防线】挂载认证中间件，解析 Token 并注入 UserID, TenantID, Role
			r.Use(middleware.Auth(tokenHelper))

			// 3. 租户私有接口（租户管理员登录后：管理本公司信息、查看套餐等）
			tenant.InitPrivateModule(r, db)

			// 4. 用户/业务接口
			user.InitModule(r, db)

			// ========================================================
			// 🔥 新增分组三：MaaS 平台运营后台特权接口 (Platform Admin Only)
			// ========================================================
			r.Group(func(r chi.Router) {
				// 【第二层防线】剥洋葱：在已登录的基础上，必须是平台超级管理员 (role == "superadmin")
				r.Use(middleware.RequiresMasterAdmin())

				// 5. 租户管理特权接口（后台手动新建租户、禁用/启用、软删除等）
				tenant.InitAdminModule(r, db)

				// 6. 系统管理员管理接口
				user.InitAdminModule(r, db)
			})
		})
	})

	return r
}