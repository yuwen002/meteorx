// Package wiki 提供知识库模块，支持知识空间、文档树、Markdown 编辑/渲染、版本管理、权限控制及全文搜索。
package wiki

import (
	"meteorx/internal/config"
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"
	"meteorx/internal/search"

	db "meteorx/internal/pkg/db"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 装配 Wiki 依赖（仓储/服务/搜索引擎/处理器）并注册路由。
func InitModule(r chi.Router, gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config) {
	h, extH := initHandlers(gormDB, tx, cfg)
	RegisterRoutes(r, h, extH)
}

// initHandlers 装配 Wiki 模块所有处理器，返回主处理器和扩展处理器。
func initHandlers(gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config) (*handler.WikiHandler, *handler.WikiHandlerExtended) {
	repo := repository.NewWikiRepository(gormDB)
	svc := service.NewWikiService(repo, tx)

	searchCfg := search.Config{
		Provider:    cfg.Search.Provider,
		Host:        cfg.Search.Host,
		APIKey:      cfg.Search.APIKey,
		IndexPrefix: cfg.Search.IndexPrefix,
	}
	engine := search.NewEngine(searchCfg)
	indexer := search.NewWikiIndexer(engine, repo, repo)
	svc.SetWikiIndexer(indexer)

	if cfg != nil {
		svc.SetUploadSigner(cfg.File.UploadURL, cfg.File.UploadSignKey(cfg.JWT.Secret))
	}
	h := handler.NewWikiHandler(svc)

	extRepo := repository.NewWikiRepositoryExtended(gormDB)
	extSvc := service.NewWikiServiceExtended(extRepo, tx)
	extH := handler.NewWikiHandlerExtended(extSvc)

	return h, extH
}