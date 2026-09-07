package service

import (
	"context"
	"errors"
	"testing"
	"time"

	planDto "meteorx/internal/modules/plan/dto"
	planModel "meteorx/internal/modules/plan/model"
	rbacModel "meteorx/internal/modules/rbac/model"
	tenantDto "meteorx/internal/modules/tenant/dto"
	tenantModel "meteorx/internal/modules/tenant/model"

	"github.com/stretchr/testify/suite"
)

// TenantServiceTestSuite 租户服务测试套件
type TenantServiceTestSuite struct {
	suite.Suite
	svc        *TenantService
	repo       *mockTenantRepo
	userRepo   *mockUserRepo
	roleRepo   *mockRoleRepo
	urRepo     *mockUserRoleRepo
	subRepo    *mockSubRepo
	assignPlan bool
	planErr    error
	ctx        context.Context
}

func (s *TenantServiceTestSuite) SetupTest() {
	s.repo = newMockTenantRepo()
	s.userRepo = newMockUserRepo()
	s.roleRepo = newMockRoleRepo()
	s.urRepo = newMockUserRoleRepo()
	s.subRepo = newMockSubRepo()

	s.svc = NewTenantService(s.repo, s.userRepo, s.roleRepo, s.urRepo)
	s.svc.SetSubscriptionRepository(s.subRepo)
	// 注入套餐分配 Provider（测试记录赋值是否被调用）
	s.svc.SetPlanAssignProvider(s)
	ctx := context.Background()
	s.ctx = ctx
	s.assignPlan = false
	s.planErr = nil
}

// AssignPlan 实现 TenantPlanAssignProvider
func (s *TenantServiceTestSuite) AssignPlan(ctx context.Context, tenantID string, req planDto.AssignPlanReq) error {
	if s.planErr != nil {
		return s.planErr
	}
	s.assignPlan = true
	return nil
}

func (s *TenantServiceTestSuite) seedRole(code string, status int) *rbacModel.Role {
	r := &rbacModel.Role{
		ID:       "role-" + code,
		Code:     code,
		TenantID: "",
		Status:   status,
	}
	s.roleRepo.seed(r)
	return r
}

