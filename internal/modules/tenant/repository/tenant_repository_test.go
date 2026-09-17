package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	tenantModel "meteorx/internal/modules/tenant/model"
	userModel "meteorx/internal/modules/user/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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

func TestCreate(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant, _ := sampleTenantAndAdmin()

	mock.ExpectExec("INSERT INTO `tenants`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), tenant)
	assert.NoError(t, err)
}

func TestCreate_DBError(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant, _ := sampleTenantAndAdmin()

	mock.ExpectExec("INSERT INTO `tenants`").
		WillReturnError(errors.New("connection refused"))

	err := repo.Create(context.Background(), tenant)
	assert.Error(t, err)
}

func TestGetByID(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	rows := sqlmock.NewRows([]string{"id", "name", "domain", "status", "created_at", "updated_at", "deleted_at"}).
		AddRow("t-001", "Acme", "acme", 1, time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT .+ FROM `tenants` .+ LIMIT \\?").
		WillReturnRows(rows)

	tenant, err := repo.GetByID(context.Background(), "t-001")
	assert.NoError(t, err)
	assert.Equal(t, "t-001", tenant.ID)
	assert.Equal(t, "Acme", tenant.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectQuery("SELECT .+ FROM `tenants` .+ LIMIT \\?").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByID(context.Background(), "ghost")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestGetByDomain(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	rows := sqlmock.NewRows([]string{"id", "name", "domain", "status", "created_at", "updated_at", "deleted_at"}).
		AddRow("t-001", "Acme", "acme", 1, time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT .+ FROM `tenants` WHERE domain = .+ LIMIT \\?").
		WillReturnRows(rows)

	tenant, err := repo.GetByDomain(context.Background(), "acme")
	assert.NoError(t, err)
	assert.Equal(t, "t-001", tenant.ID)
}

func TestGetByDomain_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectQuery("SELECT .+ FROM `tenants` WHERE domain = .+ LIMIT \\?").
		WillReturnError(gorm.ErrRecordNotFound)

	tenant, err := repo.GetByDomain(context.Background(), "ghost")
	assert.NoError(t, err)
	assert.Nil(t, tenant)
}

func TestGetByName(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	rows := sqlmock.NewRows([]string{"id", "name", "domain", "status", "created_at", "updated_at", "deleted_at"}).
		AddRow("t-001", "Acme", "acme", 1, time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT .+ FROM `tenants` WHERE name = .+ LIMIT \\?").
		WillReturnRows(rows)

	tenant, err := repo.GetByName(context.Background(), "Acme")
	assert.NoError(t, err)
	assert.Equal(t, "Acme", tenant.Name)
}

func TestGetByName_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectQuery("SELECT .+ FROM `tenants` WHERE name = .+ LIMIT \\?").
		WillReturnError(gorm.ErrRecordNotFound)

	tenant, err := repo.GetByName(context.Background(), "Ghost")
	assert.NoError(t, err)
	assert.Nil(t, tenant)
}

func TestUpdate(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("UPDATE `tenants` SET .+ WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	tenant := &tenantModel.Tenant{Name: "NewName", Description: "NewDesc"}
	err := repo.Update(context.Background(), "t-001", tenant)
	assert.NoError(t, err)
}

func TestUpdate_NoFields(t *testing.T) {
	gormDB, _ := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant := &tenantModel.Tenant{}

	err := repo.Update(context.Background(), "t-001", tenant)
	assert.NoError(t, err)
}

func TestUpdateStatus(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("UPDATE `tenants` SET .+ WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateStatus(context.Background(), "t-001", 0)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("UPDATE `tenants` SET .+deleted_at.+ WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), "t-001")
	assert.NoError(t, err)
}

func TestHardDelete(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("DELETE FROM `tenants` WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.HardDelete(context.Background(), "t-001")
	assert.NoError(t, err)
}

func TestFindPage(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT count.+FROM `tenants`").
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "name", "domain", "status", "created_at", "updated_at", "deleted_at"}).
		AddRow("t-001", "Acme", "acme", 1, now, now, nil).
		AddRow("t-002", "Beta", "beta", 1, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `tenants` .+ ORDER BY created_at DESC LIMIT \\?").
		WillReturnRows(rows)

	tenants, total, err := repo.FindPage(context.Background(), 1, 10, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, tenants, 2)
}

func TestFindPage_WithFilters(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	status := 1

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `tenants` WHERE name LIKE .+").
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "name", "domain", "status", "created_at", "updated_at", "deleted_at"}).
		AddRow("t-001", "Acme", "acme", 1, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `tenants` .+ ORDER BY created_at DESC LIMIT \\?").
		WillReturnRows(rows)

	tenants, total, err := repo.FindPage(context.Background(), 1, 10, "Acme", &status)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tenants, 1)
}

func TestFindDeleted(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `tenants` WHERE deleted_at IS NOT NULL").
		WillReturnRows(countRows)

	deletedAt := gorm.DeletedAt{Time: now, Valid: true}
	rows := sqlmock.NewRows([]string{"id", "name", "domain", "status", "created_at", "updated_at", "deleted_at"}).
		AddRow("t-001", "Acme", "acme", 0, now, now, deletedAt)
	mock.ExpectQuery("SELECT .+ FROM `tenants` WHERE deleted_at IS NOT NULL.*ORDER BY deleted_at DESC LIMIT \\?").
		WillReturnRows(rows)

	tenants, total, err := repo.FindDeleted(context.Background(), 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tenants, 1)
	assert.NotNil(t, tenants[0].DeletedAt)
}

func TestRestore(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("UPDATE `tenants` SET .+deleted_at.+ WHERE id = \\? AND deleted_at IS NOT NULL").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Restore(context.Background(), "t-001")
	assert.NoError(t, err)
}

func TestRestore_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("UPDATE `tenants` SET .+ WHERE id = \\? AND deleted_at IS NOT NULL").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Restore(context.Background(), "ghost")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found or not deleted")
}

func TestBatchUpdateStatus(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	ids := []string{"t-001", "t-002", "ghost"}

	existingRows := sqlmock.NewRows([]string{"id"}).AddRow("t-001").AddRow("t-002")
	mock.ExpectQuery("SELECT `id` FROM `tenants` WHERE id IN").
		WithArgs("t-001", "t-002", "ghost").
		WillReturnRows(existingRows)

	mock.ExpectExec("UPDATE `tenants` SET .+ WHERE id IN").WillReturnResult(sqlmock.NewResult(0, 2))

	affected, failedIDs, err := repo.BatchUpdateStatus(context.Background(), ids, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), affected)
	assert.Equal(t, []string{"ghost"}, failedIDs)
}

