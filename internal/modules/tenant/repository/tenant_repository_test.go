package repository

import (
	"context"
	"errors"
	"testing"

	tenantModel "meteorx/internal/modules/tenant/model"
	userModel "meteorx/internal/modules/user/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newTestDB 基于 sqlmock 构建 GORM DB，并断言测试结束时所有预期的 SQL 均被消费。
func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return gormDB, mock
}

func sampleTenantAndAdmin() (*tenantModel.Tenant, *userModel.User) {
	return &tenantModel.Tenant{ID: "t-001", Name: "Acme", Domain: "acme", Status: 1},
		&userModel.User{ID: "u-001", TenantID: "t-001", Username: "boss", Nickname: "Boss", Email: "boss@acme.com", Status: 1}
}

// TestCreateTenantWithAdmin_AllTablesCommitted 验证三张表在同一个事务内依次写入并最终提交。
func TestCreateTenantWithAdmin_AllTablesCommitted(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant, user := sampleTenantAndAdmin()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `tenants`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `users`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `user_roles`").WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	err := repo.CreateTenantWithAdmin(context.Background(), tenant, user, []string{"role-tenant-admin", "role-x"})
	assert.NoError(t, err)
}

// TestCreateTenantWithAdmin_RoleWriteFailureRollsBack 验证角色关联写入失败时：
// 事务整体回滚并向上返回错误——这是“注册不留无角色租户”的关键保证。
// 若实现退化为事务外三连写（或吞掉角色写入错误），BEGIN/ROLLBACK 期望将无法满足。
func TestCreateTenantWithAdmin_RoleWriteFailureRollsBack(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant, user := sampleTenantAndAdmin()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `tenants`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `users`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `user_roles`").WillReturnError(errors.New("duplicate entry"))
	mock.ExpectRollback()

	err := repo.CreateTenantWithAdmin(context.Background(), tenant, user, []string{"role-tenant-admin"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "写入用户角色关联失败")
}

// TestCreateTenantWithAdmin_NoRolesSkipsRoleTable 角色列表为空时仅写租户与用户两表，不触碰 user_roles。
func TestCreateTenantWithAdmin_NoRolesSkipsRoleTable(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant, user := sampleTenantAndAdmin()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `tenants`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `users`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.CreateTenantWithAdmin(context.Background(), tenant, user, nil)
	assert.NoError(t, err)
}
