// Package routetable 提供与生产 bootstrap.InitRouter 同构的空 handler 路由表。
//
// 背景：OpenAPI/Apifox 同步脚本与路由测试都需要一份"不带 DB 也能构造"的路由集合，
// 过去这些集合由各自手写的镜像维护（chat/task/wiki/file 逐个 r.Get 复制），
// 结果镜像与真实模块反复漂移——模块新增了 /tasks/ai/*，镜像没跟上，文档检查于是长期"假绿"。
//
// 现在路由统一从各模块导出的纯注册函数（RegisterRoutes 等）拼装，只有仍与 DB/存储强绑定的
// file 模块保留最小的手写镜像。镜像与生产路由是否一致由 bootstrap 包的
// TestRouteTable_MatchesProductionRouter 用真实 sqlite + miniredis 启动完整路由后强校验，
// 漂移会在单测阶段直接失败，而不是等到文档缺接口才被发现。
package routetable

import (
	"net/http"

	"meteorx/internal/modules/audit"
	auditHandler "meteorx/internal/modules/audit/handler"
	"meteorx/internal/modules/auth"
	authHandler "meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/chat"
	chatHandler "meteorx/internal/modules/chat/handler"
	"meteorx/internal/modules/dashboard"
	dashboardHandler "meteorx/internal/modules/dashboard/handler"
	"meteorx/internal/modules/export"
	exportHandler "meteorx/internal/modules/export/handler"
	"meteorx/internal/modules/invitation"
	invitationHandler "meteorx/internal/modules/invitation/handler"
	"meteorx/internal/modules/message"
	messageHandler "meteorx/internal/modules/message/handler"
	"meteorx/internal/modules/notification"
	notificationHandler "meteorx/internal/modules/notification/handler"
	"meteorx/internal/modules/oauth"
	oauthHandler "meteorx/internal/modules/oauth/handler"
	"meteorx/internal/modules/plan"
	planHandler "meteorx/internal/modules/plan/handler"
	"meteorx/internal/modules/rbac"
	rbacHandler "meteorx/internal/modules/rbac/handler"
	"meteorx/internal/modules/settings"
	settingsHandler "meteorx/internal/modules/settings/handler"
	"meteorx/internal/modules/task"
	taskHandler "meteorx/internal/modules/task/handler"
	"meteorx/internal/modules/tenant"
	tenantHandler "meteorx/internal/modules/tenant/handler"
	"meteorx/internal/modules/user"
	userHandler "meteorx/internal/modules/user/handler"
	"meteorx/internal/modules/wiki"
	wikiHandler "meteorx/internal/modules/wiki/handler"

	"github.com/go-chi/chi/v5"
)

// Build 返回与生产一致的路由树（handler 全为空实现，仅供 chi.Walk 提取路由）。
// 分组划分、模块顺序必须与 internal/bootstrap/router.go 保持一致。
func Build() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", noop())
	r.Get("/health/ready", noop())
	r.Get("/metrics", noop())

	r.Route("/api/v1", func(r chi.Router) {
		// --- 公开接口 ---
		r.Group(func(r chi.Router) {
			r.Post("/csp-report", noop())
			auth.RegisterRoutes(r, &authHandler.AuthHandler{})
			tenant.RegisterPublicRoutes(r, &tenantHandler.TenantHandler{})
			oauth.RegisterRoutes(r, &oauthHandler.OAuthHandler{})
			invitation.RegisterPublicRoutes(r, &invitationHandler.InvitationHandler{})
			settings.RegisterPublicRoutes(r, &settingsHandler.SettingHandler{})
			wiki.RegisterPublicShareRoutes(r, &wikiHandler.WikiHandlerExtended{})
		})

		// --- 受保护接口 ---
		r.Group(func(r chi.Router) {
			r.Get("/ws", noop())

			tenant.RegisterPrivateRoutes(r, &tenantHandler.TenantHandler{})
			tenant.RegisterTenantSettingsRoutes(r, &tenantHandler.TenantSettingsHandler{})
			user.RegisterRoutes(r, &userHandler.UserHandler{}, nil)
			user.RegisterProfileRoutes(r, &userHandler.UserHandler{})
			oauth.RegisterProtectedRoutes(r, &oauthHandler.OAuthHandler{})
			auth.RegisterAPITokenRoutes(r, &authHandler.APITokenHandler{})
			auth.RegisterEmailVerificationRoutes(r, &authHandler.AuthHandler{})
			auth.RegisterTwoFactorRoutes(r, &authHandler.AuthHandler{})
			plan.RegisterPrivateRoutes(r, &planHandler.PlanHandler{})
			registerFileRoutes(r)
			wiki.RegisterRoutes(r, &wikiHandler.WikiHandler{}, &wikiHandler.WikiHandlerExtended{})
			notification.RegisterTenantRoutes(r, &notificationHandler.AnnouncementHandler{})
			invitation.RegisterRoutes(r, &invitationHandler.InvitationHandler{}, nil)
			task.RegisterRoutes(r, &taskHandler.TaskHandler{}, &taskHandler.TaskAIHandler{})
			export.RegisterRoutes(r, &exportHandler.ExportHandler{}, nil)
			message.RegisterRoutes(r, &messageHandler.MessageHandler{})
			chat.RegisterRoutes(r, &chatHandler.ChatHandler{})
			dashboard.RegisterTenantRoutes(r, &dashboardHandler.DashboardHandler{})

			// --- 平台管理员接口 ---
			r.Group(func(r chi.Router) {
				tenant.RegisterAdminRoutes(r, &tenantHandler.TenantHandler{}, nil)
				user.RegisterAdminRoutes(r, &userHandler.UserHandler{}, nil)
				rbac.RegisterRoutes(r, &rbacHandler.RBACHandler{}, nil)
				audit.RegisterRoutes(r, &auditHandler.AuditHandler{}, &auditHandler.AlertHandler{}, &auditHandler.SessionHandler{}, &auditHandler.IntegrityHandler{}, nil, nil)
				plan.RegisterAdminRoutes(r, &planHandler.PlanHandler{}, nil)
				dashboard.RegisterRoutes(r, &dashboardHandler.DashboardHandler{}, nil)
				notification.RegisterRoutes(r, &notificationHandler.AnnouncementHandler{}, nil)
				oauth.RegisterAdminRoutes(r, &oauthHandler.OAuthProviderAdminHandler{}, nil)
				settings.RegisterAdminRoutes(r, &settingsHandler.SettingHandler{}, nil)
			})
		})
	})

	return r
}

// registerFileRoutes 手写镜像：file.RegisterRoutes 在注册期即构造存储后端与配额服务，
// 依赖真实 db/cfg，无法在空 handler 场景复用；路由清单变更时需同步此处（对齐测试会兜住）。
func registerFileRoutes(r chi.Router) {
	r.Route("/files", func(r chi.Router) {
		r.Post("/upload", noop())
		r.Get("/", noop())
		r.Get("/my", noop())
		r.Get("/{id}", noop())
		r.Get("/{id}/download", noop())
		r.Put("/{id}", noop())
		r.Delete("/{id}", noop())
		r.Post("/batch/delete", noop())
		r.Get("/deleted", noop())
		r.Put("/{id}/restore", noop())
		r.Delete("/{id}/permanent", noop())
	})
}

func noop() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