func TestBatchDelete(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	ids := []string{"t-001", "ghost"}

	existingRows := sqlmock.NewRows([]string{"id"}).AddRow("t-001")
	mock.ExpectQuery("SELECT `id` FROM `tenants` WHERE id IN").
		WithArgs("t-001", "ghost").
		WillReturnRows(existingRows)

	mock.ExpectExec("UPDATE `tenants` SET .+deleted_at.+ WHERE id IN").WillReturnResult(sqlmock.NewResult(0, 1))

	affected, failedIDs, err := repo.BatchDelete(context.Background(), ids)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), affected)
	assert.Equal(t, []string{"ghost"}, failedIDs)
}

func sampleCancelRequest() *tenantModel.CancelRequest {
	now := time.Now()
	return &tenantModel.CancelRequest{
		ID:       "cr-001",
		TenantID: "t-001", TenantName: "Acme",
		Reason: "不再需要", Status: tenantModel.CancelRequestStatusPending,
		AppliedAt: now, CreatedAt: now, UpdatedAt: now,
	}
}

func TestCreateCancelRequest(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("INSERT INTO `cancel_requests`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateCancelRequest(context.Background(), sampleCancelRequest())
	assert.NoError(t, err)
}

func TestGetCancelRequestByID(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "tenant_name", "reason", "status",
		"approver_id", "review_remark", "applied_at", "created_at", "updated_at", "deleted_at"}).
		AddRow("cr-001", "t-001", "Acme", "no need", 1, "", "", now, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `cancel_requests` WHERE id = .+ LIMIT \\?").
		WillReturnRows(rows)

	req, err := repo.GetCancelRequestByID(context.Background(), "cr-001")
	assert.NoError(t, err)
	assert.Equal(t, "cr-001", req.ID)
	assert.Equal(t, "t-001", req.TenantID)
}

func TestGetPendingCancelRequestByTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "tenant_name", "reason", "status",
		"approver_id", "review_remark", "applied_at", "created_at", "updated_at", "deleted_at"}).
		AddRow("cr-001", "t-001", "Acme", "no need", tenantModel.CancelRequestStatusPending, "", "", now, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `cancel_requests` WHERE .+ LIMIT \\?").
		WillReturnRows(rows)

	req, err := repo.GetPendingCancelRequestByTenant(context.Background(), "t-001")
	assert.NoError(t, err)
	assert.Equal(t, tenantModel.CancelRequestStatusPending, req.Status)
}

func TestUpdateCancelRequest(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	mock.ExpectExec("UPDATE `cancel_requests` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := sampleCancelRequest()
	req.Status = tenantModel.CancelRequestStatusApproved
	err := repo.UpdateCancelRequest(context.Background(), req)
	assert.NoError(t, err)
}

func TestFindCancelRequests(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `cancel_requests`").
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "tenant_name", "reason", "status",
		"approver_id", "review_remark", "applied_at", "created_at", "updated_at", "deleted_at"}).
		AddRow("cr-001", "t-001", "Acme", "no need", 1, "", "", now, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `cancel_requests` .+ ORDER BY created_at DESC LIMIT \\?").
		WillReturnRows(rows)

	items, total, err := repo.FindCancelRequests(context.Background(), 1, 20, 0, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)
}

func TestFindApprovedDueCancelRequests(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "tenant_name", "reason", "status",
		"approver_id", "review_remark", "applied_at", "created_at", "updated_at", "deleted_at"}).
		AddRow("cr-001", "t-001", "Acme", "过期", tenantModel.CancelRequestStatusApproved, "approver-1", "", now, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `cancel_requests` WHERE .+").
		WithArgs(tenantModel.CancelRequestStatusApproved, now).
		WillReturnRows(rows)

	items, err := repo.FindApprovedDueCancelRequests(context.Background(), now)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "cr-001", items[0].ID)
}

func TestBatchUpdateStatus_AllMissing(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	ids := []string{"ghost1", "ghost2"}

	existingRows := sqlmock.NewRows([]string{"id"})
	mock.ExpectQuery("SELECT `id` FROM `tenants` WHERE id IN").
		WithArgs("ghost1", "ghost2").
		WillReturnRows(existingRows)

	affected, failedIDs, err := repo.BatchUpdateStatus(context.Background(), ids, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), affected)
	assert.Equal(t, ids, failedIDs)
}

func TestBatchDelete_AllMissing(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)

	ids := []string{"ghost1"}

	existingRows := sqlmock.NewRows([]string{"id"})
	mock.ExpectQuery("SELECT `id` FROM `tenants` WHERE id IN").
		WithArgs("ghost1").
		WillReturnRows(existingRows)

	affected, failedIDs, err := repo.BatchDelete(context.Background(), ids)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), affected)
	assert.Equal(t, ids, failedIDs)
}

func TestCreate_WithDeletedAt(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewTenantRepository(gormDB).(*tenantRepository)
	tenant, _ := sampleTenantAndAdmin()
	delTime := time.Now()
	tenant.DeletedAt = &delTime

	mock.ExpectExec("INSERT INTO `tenants`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), tenant)
	assert.NoError(t, err)
}