func (s *TenantServiceTestSuite) seedTenant(id, name, domain string) *tenantModel.Tenant {
	t := &tenantModel.Tenant{
		ID:        id,
		Name:      name,
		Domain:    domain,
		Status:    tenantModel.StatusEnabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.repo.seed(t)
	return t
}

func (s *TenantServiceTestSuite) seedUser(username string) {
	s.userRepo.names[username] = true
}

func TestTenantServiceSuite(t *testing.T) {
	suite.Run(t, new(TenantServiceTestSuite))
}

// ---------- Register 租户注册 ----------

func (s *TenantServiceTestSuite) TestRegisterSuccess() {
	s.seedRole("tenant_admin", 1)
	req := tenantDto.RegisterTenantReq{}
	req.Name = "Acme"
	req.Domain = "acme"
	req.AdminUser.Username = "admin"
	req.AdminUser.Password = "Admin@123456"
	req.AdminUser.Nickname = "Admin"
	req.AdminUser.Email = "admin@acme.com"

	tenant, err := s.svc.Register(s.ctx, req)

	s.NoError(err)
	s.Require().NotNil(tenant)
	s.Equal("acme", tenant.Domain)
	s.Equal(tenantModel.StatusEnabled, tenant.Status)
	s.NotEmpty(tenant.ID)
	// 已分配默认角色
	s.Len(s.urRepo.assigned, 1)
	for _, roleIDs := range s.urRepo.assigned {
		s.Equal([]string{"role-tenant_admin"}, roleIDs)
	}
}

func (s *TenantServiceTestSuite) TestRegisterDomainConflict() {
	s.seedTenant("t-001", "Old", "acme")
	s.seedRole("tenant_admin", 1)
	req := tenantDto.RegisterTenantReq{}
	req.Name = "Acme"
	req.Domain = "acme"
	req.AdminUser.Username = "admin"
	req.AdminUser.Password = "Admin@123456"

	_, err := s.svc.Register(s.ctx, req)
	s.ErrorIs(err, ErrDomainConflict)
}

func (s *TenantServiceTestSuite) TestRegisterUsernameConflict() {
	s.seedRole("tenant_admin", 1)
	s.seedUser("admin")
	req := tenantDto.RegisterTenantReq{}
	req.Name = "Acme"
	req.Domain = "acme"
	req.AdminUser.Username = "admin"
	req.AdminUser.Password = "Admin@123456"

	_, err := s.svc.Register(s.ctx, req)
	s.ErrorIs(err, ErrUsernameConflict)
}

func (s *TenantServiceTestSuite) TestRegisterRoleMissing() {
	req := tenantDto.RegisterTenantReq{}
	req.Name = "Acme"
	req.Domain = "acme"
	req.AdminUser.Username = "admin"
	req.AdminUser.Password = "Admin@123456"

	_, err := s.svc.Register(s.ctx, req)
	s.Error(err)
	s.Contains(err.Error(), "默认租户管理员角色未配置")
}

func (s *TenantServiceTestSuite) TestRegisterRoleDisabled() {
	s.seedRole("tenant_admin", 0)
	req := tenantDto.RegisterTenantReq{}
	req.Name = "Acme"
	req.Domain = "acme"
	req.AdminUser.Username = "admin"
	req.AdminUser.Password = "Admin@123456"

	_, err := s.svc.Register(s.ctx, req)
	s.Error(err)
	s.Contains(err.Error(), "默认租户管理员角色已禁用")
}

// ---------- 查询类 ----------

func (s *TenantServiceTestSuite) TestAdminDetailNotFound() {
	_, err := s.svc.AdminDetail(s.ctx, "not-exist")
	s.ErrorIs(err, ErrTenantNotFound)
}

func (s *TenantServiceTestSuite) TestAdminDetail() {
	t := s.seedTenant("t-001", "Acme", "acme")

	got, err := s.svc.AdminDetail(s.ctx, "t-001")
	s.NoError(err)
	s.Equal(t.ID, got.ID)
	s.Equal("Acme", got.Name)
}

func (s *TenantServiceTestSuite) TestUpdateTenantStatus() {
	s.seedTenant("t-001", "Acme", "acme")

	err := s.svc.UpdateTenantStatus(s.ctx, "t-001", tenantModel.StatusDisabled)
	s.NoError(err)
	s.Equal(tenantModel.StatusDisabled, s.repo.tenants["t-001"].Status)
}

func (s *TenantServiceTestSuite) TestUpdateTenantStatusNotFound() {
	err := s.svc.UpdateTenantStatus(s.ctx, "ghost", tenantModel.StatusDisabled)
	s.ErrorIs(err, ErrTenantNotFound)
}

func (s *TenantServiceTestSuite) TestAdminUpdateNameConflict() {
	s.seedTenant("t-001", "Acme", "acme")
	s.seedTenant("t-002", "Globex", "globex")

	err := s.svc.AdminUpdate(s.ctx, "t-001", tenantDto.AdminUpdateTenantReq{Name: "Globex"})
	s.ErrorIs(err, ErrNameConflict)
}

func (s *TenantServiceTestSuite) TestAdminUpdateDomainConflict() {
	s.seedTenant("t-001", "Acme", "acme")
	s.seedTenant("t-002", "Globex", "globex")

	err := s.svc.AdminUpdate(s.ctx, "t-001", tenantDto.AdminUpdateTenantReq{Domain: "globex"})
	s.ErrorIs(err, ErrDomainConflict)
}

func (s *TenantServiceTestSuite) TestAdminUpdateSuccess() {
	s.seedTenant("t-001", "Acme", "acme")

	err := s.svc.AdminUpdate(s.ctx, "t-001", tenantDto.AdminUpdateTenantReq{Name: "Acme2", Domain: "acme2"})
	s.NoError(err)
	got := s.repo.tenants["t-001"]
	s.Equal("Acme2", got.Name)
	s.Equal("acme2", got.Domain)
}

func (s *TenantServiceTestSuite) TestAdminUpdateNotFound() {
	err := s.svc.AdminUpdate(s.ctx, "ghost", tenantDto.AdminUpdateTenantReq{Name: "X"})
	s.ErrorIs(err, ErrTenantNotFound)
}

func (s *TenantServiceTestSuite) TestGetInitStatus() {
	s.seedTenant("t-001", "Acme", "acme")

	resp, err := s.svc.GetInitStatus(s.ctx, "t-001")
	s.NoError(err)
	s.Equal("completed", resp.Status)
	s.True(resp.Initialized)
}

// ---------- 删除 / 恢复 ----------

func (s *TenantServiceTestSuite) TestAdminDeleteNotFound() {
	err := s.svc.AdminDelete(s.ctx, "ghost")
	s.ErrorIs(err, ErrTenantNotFound)
}

func (s *TenantServiceTestSuite) TestAdminDelete() {
	s.seedTenant("t-001", "Acme", "acme")

	err := s.svc.AdminDelete(s.ctx, "t-001")
	s.NoError(err)
	s.NotNil(s.repo.tenants["t-001"].DeletedAt)
}

func (s *TenantServiceTestSuite) TestAdminHardDeleteCancelsSubscription() {
	s.seedTenant("t-001", "Acme", "acme")
	s.subRepo.active = map[string]*planModel.TenantSubscription{
		"t-001": {ID: "sub-001", TenantID: "t-001", Status: planModel.SubscriptionActive},
	}

	err := s.svc.AdminHardDelete(s.ctx, "t-001")
	s.NoError(err)
	s.Equal(planModel.SubscriptionCancelled, s.subRepo.updated["sub-001"])
}

func (s *TenantServiceTestSuite) TestRestore() {
	t := s.seedTenant("t-001", "Acme", "acme")
	now := time.Now()
	t.DeletedAt = &now

	err := s.svc.Restore(s.ctx, "t-001")
	s.NoError(err)
	s.Nil(s.repo.tenants["t-001"].DeletedAt)
}

func (s *TenantServiceTestSuite) TestRestoreNotDeleted() {
	s.seedTenant("t-001", "Acme", "acme")

	err := s.svc.Restore(s.ctx, "t-001")
	s.ErrorIs(err, ErrTenantNotDeleted)
}

// ---------- 套餐 ----------

func (s *TenantServiceTestSuite) TestAdminUpdatePlan() {
	s.seedTenant("t-001", "Acme", "acme")

	err := s.svc.AdminUpdatePlan(s.ctx, "t-001", planDto.AssignPlanReq{PlanID: "plan-001"})
	s.NoError(err)
	s.True(s.assignPlan)
}

func (s *TenantServiceTestSuite) TestAdminUpdatePlanTenantNotFound() {
	err := s.svc.AdminUpdatePlan(s.ctx, "ghost", planDto.AssignPlanReq{PlanID: "plan-001"})
	s.ErrorIs(err, ErrTenantNotFound)
	s.False(s.assignPlan)
}

// ---------- 注销申请 ----------

func (s *TenantServiceTestSuite) TestApplyCancellation() {
	s.seedTenant("t-001", "Acme", "acme")

	resp, err := s.svc.ApplyCancellation(s.ctx, "t-001", tenantDto.ApplyCancellationReq{Reason: "测试注销"})
	s.NoError(err)
	s.Require().NotNil(resp)
	s.Equal("pending", resp.Status)
	s.Equal(7, resp.EstimatedDay)
	s.Len(s.repo.cancelRequests, 1)
}

func (s *TenantServiceTestSuite) TestApplyCancellationNotFound() {
	_, err := s.svc.ApplyCancellation(s.ctx, "ghost", tenantDto.ApplyCancellationReq{Reason: "x"})
	s.ErrorIs(err, ErrTenantNotFound)
}

func (s *TenantServiceTestSuite) TestApplyCancellationDuplicate() {
	s.seedTenant("t-001", "Acme", "acme")
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:       "cr-001",
		TenantID: "t-001",
		Status:   tenantModel.CancelRequestStatusPending,
	})

	_, err := s.svc.ApplyCancellation(s.ctx, "t-001", tenantDto.ApplyCancellationReq{Reason: "x"})
	s.Error(err)
}

