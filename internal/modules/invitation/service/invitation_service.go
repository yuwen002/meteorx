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

var (
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvitationExpired     = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed = errors.New("invitation already used")
	ErrInvitationCancelled   = errors.New("invitation has been cancelled")
	ErrEmailAlreadyInvited   = errors.New("this email already has a pending invitation for this tenant")
	ErrEmailAlreadyMember    = errors.New("this email is already a member of this tenant")
	ErrEmailNotConfigured    = errors.New("email service not configured")
)

type InvitationService struct {
	invRepo     repository.InvitationRepository
	userRepo    userRepo.UserRepository
	tenantRepo  tenantRepo.TenantRepository
	roleRepo    rbacRepo.RoleRepository
	userRoleRepo rbacRepo.UserRoleRepository
	emailer     *emailer.Emailer
	emailCfg    config.EmailConfig
	clientCfg   config.ClientConfig
	securityCfg config.SecurityConfig
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

func (s *InvitationService) List(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error) {
	return s.invRepo.ListByTenant(ctx, tenantID, page, pageSize, keyword, status)
}

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

func (s *InvitationService) GetByToken(ctx context.Context, token string) (*model.Invitation, error) {
	inv, err := s.invRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}