package service

import (
	"context"
	"errors"
	"fmt"
	"meteorx/internal/modules/tenant/dto"
	tenantModel "meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	"meteorx/pkg/crypto"
	"meteorx/pkg/uuid"
)

// 定义业务层特有的错误，方便 Handler 层判断
var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrDomainConflict = errors.New("domain already exists")
)

type TenantService struct {
	repo repository.TenantRepository
}

func NewTenantService(repo repository.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

// Register 注册新租户及其管理员用户
// 该方法会生成租户ID和用户ID，加密管理员密码，创建租户和管理员用户模型，并通过事务持久化到数据库
func (s *TenantService) Register(ctx context.Context, req dto.RegisterTenantReq) (*tenantModel.Tenant, error) {
	// 1. 生成符合 size:26 限制的唯一 ID (这里使用高并发安全的伪随机/时间戳方案，之后可换 ULID)
	tenantID := uuid.Generate()
	userID := uuid.Generate()

	// 2. 密码加密
	hashedPassword, err := crypto.HashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, fmt.Errorf("password encryption failed: %w", err)
	}

	// 3. 构造数据库租户模型 PO (全量对齐你精致的 DTO)
	tenantPO := &tenantModel.Tenant{
		ID:           tenantID,
		Name:         req.Name, // 修正为 req.Name
		Domain:       req.Domain,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		Region:       req.Region,
		Logo:         req.Logo,
		Extra:        req.Extra,
		Status:       1, // 默认正常激活
	}

	// 4. 构造数据库用户模型 PO
	userPO := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: req.AdminUser.Username,
		Password: string(hashedPassword),
		Nickname: req.AdminUser.Nickname,
		Email:    req.AdminUser.Email,
		Role:     "admin", // 租户创始人，锁定 admin 角色
		IsMaster: false,   // 绝不是 MaaS 平台上帝
		Status:   1,       // 默认激活
	}

	// 5. 抛给持久层执行事务
	if err := s.repo.CreateTenantWithAdmin(ctx, tenantPO, userPO); err != nil {
		return nil, err
	}

	return tenantPO, nil
}

// AdminCreate 后台管理员手动创建租户
func (s *TenantService) AdminCreate(ctx context.Context, req dto.AdminCreateTenantReq) (*tenantModel.Tenant, error) {
	// 1. 唯一 ID 生成
	tenantID := uuid.Generate()
	userID := uuid.Generate()

	// 2. 初始管理员密码加密
	hashedPassword, err := crypto.HashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, fmt.Errorf("admin user password encryption failed: %w", err)
	}

	// 3. 组装纯业务租户模型
	tenant := &tenantModel.Tenant{
		ID:           tenantID,
		Name:         req.Name,
		Domain:       req.Domain,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		Region:       req.Region,
		Logo:         req.Logo,
		Extra:        req.Extra,
		Status:       req.Status, // 使用后台指定的的状态
	}

	// 4. 组装纯业务用户模型
	user := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: req.AdminUser.Username,
		Password: string(hashedPassword),
		Nickname: req.AdminUser.Nickname,
		Email:    req.AdminUser.Email,
		Role:     "admin", // 锁定为该租户的主管理员
		IsMaster: false,   // 后台创建的也只是普通租户管理员，绝非 MaaS 平台超级管理员
		Status:   1,       // 默认激活用户状态
	}

	// 5. 交付底层 Repository 开启强一致性事务落库
	// 完美复用你之前写好的 CreateTenantWithAdmin 事务方法
	if err := s.repo.CreateTenantWithAdmin(ctx, tenant, user); err != nil {
		return nil, err
	}

	return tenant, nil
}