func (s *TenantServiceTestSuite) TestApproveCancellationImmediate() {
	s.seedTenant("t-001", "Acme", "acme")
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:         "cr-001",
		TenantID:   "t-001",
		TenantName: "Acme",
		Status:     tenantModel.CancelRequestStatusPending,
		AppliedAt:  time.Now(),
	})

	resp, err := s.svc.ApproveCancellation(s.ctx, "cr-001", "approver-1", tenantDto.AdminApproveCancelReq{EffectiveDays: 0})
	s.NoError(err)
	s.Require().NotNil(resp)
	// 立即生效 → 租户被软删除、申请标记完成
	s.Equal(tenantModel.CancelRequestStatusCompleted, resp.Status)
	s.NotNil(s.repo.tenants["t-001"].DeletedAt)
	s.Equal("已完成", resp.StatusText)
}

func (s *TenantServiceTestSuite) TestApproveCancellationDelayed() {
	s.seedTenant("t-001", "Acme", "acme")
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:         "cr-001",
		TenantID:   "t-001",
		TenantName: "Acme",
		Status:     tenantModel.CancelRequestStatusPending,
		AppliedAt:  time.Now(),
	})

	resp, err := s.svc.ApproveCancellation(s.ctx, "cr-001", "approver-1", tenantDto.AdminApproveCancelReq{EffectiveDays: 3})
	s.NoError(err)
	s.Require().NotNil(resp)
	s.Equal(tenantModel.CancelRequestStatusApproved, resp.Status)
	// 延迟执行不立即删除租户
	s.Nil(s.repo.tenants["t-001"].DeletedAt)
}

