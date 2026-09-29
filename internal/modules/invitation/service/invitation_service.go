// Package service 实现邀请模块的业务逻辑。
// 提供邀请创建、邮件发送、接受注册、状态管理等功能。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"meteorx/internal/config"
	"meteorx/internal/modules/invitation/dto"
	"meteorx/internal/modules/invitation/model"
	"meteorx/internal/modules/invitation/repository"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/internal/pkg/emailer"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
	"meteorx/pkg/security"
	"time"
)

// 业务错误定义
var (
	// ErrInvitationNotFound 邀请不存在
	ErrInvitationNotFound = errors.New("invitation not found")
	// ErrInvitationExpired 邀请已过期
	ErrInvitationExpired = errors.New("invitation has expired")
	// ErrInvitationAlreadyUsed 邀请已被使用（已接受/已取消）
	ErrInvitationAlreadyUsed = errors.New("invitation already used")
	// ErrInvitationCancelled 邀请已被取消
	ErrInvitationCancelled = errors.New("invitation has been cancelled")
	// ErrEmailAlreadyInvited 该邮箱已有待处理的邀请
	ErrEmailAlreadyInvited = errors.New("this email already has a pending invitation for this tenant")
	// ErrEmailAlreadyMember 该邮箱已是本租户成员
	ErrEmailAlreadyMember = errors.New("this email is already a member of this tenant")
	// ErrEmailNotConfigured 邮件服务未配置
	ErrEmailNotConfigured = errors.New("email service not configured")
)

// InvitationService 邀请模块业务服务。
// 管理邀请的完整生命周期：创建→邮件通知→接受→用户注册→角色分配。
type InvitationService struct {
	invRepo      repository.InvitationRepository // 邀请仓储
	userRepo     userRepo.UserRepository         // 用户仓储
	tenantRepo   tenantRepo.TenantRepository     // 租户仓储
	roleRepo     rbacRepo.RoleRepository         // 角色仓储
	userRoleRepo rbacRepo.UserRoleRepository     // 用户角色关联仓储
	emailer      *emailer.Emailer                // 邮件发送器（nil 表示未配置）
	emailCfg     config.EmailConfig              // 邮件配置
	clientCfg    config.ClientConfig             // 客户端配置（用于生成链接）
	securityCfg  config.SecurityConfig           // 安全配置（密码策略）
}

func NewInvitationService(
	invRepo repository.InvitationRepository,
	userRepo userRepo.UserRepository,
	tenantRepo tenantRepo.TenantRepository,
	roleRepo rbacRepo.RoleRepository,
	userRoleRepo rbacRepo.UserRoleRepository,
	emailCfg config.EmailConfig,
	clientCfg config.ClientConfig,
	securityCfg config.SecurityConfig,
) *InvitationService {
	var em *emailer.Emailer
	if emailCfg.Enabled {
		em = emailer.NewEmailer(emailCfg.Host, emailCfg.Port, emailCfg.Username, emailCfg.Password, emailCfg.From, emailCfg.FromName)
	}
	return &InvitationService{
		invRepo:     invRepo,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		roleRepo:    roleRepo,
		userRoleRepo: userRoleRepo,
		emailer:     em,
		emailCfg:    emailCfg,
		clientCfg:   clientCfg,
		securityCfg: securityCfg,
	}
}

// Create 创建租户邀请。
// 1. 检查邮箱是否已存在待处理邀请（同租户内不重复）
// 2. 检查邮箱是否已是租户成员
// 3. 生成 UUID 邀请令牌和过期时间（7 天）
// 4. 保存邀请记录到数据库（状态 pending）
// 5. 发送邀请邮件到目标邮箱
func (s *InvitationService) Create(ctx context.Context, tenantID, invitedBy string, req dto.CreateInvitationReq) (*model.Invitation, error) {
	existing, err := s.invRepo.FindByEmailAndTenant(ctx, req.Email, tenantID)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyInvited
	}

	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil && existingUser.TenantID == tenantID {
		return nil, ErrEmailAlreadyMember
	}

	roleIDsJSON, _ := json.Marshal(req.RoleIDs)

	token := idgen.NewUUID()
	now := time.Now()
	inv := &model.Invitation{
		ID:        idgen.New(),
		TenantID:  tenantID,
		Email:     req.Email,
		Token:     token,
		RoleIDs:   string(roleIDsJSON),
		Status:    model.InvitationStatusPending,
		InvitedBy: invitedBy,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.invRepo.Create(ctx, inv); err != nil {
		return nil, err
	}

	if s.emailer != nil {
		inviter, _ := s.userRepo.GetByID(ctx, invitedBy)
		inviterName := "管理员"
		if inviter != nil {
			inviterName = inviter.Nickname
		}

		tenant, _ := s.tenantRepo.GetByID(ctx, tenantID)
		tenantName := "团队"
		if tenant != nil {
			tenantName = tenant.Name
		}

		inviteLink := fmt.Sprintf("%s/accept-invitation?token=%s", s.clientCfg.BaseURL, token)
		_ = s.emailer.SendInvitationEmail(req.Email, inviteLink, inviterName, tenantName)
	}

	return inv, nil
}

