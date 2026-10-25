// E2E 数据夹具：浏览器测试需要的账号与直达数据库的断言/清理能力。
//
// 子命令：
//
//	create-admin        创建临时超管（is_master=true），输出凭据
//	delete-admin <用户名> 删除临时超管
//	verify-plan <code>  直接查 MySQL 验证套餐真实落库
//	delete-plan <code>  物理删除测试套餐
//	ensure-tenant       取一个可用租户；库里没有时新建一条，并以 created 标记归属
//	delete-tenant <id>  物理删除测试租户（仅删 ensure-tenant 返回 created=true 的那条）
//	seed-alert <username>  造一条审计告警（user_id 由夹具生成），供告警详情深链用例定位
//	delete-alert <id>   物理删除测试告警
//
// 约定：结果以单行 `E2E_JSON:<json>` 输出，便于测试侧在 GORM 日志中稳定解析。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"meteorx/internal/bootstrap"
	"meteorx/pkg/ulid"

	"golang.org/x/crypto/bcrypt"
)

const (
	e2ePassword = "E2e!Pass123"
	jsonPrefix  = "E2E_JSON:"
)

type adminResult struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ID       string `json:"id"`
}

type planResult struct {
	Found        bool    `json:"found"`
	ID           string  `json:"id,omitempty"`
	Name         string  `json:"name,omitempty"`
	Code         string  `json:"code,omitempty"`
	UserLimit    int     `json:"user_limit"`
	StorageLimit int64   `json:"storage_limit"`
	Price        float64 `json:"price"`
	Status       int     `json:"status"`
	CreatedAt    string  `json:"created_at,omitempty"`
}

type deleteResult struct {
	Deleted int64 `json:"deleted"`
}

type tenantRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Created 为 true 表示这条租户是本进程新建的，调用方负责在用例结束时删除
	Created bool `json:"created"`
}

type alertRef struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

func main() {
	if len(os.Args) < 2 {
		fatal("usage: fixture create-admin|delete-admin <username>|verify-plan <code>|delete-plan <code>|ensure-tenant|delete-tenant <id>|seed-alert <username>|delete-alert <id>")
	}

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		fatal("load config: %v", err)
	}
	db, err := bootstrap.InitDB(&cfg.Database)
	if err != nil {
		fatal("init db: %v", err)
	}

	switch os.Args[1] {
	case "create-admin":
		username := fmt.Sprintf("e2e_admin_%d", time.Now().UnixNano())
		id := ulid.Generate()
		hash, err := bcrypt.GenerateFromPassword([]byte(e2ePassword), bcrypt.DefaultCost)
		if err != nil {
			fatal("hash password: %v", err)
		}
		err = db.Exec(
			"INSERT INTO users (id, tenant_id, username, password, nickname, email, status, is_master, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
			id, "SYSTEM_ROOT", username, string(hash), "E2E Admin", username+"@example.com", 1, true, time.Now(), time.Now(),
		).Error
		if err != nil {
			fatal("create admin: %v", err)
		}
		emit(adminResult{Username: username, Password: e2ePassword, ID: id})

	case "delete-admin":
		requireArgs(3)
		res := db.Exec("DELETE FROM users WHERE username = ?", os.Args[2])
		if res.Error != nil {
			fatal("delete admin: %v", res.Error)
		}
		emit(deleteResult{Deleted: res.RowsAffected})

	case "verify-plan":
		requireArgs(3)
		var row struct {
			ID           string
			Name         string
			Code         string
			UserLimit    int
			StorageLimit int64
			Price        float64
			Status       int
			CreatedAt    time.Time
		}
		err := db.Raw(
			"SELECT id, name, code, user_limit, storage_limit, price, status, created_at FROM tenant_plans WHERE code = ? AND deleted_at IS NULL LIMIT 1",
			os.Args[2],
		).Scan(&row).Error
		if err != nil {
			fatal("verify plan: %v", err)
		}
		out := planResult{Found: row.ID != "", UserLimit: row.UserLimit, StorageLimit: row.StorageLimit, Price: row.Price, Status: row.Status}
		if row.ID != "" {
			out.ID, out.Name, out.Code = row.ID, row.Name, row.Code
			out.CreatedAt = row.CreatedAt.Format(time.RFC3339)
		}
		emit(out)

	case "delete-plan":
		requireArgs(3)
		res := db.Exec("DELETE FROM tenant_plans WHERE code = ?", os.Args[2])
		if res.Error != nil {
			fatal("delete plan: %v", res.Error)
		}
		emit(deleteResult{Deleted: res.RowsAffected})

	case "ensure-tenant":
		var row struct {
			ID   string
			Name string
		}
		err := db.Raw(
			"SELECT id, name FROM tenants WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT 1",
		).Scan(&row).Error
		if err != nil {
			fatal("ensure tenant: %v", err)
		}
		if row.ID != "" {
			emit(tenantRef{ID: row.ID, Name: row.Name})
			return
		}
		// CI 的 e2e 库由 AutoMigrate + Seed 建表，而 seed.sql 不写 tenants：本地恰好有存量租户，
		// CI 上则一条都没有。缺租户时现建一条，深链用例才能在任意环境定位到真实租户。
		ts := time.Now().UnixNano()
		id := ulid.Generate()
		name := fmt.Sprintf("E2E租户-%d", ts)
		domain := fmt.Sprintf("e2e-%d.local", ts)
		err = db.Exec(
			"INSERT INTO tenants (id, name, domain, status, created_at, updated_at) VALUES (?,?,?,?,?,?)",
			id, name, domain, 1, time.Now(), time.Now(),
		).Error
		if err != nil {
			fatal("create tenant: %v", err)
		}
		emit(tenantRef{ID: id, Name: name, Created: true})

	case "delete-tenant":
		requireArgs(3)
		res := db.Exec("DELETE FROM tenants WHERE id = ?", os.Args[2])
		if res.Error != nil {
			fatal("delete tenant: %v", res.Error)
		}
		emit(deleteResult{Deleted: res.RowsAffected})

	case "seed-alert":
		// seed-alert <username>：user_id 列宽只有 26（ULID），由夹具生成并回传，
		// 避免调用方传入超长值时 INSERT 被数据库拒绝。
		requireArgs(3)
		alertID := ulid.Generate()
		userID := ulid.Generate()
		err := db.Exec(
			"INSERT INTO audit_alerts (id, rule_id, rule_name, audit_log_id, user_id, username, risk_level, action, message, notified, notify_time, created_at) "+
				"VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
			alertID, "e2e-deeplink-rule", "E2E 深链规则", "", userID, os.Args[2],
			"high", "delete", "E2E 深链用例生成的告警记录", false, time.Now(), time.Now(),
		).Error
		if err != nil {
			fatal("seed alert: %v", err)
		}
		emit(alertRef{ID: alertID, UserID: userID})

	case "delete-alert":
		requireArgs(3)
		res := db.Exec("DELETE FROM audit_alerts WHERE id = ?", os.Args[2])
		if res.Error != nil {
			fatal("delete alert: %v", res.Error)
		}
		emit(deleteResult{Deleted: res.RowsAffected})

	default:
		fatal("unknown action: %s", os.Args[1])
	}
}

func emit(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fatal("marshal result: %v", err)
	}
	fmt.Println(jsonPrefix + string(b))
}

func requireArgs(n int) {
	if len(os.Args) < n {
		fatal("missing argument, usage: fixture %s <%s>", os.Args[1], "value")
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fixture: "+format+"\n", args...)
	os.Exit(1)
}