func (s *TenantServiceTestSuite) TestApproveCancellationNotPending() {
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:        "cr-001",
		TenantID:  "t-001",
		Status:    tenantModel.CancelRequestStatusRejected,
		AppliedAt: time.Now(),
	})

	_, err := s.svc.ApproveCancellation(s.ctx, "cr-001", "approver-1", tenantDto.AdminApproveCancelReq{EffectiveDays: 0})
	s.Error(err)
	s.Contains(err.Error(), "not pending")
}

func (s *TenantServiceTestSuite) TestRejectCancellation() {
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:        "cr-001",
		TenantID:  "t-001",
		Status:    tenantModel.CancelRequestStatusPending,
		AppliedAt: time.Now(),
	})

	resp, err := s.svc.RejectCancellation(s.ctx, "cr-001", "approver-1", tenantDto.AdminRejectCancelReq{})
	s.NoError(err)
	s.Require().NotNil(resp)
	s.Equal(tenantModel.CancelRequestStatusRejected, resp.Status)
	s.Equal("approver-1", resp.ApproverID)
}

func (s *TenantServiceTestSuite) TestListCancelRequests() {
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:         "cr-001",
		TenantID:   "t-001",
		TenantName: "Acme",
		Status:     tenantModel.CancelRequestStatusPending,
		AppliedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	resp, err := s.svc.ListCancelRequests(s.ctx, 1, 20, tenantModel.CancelRequestStatusPending, "")
	s.NoError(err)
	s.Require().NotNil(resp)
	s.Equal(int64(1), resp.Total)
	s.Len(resp.Items, 1)
	s.Equal("待审批", resp.Items[0].StatusText)
}

func (s *TenantServiceTestSuite) TestExecuteDueCancellations() {
	s.seedTenant("t-001", "Acme", "acme")
	effective := time.Now().Add(-time.Hour)
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:          "cr-001",
		TenantID:    "t-001",
		TenantName:  "Acme",
		Status:      tenantModel.CancelRequestStatusApproved,
		EffectiveAt: &effective,
		AppliedAt:   time.Now(),
	})
	future := time.Now().Add(time.Hour)
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:          "cr-002",
		TenantID:    "t-002",
		TenantName:  "Globex",
		Status:      tenantModel.CancelRequestStatusApproved,
		EffectiveAt: &future,
		AppliedAt:   time.Now(),
	})

	executed, err := s.svc.ExecuteDueCancellations(s.ctx)
	s.NoError(err)
	s.Equal(1, executed)
	s.Equal(tenantModel.CancelRequestStatusCompleted, s.repo.cancelRequests["cr-001"].Status)
	// 未到期的不应被执行
	s.Equal(tenantModel.CancelRequestStatusApproved, s.repo.cancelRequests["cr-002"].Status)
}

