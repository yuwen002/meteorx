package service

import (
	"context"
	"testing"

	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/stretchr/testify/suite"
)

// SessionServiceTestSuite 会话服务测试套件
type SessionServiceTestSuite struct {
	suite.Suite
	svc  *SessionService
	repo *repository.MockAuditLogRepository
	ctx  context.Context
}

func (s *SessionServiceTestSuite) SetupTest() {
	s.repo = repository.NewMockAuditLogRepository()
	s.svc = NewSessionService(s.repo)
	s.ctx = context.Background()
}

func TestSessionServiceSuite(t *testing.T) {
	suite.Run(t, new(SessionServiceTestSuite))
}

// TestGetSessionLogs 测试获取会话日志
func (s *SessionServiceTestSuite) TestGetSessionLogs() {
	sessionID := "session-001"

	// 创建多条同一会话的日志
	for i := 0; i < 5; i++ {
		log := &model.AuditLog{
			UserID:     "user-001",
			Username:   "admin",
			SessionID:  sessionID,
			Module:     "user",
			Action:     model.ActionTypeCreate,
			Method:     "POST",
			Path:       "/api/v1/users",
			StatusCode: 200,
			Result:     model.ResultSuccess,
			ClientIP:   "127.0.0.1",
			Duration:   int64(100 + i*10),
		}
		s.repo.Create(s.ctx, log)
	}

	result, err := s.svc.GetSessionLogs(s.ctx, sessionID)

	s.NoError(err)
	s.NotNil(result)
	s.Equal(sessionID, result.SessionID)
	s.Equal("user-001", result.UserID)
	s.Equal("admin", result.Username)
	s.Equal(int64(5), result.TotalOps)
	s.Equal(5, len(result.Logs))
	s.Equal(int64(600), result.Duration)
}

// TestGetSessionLogsEmpty 测试获取空会话日志
func (s *SessionServiceTestSuite) TestGetSessionLogsEmpty() {
	result, err := s.svc.GetSessionLogs(s.ctx, "non-existent-session")

	s.NoError(err)
	s.NotNil(result)
	s.Equal(int64(0), result.TotalOps)
	s.Equal(0, len(result.Logs))
}

// TestListSessions 测试获取会话列表
func (s *SessionServiceTestSuite) TestListSessions() {
	// 创建多个会话的日志
	sessions := []string{"session-001", "session-002", "session-003"}
	for _, sessionID := range sessions {
		for i := 0; i < 3; i++ {
			log := &model.AuditLog{
				UserID:     "user-001",
				Username:   "admin",
				SessionID:  sessionID,
				Module:     "user",
				Action:     model.ActionTypeCreate,
				StatusCode: 200,
				Result:     model.ResultSuccess,
				Duration:   int64(100 + i*10),
			}
			s.repo.Create(s.ctx, log)
		}
	}

	result, total, err := s.svc.ListSessions(s.ctx, 1, 20, "")

	s.NoError(err)
	s.NotNil(result)
	s.True(total >= 1)
}

// TestListSessionsByUser 测试按用户筛选会话列表
func (s *SessionServiceTestSuite) TestListSessionsByUser() {
	// 创建不同用户的会话
	log1 := &model.AuditLog{
		UserID:     "user-001",
		Username:   "admin",
		SessionID:  "session-001",
		Module:     "user",
		Action:     model.ActionTypeCreate,
		StatusCode: 200,
		Result:     model.ResultSuccess,
		Duration:   100,
	}
	s.repo.Create(s.ctx, log1)

	log2 := &model.AuditLog{
		UserID:     "user-002",
		Username:   "editor",
		SessionID:  "session-002",
		Module:     "wiki",
		Action:     model.ActionTypeUpdate,
		StatusCode: 200,
		Result:     model.ResultSuccess,
		Duration:   150,
	}
	s.repo.Create(s.ctx, log2)

	// 按用户筛选
	result, _, err := s.svc.ListSessions(s.ctx, 1, 20, "user-001")

	s.NoError(err)
	s.NotNil(result)
	// 应该只返回 user-001 的会话
	s.Len(result, 1)
	s.Equal("session-001", result[0].SessionID)
}
