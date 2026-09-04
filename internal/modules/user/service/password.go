package service

import (
	"context"

	"meteorx/pkg/crypto"
)

func (s *UserService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !crypto.CheckPassword(oldPassword, user.Password) {
		return ErrWrongOldPassword
	}

	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.repo.Update(ctx, user)
}

// ResetPassword 管理员重置用户密码（不需要原密码）
func (s *UserService) ResetPassword(ctx context.Context, userID, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.repo.Update(ctx, user)
}

// CountByTenant 统计指定租户下的用户总数
func (s *UserService) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	return s.repo.CountByTenant(ctx, tenantID)
}

// CountAllUsers 统计所有用户总数（跨租户）
func (s *UserService) CountAllUsers(ctx context.Context) (int64, error) {
	return s.repo.CountAllUsers(ctx)
}
