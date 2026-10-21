// E2E 数据夹具：浏览器测试需要的账号与直达数据库的断言/清理能力。
//
// 子命令：
//   create-admin        创建临时超管（is_master=true），输出凭据
//   delete-admin <用户名> 删除临时超管
//   verify-plan <code>  直接查 MySQL 验证套餐真实落库
//   delete-plan <code>  物理删除测试套餐
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

func main() {
	if len(os.Args) < 2 {
		fatal("usage: fixture create-admin|delete-admin <username>|verify-plan <code>|delete-plan <code>")
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

