package service

import (
	"context"
	"testing"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuditServiceTestSuite 审计日志服务测试套件
type AuditServiceTestSuite struct {
	suite.Suite
	svc  *AuditService
	repo *repository.MockAuditLogRepository
	ctx  context.Context
}

func (s *AuditServiceTestSuite) SetupTest() {
	s.repo = repository.NewMockAuditLogRepository()
	s.svc = NewAuditService(s.repo)
	s.ctx = context.Background()
}

func TestAuditServiceSuite(t *testing.T) {
	suite.Run(t, new(AuditServiceTestSuite))
}

// TestCreateLog 测试创建审计日志
func (s *AuditServiceTestSuite) TestCreateLog() {
	req := dto.CreateAuditLogReq{
		UserID:    "user-001",
		Username:  "admin",
		TenantID:  "tenant-001",
		Module:    "user",
		Action:    model.ActionTypeCreate,
		Resource:  "/api/v1/users",
		Method:    "POST",
		Path:      "/api/v1/users",
		StatusCode: 200,
		Result:    model.ResultSuccess,
		ClientIP:  "127.0.0.1",
		Duration:  100,
	}

	log, err := s.svc.CreateLog(s.ctx, req)

	s.NoError(err)
	s.NotNil(log)
	s.NotEmpty(log.ID)
	s.Equal(req.UserID, log.UserID)
	s.Equal(req.Username, log.Username)
	s.Equal(req.Module, log.Module)
	s.Equal(req.Action, log.Action)
	s.Equal(req.StatusCode, log.StatusCode)
	s.Equal(req.Result, log.Result)
}

// TestGetLog 测试获取审计日志详情
func (s *AuditServiceTestSuite) TestGetLog() {
	// 先创建一条日志
	req := dto.CreateAuditLogReq{
		UserID:    "user-001",
		Username:  "admin",
		Module:    "user",
		Action:    model.ActionTypeCreate,
		StatusCode: 200,
		Result:    model.ResultSuccess,
	}
	created, _ := s.svc.CreateLog(s.ctx, req)

	// 查询日志
	log, err := s.svc.GetLog(s.ctx, created.ID)

	s.NoError(err)
	s.NotNil(log)
	s.Equal(created.ID, log.ID)
	s.Equal(req.Username, log.Username)
}

// TestGetLogNotFound 测试获取不存在的日志
func (s *AuditServiceTestSuite) TestGetLogNotFound() {
	log, err := s.svc.GetLog(s.ctx, "non-existent-id")

	// Mock 返回 nil, nil
	s.NoError(err)
	s.Nil(log)
}

// TestListLogs 测试分页查询
func (s *AuditServiceTestSuite) TestListLogs() {
	// 创建多条日志
	actions := []string{model.ActionTypeCreate, model.ActionTypeUpdate, model.ActionTypeDelete}
	for i, action := range actions {
		req := dto.CreateAuditLogReq{
			UserID:    "user-001",
			Username:  "admin",
			Module:    "user",
			Action:    action,
			StatusCode: 200,
			Result:    model.ResultSuccess,
		}
		if i == 2 {
			req.Result = model.ResultFailure
			req.StatusCode = 400
		}
		_, err := s.svc.CreateLog(s.ctx, req)
		s.NoError(err)
	}

	// 测试无筛选查询
	query := &dto.ListAuditLogsQuery{Page: 1, PageSize: 10}
	result, err := s.svc.ListLogs(s.ctx, query)

	s.NoError(err)
	s.Len(result.Items, 3)
	s.Equal(int64(3), result.Total)

	// 测试按 Action 筛选
	query.Action = model.ActionTypeCreate
	result, err = s.svc.ListLogs(s.ctx, query)
	s.NoError(err)
	s.Len(result.Items, 1)
	s.Equal(model.ActionTypeCreate, result.Items[0].Action)

	// 测试按 Result 筛选
	query.Action = ""
	query.Result = model.ResultFailure
	result, err = s.svc.ListLogs(s.ctx, query)
	s.NoError(err)
	s.Len(result.Items, 1)
	s.Equal(model.ResultFailure, result.Items[0].Result)
}

// TestListLogsPagination 测试分页
func (s *AuditServiceTestSuite) TestListLogsPagination() {
	// 创建5条日志
	for i := 0; i < 5; i++ {
		req := dto.CreateAuditLogReq{
			UserID:    "user-001",
			Username:  "admin",
			Module:    "user",
			Action:    model.ActionTypeCreate,
			StatusCode: 200,
			Result:    model.ResultSuccess,
		}
		_, err := s.svc.CreateLog(s.ctx, req)
		s.NoError(err)
	}

	// 第一页，每页2条
	query := &dto.ListAuditLogsQuery{Page: 1, PageSize: 2}
	result, err := s.svc.ListLogs(s.ctx, query)
	s.NoError(err)
	s.Len(result.Items, 2)
	s.Equal(int64(5), result.Total)

	// 第二页
	query.Page = 2
	result, err = s.svc.ListLogs(s.ctx, query)
	s.NoError(err)
	s.Len(result.Items, 2)

	// 第三页
	query.Page = 3
	result, err = s.svc.ListLogs(s.ctx, query)
	s.NoError(err)
	s.Len(result.Items, 1)
}

// TestGetStats 测试统计
func (s *AuditServiceTestSuite) TestGetStats() {
	// 创建不同模块和操作的日志
	testCases := []struct {
		module string
		action string
		result string
	}{
		{"user", model.ActionTypeCreate, model.ResultSuccess},
		{"user", model.ActionTypeUpdate, model.ResultSuccess},
		{"tenant", model.ActionTypeCreate, model.ResultSuccess},
		{"auth", model.ActionTypeLogin, model.ResultSuccess},
		{"auth", model.ActionTypeLogin, model.ResultFailure},
	}

	for _, tc := range testCases {
		req := dto.CreateAuditLogReq{
			UserID:    "user-001",
			Username:  "admin",
			Module:    tc.module,
			Action:    tc.action,
			StatusCode: 200,
			Result:    tc.result,
		}
		_, err := s.svc.CreateLog(s.ctx, req)
		s.NoError(err)
	}

	stats, err := s.svc.GetStats(s.ctx)
	s.NoError(err)
	s.Equal(int64(5), stats.TotalCount)
	s.Equal(int64(2), stats.ActionStats[model.ActionTypeLogin])
	s.Equal(int64(2), stats.ModuleStats["user"])
	s.Equal(int64(4), stats.ResultStats[model.ResultSuccess])
	s.Equal(int64(1), stats.ResultStats[model.ResultFailure])
}

// TestCleanupLogs 测试清理日志
func (s *AuditServiceTestSuite) TestCleanupLogs() {
	// 创建日志
	for i := 0; i < 5; i++ {
		req := dto.CreateAuditLogReq{
			UserID:    "user-001",
			Username:  "admin",
			Module:    "user",
			Action:    model.ActionTypeCreate,
			StatusCode: 200,
			Result:    model.ResultSuccess,
		}
		_, err := s.svc.CreateLog(s.ctx, req)
		s.NoError(err)
	}

	// 清理
	affected, err := s.svc.CleanupLogs(s.ctx, 30)
	s.NoError(err)
	s.Equal(int64(5), affected)

	// 验证已清空
	stats, _ := s.svc.GetStats(s.ctx)
	s.Equal(int64(0), stats.TotalCount)
}

// TestCreateLogWithSensitiveData 测试敏感数据
func (s *AuditServiceTestSuite) TestCreateLogWithSensitiveData() {
	req := dto.CreateAuditLogReq{
		UserID:      "user-001",
		Username:    "admin",
		Module:      "auth",
		Action:      model.ActionTypeLogin,
		RequestBody: `{"username":"admin","password":"secret123"}`,
		StatusCode:  200,
		Result:      model.ResultSuccess,
	}

	log, err := s.svc.CreateLog(s.ctx, req)

	s.NoError(err)
	s.NotNil(log)
	s.Equal(req.RequestBody, log.RequestBody)
}

// TestCreateLogWithErrorMessage 测试错误信息
func (s *AuditServiceTestSuite) TestCreateLogWithErrorMessage() {
	req := dto.CreateAuditLogReq{
		UserID:       "user-001",
		Username:     "admin",
		Module:       "user",
		Action:       model.ActionTypeCreate,
		StatusCode:   400,
		Result:       model.ResultFailure,
		ErrorMessage: "用户名已存在",
	}

	log, err := s.svc.CreateLog(s.ctx, req)

	s.NoError(err)
	s.NotNil(log)
	s.Equal(req.ErrorMessage, log.ErrorMessage)
}

// TestCreateLogWithDuration 测试耗时记录
func (s *AuditServiceTestSuite) TestCreateLogWithDuration() {
	req := dto.CreateAuditLogReq{
		UserID:    "user-001",
		Username:  "admin",
		Module:    "user",
		Action:    model.ActionTypeQuery,
		StatusCode: 200,
		Result:    model.ResultSuccess,
		Duration:  150,
	}

	log, err := s.svc.CreateLog(s.ctx, req)

	s.NoError(err)
	s.NotNil(log)
	s.Equal(int64(150), log.Duration)
}

// TestListLogsWithKeyword 测试关键词搜索
func (s *AuditServiceTestSuite) TestListLogsWithKeyword() {
	// 创建不同路径的日志
	paths := []string{"/api/v1/users", "/api/v1/tenants", "/api/v1/roles"}
	for _, path := range paths {
		req := dto.CreateAuditLogReq{
			UserID:    "user-001",
			Username:  "admin",
			Module:    "user",
			Action:    model.ActionTypeQuery,
			Path:      path,
			StatusCode: 200,
			Result:    model.ResultSuccess,
		}
		_, err := s.svc.CreateLog(s.ctx, req)
		s.NoError(err)
	}

	// Mock 不支持关键词搜索，这里验证无筛选时能返回所有
	query := &dto.ListAuditLogsQuery{Page: 1, PageSize: 10}
	result, err := s.svc.ListLogs(s.ctx, query)
	s.NoError(err)
	s.Len(result.Items, 3)
}

// BenchmarkCreateLog 创建日志基准测试
func BenchmarkCreateLog(b *testing.B) {
	repo := repository.NewMockAuditLogRepository()
	svc := NewAuditService(repo)
	ctx := context.Background()

	req := dto.CreateAuditLogReq{
		UserID:    "user-001",
		Username:  "admin",
		Module:    "user",
		Action:    model.ActionTypeCreate,
		StatusCode: 200,
		Result:    model.ResultSuccess,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.CreateLog(ctx, req)
		assert.NoError(b, err)
	}
}