// Accept 接受租户邀请。
// 1. 通过令牌查找邀请记录
// 2. 验证邀请状态和有效期
// 3. 验证密码强度
// 4. 创建新用户（关联到邀请的租户，邮箱标记已验证）
// 5. 分配邀请中指定的角色
// 6. 更新邀请状态为 accepted
func (s *InvitationService) Accept(ctx context.Context, req dto.AcceptInvitationReq) error {
	inv, err := s.invRepo.GetByToken(ctx, req.Token)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.Status != model.InvitationStatusPending {
		if inv.Status == model.InvitationStatusAccepted {
			return ErrInvitationAlreadyUsed
		}
		if inv.Status == model.InvitationStatusCancelled {
			return ErrInvitationCancelled
		}
		return ErrInvitationExpired
	}

	if time.Now().After(inv.ExpiresAt) {
		_ = s.invRepo.UpdateStatus(ctx, inv.ID, model.InvitationStatusExpired)
		return ErrInvitationExpired
	}

	if err := security.ValidatePassword(req.Password, s.securityCfg.PasswordPolicy); err != nil {
		return err
	}

	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return err
	}

	var roleIDs []string
	if err := json.Unmarshal([]byte(inv.RoleIDs), &roleIDs); err != nil {
		roleIDs = []string{}
	}

	now := time.Now()
	user := &userModel.User{
		ID:        idgen.NewUUID(),
		TenantID:  inv.TenantID,
		Username:  req.Username,
		Password:  hashedPassword,
		Nickname:  req.Nickname,
		Email:     inv.Email,
		EmailVerified: true,
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}

	if len(roleIDs) > 0 {
		if err := s.userRoleRepo.AssignRoles(ctx, user.ID, roleIDs); err != nil {
			return err
		}
	}

	if err := s.invRepo.UpdateStatus(ctx, inv.ID, model.InvitationStatusAccepted); err != nil {
		return err
	}

	return nil
}

// List 按租户分页查询邀请列表，支持关键词和状态筛选。
func (s *InvitationService) List(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error) {
	return s.invRepo.ListByTenant(ctx, tenantID, page, pageSize, keyword, status)
}

// Cancel 取消邀请。仅 pending 状态的邀请可取消，取消后状态变为 cancelled。
func (s *InvitationService) Cancel(ctx context.Context, tenantID, invitationID string) error {
	inv, err := s.invRepo.GetByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}
	if inv.TenantID != tenantID {
		return ErrInvitationNotFound
	}
	if inv.Status != model.InvitationStatusPending {
		return ErrInvitationAlreadyUsed
	}
	return s.invRepo.UpdateStatus(ctx, invitationID, model.InvitationStatusCancelled)
}

// Resend 重发邀请邮件。仅 pending 状态可重发，使用原令牌，不重置过期时间。
func (s *InvitationService) Resend(ctx context.Context, tenantID, invitationID string) (*model.Invitation, error) {
	inv, err := s.invRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	if inv.TenantID != tenantID {
		return nil, ErrInvitationNotFound
	}
	if inv.Status != model.InvitationStatusPending {
		return nil, ErrInvitationAlreadyUsed
	}

	if s.emailer == nil {
		return nil, ErrEmailNotConfigured
	}

	inviter, _ := s.userRepo.GetByID(ctx, inv.InvitedBy)
	inviterName := "管理员"
	if inviter != nil {
		inviterName = inviter.Nickname
	}

	tenant, _ := s.tenantRepo.GetByID(ctx, tenantID)
	tenantName := "团队"
	if tenant != nil {
		tenantName = tenant.Name
	}

	inviteLink := fmt.Sprintf("%s/accept-invitation?token=%s", s.clientCfg.BaseURL, inv.Token)
	if err := s.emailer.SendInvitationEmail(inv.Email, inviteLink, inviterName, tenantName); err != nil {
		return nil, err
	}

	return inv, nil
}

// Delete 物理删除邀请记录，不可恢复。
func (s *InvitationService) Delete(ctx context.Context, tenantID, invitationID string) error {
	inv, err := s.invRepo.GetByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}
	if inv.TenantID != tenantID {
		return ErrInvitationNotFound
	}
	return s.invRepo.Delete(ctx, invitationID)
}

// GetByToken 通过令牌查询邀请信息。用于接受邀请页面展示邀请详情。
func (s *InvitationService) GetByToken(ctx context.Context, token string) (*model.Invitation, error) {
	inv, err := s.invRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}