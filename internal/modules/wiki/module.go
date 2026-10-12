// Package wiki 提供知识库模块，支持知识空间、文档树、Markdown 编辑/渲染、版本管理、权限控制及全文搜索。
package wiki

import (
	"meteorx/internal/cache"
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
// rdb 可为 nil：仅用于回收站过期清理的跨副本分布式锁，缺失时降级为进程内互斥。
func InitModule(r chi.Router, gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config, rdb *cache.Redis) {
	h, extH := initHandlers(gormDB, tx, cfg, rdb)
	RegisterRoutes(r, h, extH)
}

// initHandlers 装配 Wiki 模块所有处理器，返回主处理器和扩展处理器。
func initHandlers(gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config, rdb *cache.Redis) (*handler.WikiHandler, *handler.WikiHandlerExtended) {
	// 统一使用扩展仓储：它继承 WikiRepository，且额外实现订阅/通知等扩展方法。
	// 主服务内部 notifySubscribers 依赖 WikiRepositoryExtended 类型断言，
	// 若注入 NewWikiRepository 的基础实现会使文档编辑通知静默失效。
	extRepo := repository.NewWikiRepositoryExtended(gormDB)
	svc := service.NewWikiService(extRepo, tx, service.WithExpireCacheLock(rdb))

	searchCfg := search.Config{
		Provider:    cfg.Search.Provider,
		Host:        cfg.Search.Host,
		APIKey:      cfg.Search.APIKey,
		IndexPrefix: cfg.Search.IndexPrefix,
	}
	engine := search.NewEngine(searchCfg)
	indexer := search.NewWikiIndexer(engine, extRepo, extRepo)
	svc.SetWikiIndexer(indexer)

	if cfg != nil {
		svc.SetUploadSigner(cfg.File.UploadURL, cfg.File.UploadSignKey(cfg.JWT.Secret))
	}
	h := handler.NewWikiHandler(svc)

	extSvc := service.NewWikiServiceExtended(extRepo, tx)
	extH := handler.NewWikiHandlerExtended(extSvc)

	return h, extH
}