func (s *TenantServiceTestSuite) TestExecuteDueCancellations_CancelsActiveSubscription() {
	s.seedTenant("t-001", "Acme", "acme")
	effective := time.Now().Add(-time.Hour)
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:          "cr-001",
		TenantID:    "t-001",
		TenantName:  "Acme",
		Status:      tenantModel.CancelRequestStatusApproved,
		EffectiveAt: &effective,
		AppliedAt:   time.Now(),
	})
	s.subRepo.active["t-001"] = &planModel.TenantSubscription{
		ID: "sub-001", TenantID: "t-001", Status: planModel.SubscriptionActive,
	}

	executed, err := s.svc.ExecuteDueCancellations(s.ctx)

	s.NoError(err)
	s.Equal(1, executed)
	s.Equal(planModel.SubscriptionCancelled, s.subRepo.updated["sub-001"])
	s.Equal(tenantModel.CancelRequestStatusCompleted, s.repo.cancelRequests["cr-001"].Status)
}

// TestExecuteCancellation_CancelErrorKeepsRequestUncompleted 订阅取消失败必须报错，
// 且申请不标记完成（下轮任务重试），避免“租户没了订阅仍生效”的静默不一致
func (s *TenantServiceTestSuite) TestExecuteCancellation_CancelErrorKeepsRequestUncompleted() {
	s.seedTenant("t-001", "Acme", "acme")
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:         "cr-001",
		TenantID:   "t-001",
		TenantName: "Acme",
		Status:     tenantModel.CancelRequestStatusApproved,
		AppliedAt:  time.Now(),
	})
	s.subRepo.active["t-001"] = &planModel.TenantSubscription{
		ID: "sub-001", TenantID: "t-001", Status: planModel.SubscriptionActive,
	}
	s.subRepo.updateErr = errors.New("db down")

	err := s.svc.ExecuteCancellation(s.ctx, s.repo.cancelRequests["cr-001"])

	s.Error(err)
	s.Contains(err.Error(), "取消租户生效订阅失败")
	// 租户已软删（幂等可重试），但申请必须保持“已通过”，不得提前标记完成
	s.NotNil(s.repo.tenants["t-001"].DeletedAt)
	s.Equal(tenantModel.CancelRequestStatusApproved, s.repo.cancelRequests["cr-001"].Status)
	s.Empty(s.subRepo.updated)
}

// TestExecuteCancellation_QueryErrorReturns 订阅查询遇到真实数据库错误同样返回，不静默
func (s *TenantServiceTestSuite) TestExecuteCancellation_QueryErrorReturns() {
	s.seedTenant("t-001", "Acme", "acme")
	s.repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID:         "cr-001",
		TenantID:   "t-001",
		TenantName: "Acme",
		Status:     tenantModel.CancelRequestStatusApproved,
		AppliedAt:  time.Now(),
	})
	s.subRepo.getErr = errors.New("db down")

	err := s.svc.ExecuteCancellation(s.ctx, s.repo.cancelRequests["cr-001"])

	s.Error(err)
	s.Contains(err.Error(), "查询租户生效订阅失败")
	s.Equal(tenantModel.CancelRequestStatusApproved, s.repo.cancelRequests["cr-001"].Status)
}

// TestAdminHardDelete_CancelErrorReported 后台硬删除后订阅取消失败不再被吞掉
func (s *TenantServiceTestSuite) TestAdminHardDelete_CancelErrorReported() {
	s.seedTenant("t-001", "Acme", "acme")
	s.subRepo.active["t-001"] = &planModel.TenantSubscription{
		ID: "sub-001", TenantID: "t-001", Status: planModel.SubscriptionActive,
	}
	s.subRepo.updateErr = errors.New("db down")

	err := s.svc.AdminHardDelete(s.ctx, "t-001")

	s.Error(err)
	s.Contains(err.Error(), "取消其生效订阅失败")
	// 租户确已物理删除
	_, ok := s.repo.tenants["t-001"]
	s.False(ok)
}
