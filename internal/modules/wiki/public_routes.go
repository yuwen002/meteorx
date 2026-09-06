package wiki

import (
	"meteorx/internal/config"
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"

	db "meteorx/internal/pkg/db"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// RegisterPublicShareRoute 注册免登录的 Wiki 文档分享读取接口（公开分组内调用）。
// 分享数据通过 token+可选密码访问，不依赖登录态与租户上下文；
// 文档内嵌图片会在响应时动态重签短时效签名地址。
func RegisterPublicShareRoute(r chi.Router, gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config) {
	extRepo := repository.NewWikiRepositoryExtended(gormDB)
	extSvc := service.NewWikiServiceExtended(extRepo, tx)
	if cfg != nil {
		extSvc.SetUploadSigner(cfg.File.UploadURL, cfg.JWT.Secret)
	}
	extH := handler.NewWikiHandlerExtended(extSvc)

	// GET /api/v1/wiki/share/{token}?password=xxx
	r.Get("/wiki/share/{token}", extH.AccessSharedDocument)
}
