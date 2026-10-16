package bootstrap

import (
	"fmt"
	"sync"

	auditrepo "meteorx/internal/modules/audit/repository"
	exportrepo "meteorx/internal/modules/export/repository"
	"meteorx/internal/modules/file"
	invitationrepo "meteorx/internal/modules/invitation/repository"
	notificationrepo "meteorx/internal/modules/notification/repository"
	oauthrepo "meteorx/internal/modules/oauth/repository"
	planrepo "meteorx/internal/modules/plan/repository"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	taskrepo "meteorx/internal/modules/task/repository"
	tenantrepo "meteorx/internal/modules/tenant/repository"
	authrepo "meteorx/internal/modules/user/repository"
	wikirepo "meteorx/internal/modules/wiki/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// OrphanColumnReport 描述一张表上「数据库实际存在、但当前模型已不再定义」的孤儿列。
// AutoMigrate 只增不删，这类残留列不会被自动清理，仅供人工确认后手动 DROP。
type OrphanColumnReport struct {
	Table         string   // 表名
	OrphanColumns []string // 模型已删除、库里仍残留的列名
}

// migrationModels 汇总全部模块待迁移模型，作为脏列巡检的枚举来源。
// 各模块的 Models() 即其 AutoMigrate 迁移的模型清单；新增/删除表时随其一并更新，无需改本函数。
func migrationModels() []interface{} {
	var out []interface{}
	out = append(out, authrepo.Models()...)         // users
	out = append(out, tenantrepo.Models()...)       // tenants / tenant_settings / cancel_requests
	out = append(out, planrepo.Models()...)         // plans / tenant_subscriptions
	out = append(out, rbacrepo.Models()...)         // roles / permissions / role_permissions / user_roles
	out = append(out, auditrepo.Models()...)        // audit_logs / alert_rules / audit_alerts
	out = append(out, file.Models()...)             // files
	out = append(out, notificationrepo.Models()...) // announcements
	out = append(out, invitationrepo.Models()...)   // invitations
	out = append(out, oauthrepo.Models()...)        // oauth_accounts / oauth_providers / oauth_provider_meta
	out = append(out, taskrepo.Models()...)         // tasks
	out = append(out, wikirepo.Models()...)         // wiki_* 等 18 张表
	out = append(out, exportrepo.Models()...)       // export_jobs
	return out
}

// AuditOrphanColumns 只读巡检：逐表比对「当前模型定义列」与「数据库实际列」，
// 报出模型已删除但库里仍残留的孤儿列（AutoMigrate 只加不删，需人工确认后清理）。
//
// 全程仅执行 SELECT，不做任何 DDL 或写操作；库中尚不存在的表（如全新环境未迁移）
// 查询返回空列，直接跳过。返回按表聚合的孤儿列报告，无孤儿列时返回空切片。
func AuditOrphanColumns(db *gorm.DB) ([]OrphanColumnReport, error) {
	return auditOrphanColumns(db, migrationModels())
}

// auditOrphanColumns 是接受显式模型列表的内部实现，便于单测注入少量模型。
func auditOrphanColumns(db *gorm.DB, models []interface{}) ([]OrphanColumnReport, error) {
	var reports []OrphanColumnReport

	// 独立的 cacheStore，避免污染正常启动路径的 GORM 模型缓存。
	// 期望列名直接来自模型结构体的 schema 解析，永远与当前代码一致（不会漂移）。
	cacheStore := &sync.Map{}
	naming := schema.NamingStrategy{}

	for _, m := range models {
		s, err := schema.Parse(m, cacheStore, naming)
		if err != nil {
			return nil, fmt.Errorf("解析模型 %T 失败: %w", m, err)
		}

		// MySQL 专有：information_schema 反映实际列。若换库需改用对应元数据源。
		var actual []string
		if err := db.Table("information_schema.COLUMNS").
			Where("TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", s.Table).
			Pluck("COLUMN_NAME", &actual).Error; err != nil {
			return nil, fmt.Errorf("查询表 %s 列信息失败: %w", s.Table, err)
		}
		if len(actual) == 0 {
			continue // 表尚未创建，无孤儿列可言
		}

		expected := make(map[string]struct{}, len(s.FieldsByDBName))
		for col := range s.FieldsByDBName {
			expected[col] = struct{}{}
		}

		if orphans := orphanColumns(actual, expected); len(orphans) > 0 {
			reports = append(reports, OrphanColumnReport{Table: s.Table, OrphanColumns: orphans})
		}
	}

	return reports, nil
}

// orphanColumns 返回出现在 actual 但不在 expected 集合中的列，保持数据库返回顺序。
func orphanColumns(actual []string, expected map[string]struct{}) []string {
	var orphans []string
	for _, c := range actual {
		if _, ok := expected[c]; !ok {
			orphans = append(orphans, c)
		}
	}
	return orphans
}
