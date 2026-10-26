package service

import (
	"context"
	"errors"
	"fmt"
	"meteorx/internal/common/validator"
	"meteorx/internal/config"
	rbacModel "meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/user/importer"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
	"meteorx/pkg/security"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// 单次导入的默认初始密码长度：12 位在"可复制下发"与"足够随机"之间取平衡，
// 若安全策略要求更长则按策略下限 +2 自动放宽。
const defaultInitialPasswordLength = 12

// ImportRowError 单行校验失败明细。Row 是文件中的物理行号（表头为第 1 行），
// 操作者据此可以直接定位到 Excel 的哪一行。
type ImportRowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ImportUserResult 单行导入成功明细。InitialPassword 为服务端生成的随机初始密码，
// 只在本次响应中回显一次（不落库、不进审计日志），由管理员当面或通过安全渠道下发。
type ImportUserResult struct {
	Row             int      `json:"row"`
	UserID          string   `json:"user_id"`
	Username        string   `json:"username"`
	Nickname        string   `json:"nickname"`
	Email           string   `json:"email"`
	RoleCodes       []string `json:"role_codes"`
	InitialPassword string   `json:"initial_password"`
}

// ImportResult 批量导入结果汇总。
type ImportResult struct {
	Total    int                `json:"total"`
	Created  int                `json:"created"`
	Users    []ImportUserResult `json:"users"`
	Warnings []string           `json:"warnings,omitempty"`
}

// importRowInput 单行业务字段校验载体：规则与 dto.CreateUserReq 对齐，
// 使"逐个新建"与"批量导入"对同一份数据给出一致的判定。
type importRowInput struct {
	Username   string `json:"username" label:"用户名" validate:"required,alphanum,min=4,max=50"`
	Nickname   string `json:"nickname" label:"昵称" validate:"required,max=50"`
	Email      string `json:"email" label:"邮箱" validate:"omitempty,email"`
	Department string `json:"department" label:"部门" validate:"omitempty,max=100"`
	Position   string `json:"position" label:"职位" validate:"omitempty,max=100"`
}

// resolvedRole 角色编码解析结果（含解析失败原因，供缓存复用）。
type resolvedRole struct {
	id     string
	code   string
	errMsg string
}

// preparedRow 通过校验、等待写入的一行。
type preparedRow struct {
	lineNo    int
	input     importRowInput
	roleIDs   []string
	roleCodes []string
}

// SetImportRepository 注入批量导入仓储（由 bootstrap 组装；未注入时导入接口不可用）。
func (s *UserService) SetImportRepository(repo repository.ImportRepository) {
	s.importRepo = repo
}

// SetPasswordPolicy 注入密码策略，用于决定生成初始密码的长度。
func (s *UserService) SetPasswordPolicy(policy config.PasswordPolicyConfig) {
	s.passwordPolicy = policy
}

// ImportUsers 批量导入用户：全量校验通过后，在单个事务内一次性写入。
//
// 返回值有三态：
//   - (result, nil, nil)：全部成功；
//   - (nil, rowErrors, nil)：存在行级问题，一条都没写库（全量事务语义）；
//   - (nil, nil, err)：配额不足、角色不可用、数据库失败等整体性错误。
func (s *UserService) ImportUsers(ctx context.Context, tenantID string, rows []importer.Row, defaultRoleIDs []string) (*ImportResult, []ImportRowError, error) {
	if s.importRepo == nil {
		return nil, nil, errors.New("批量导入未启用")
	}
	if tenantID == "" {
		return nil, nil, newValidationErr("未获取到租户信息")
	}
	if len(rows) == 0 {
		return nil, nil, newValidationErr("文件中没有可导入的数据行")
	}

	// 1. 默认角色（表单选择）先校验一次，避免每行重复查库
	defaultRoleIDs, defaultRoleCodes, err := s.validateDefaultRoles(ctx, defaultRoleIDs)
	if err != nil {
		return nil, nil, err
	}

	// 2. 配额：整批导入要么放得下，要么直接拒绝（否则事务会在写入末尾撞上限）
	if err := s.checkImportQuota(ctx, tenantID, len(rows)); err != nil {
		return nil, nil, err
	}

	// 3. 逐行校验（字段规则 + 文件内重名 + 角色编码解析），收集全部问题后一起返回
	rowErrs := make([]ImportRowError, 0)
	prepared := make([]preparedRow, 0, len(rows))
	seenUsernames := make(map[string]int, len(rows))
	roleCache := make(map[string]resolvedRole)
	warnings := make([]string, 0)

	for _, row := range rows {
		input := importRowInput{
			Username:   row.Username,
			Nickname:   row.Nickname,
			Email:      row.Email,
			Department: row.Department,
			Position:   row.Position,
		}

		fieldErrs := validator.FieldErrors(input, validator.ValidateStruct(input))
		if len(fieldErrs) > 0 {
			for _, field := range sortedKeys(fieldErrs) {
				rowErrs = append(rowErrs, ImportRowError{Row: row.LineNo, Field: field, Message: fieldErrs[field]})
			}
			continue
		}

		if firstLine, dup := seenUsernames[input.Username]; dup {
			rowErrs = append(rowErrs, ImportRowError{
				Row:     row.LineNo,
				Field:   "username",
				Message: fmt.Sprintf("用户名在本文件中重复（首次出现在第 %d 行）", firstLine),
			})
			continue
		}
		seenUsernames[input.Username] = row.LineNo

		roleIDs, roleCodes, roleErrs := s.resolveRowRoles(ctx, tenantID, row.RoleCodes, defaultRoleIDs, defaultRoleCodes, roleCache)
		if len(roleErrs) > 0 {
			for _, msg := range roleErrs {
				rowErrs = append(rowErrs, ImportRowError{Row: row.LineNo, Field: "roles", Message: msg})
			}
			continue
		}
		if len(row.RoleCodes) == 0 {
			warnings = append(warnings, fmt.Sprintf("第 %d 行未填写角色，已使用所选默认角色", row.LineNo))
		}

		prepared = append(prepared, preparedRow{lineNo: row.LineNo, input: input, roleIDs: roleIDs, roleCodes: roleCodes})
	}

	// 4. 库内占用检查（现存用户 + 本租户回收站）
	usernames := make([]string, 0, len(prepared))
	for _, p := range prepared {
		usernames = append(usernames, p.input.Username)
	}
	active, trash, err := s.importRepo.CheckUsernames(ctx, tenantID, usernames)
	if err != nil {
		return nil, nil, fmt.Errorf("检查用户名占用失败: %w", err)
	}
	for _, p := range prepared {
		if _, ok := active[p.input.Username]; ok {
			rowErrs = append(rowErrs, ImportRowError{Row: p.lineNo, Field: "username", Message: "用户名已被使用"})
			continue
		}
		if _, ok := trash[p.input.Username]; ok {
			rowErrs = append(rowErrs, ImportRowError{
				Row:     p.lineNo,
				Field:   "username",
				Message: "该租户回收站中已存在同名账号，请先在回收站彻底删除后再导入",
			})
		}
	}

	// 5. 有任何一行不合法就整批拒绝，一条都不写库
	if len(rowErrs) > 0 {
		sort.SliceStable(rowErrs, func(i, j int) bool {
			if rowErrs[i].Row != rowErrs[j].Row {
				return rowErrs[i].Row < rowErrs[j].Row
			}
			return rowErrs[i].Field < rowErrs[j].Field
		})
		return nil, rowErrs, nil
	}

	// 6. 生成初始密码并哈希（bcrypt 是 CPU 密集操作，按 CPU 数并发，避免 500 行串行等十几秒）
	users, passwords, roleIDsByUserID, err := s.buildImportUsers(tenantID, prepared)
	if err != nil {
		return nil, nil, err
	}

	// 7. 单事务写入（用户 + 角色关联），失败即整体回滚
	if err := s.importRepo.CreateUsersWithRoles(ctx, users, roleIDsByUserID); err != nil {
		return nil, nil, fmt.Errorf("批量写入失败（已整体回滚）: %w", err)
	}

	// 8. 组装结果（保持文件行序）
	result := &ImportResult{
		Total:    len(rows),
		Created:  len(users),
		Users:    make([]ImportUserResult, 0, len(users)),
		Warnings: warnings,
	}
	for i, u := range users {
		result.Users = append(result.Users, ImportUserResult{
			Row:             prepared[i].lineNo,
			UserID:          u.ID,
			Username:        u.Username,
			Nickname:        u.Nickname,
			Email:           u.Email,
			RoleCodes:       prepared[i].roleCodes,
			InitialPassword: passwords[i],
		})
	}
	return result, nil, nil
}

// buildImportUsers 并发生成用户与初始密码（bcrypt 哈希），结果下标与文件行序一致。
func (s *UserService) buildImportUsers(tenantID string, prepared []preparedRow) ([]*model.User, []string, map[string][]string, error) {
	n := len(prepared)
	users := make([]*model.User, n)
	passwords := make([]string, n)
	roleIDsByUserID := make(map[string][]string, n)
	if n == 0 {
		return users, passwords, roleIDsByUserID, nil
	}

	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8 // 限制并发：bcrypt 本身吃满 CPU，过多协程只会互相抢占
	}
	if workers > n {
		workers = n
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		next     int
	)
	worker := func() {
		for {
			mu.Lock()
			i := next
			next++
			mu.Unlock()
			if i >= n || firstErr != nil {
				return
			}

			plain, err := s.generateInitialPassword()
			var hashed string
			if err == nil {
				hashed, err = crypto.HashPassword(plain)
			}

			mu.Lock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			p := prepared[i]
			u := &model.User{
				ID:         idgen.New(),
				TenantID:   tenantID,
				Username:   p.input.Username,
				Password:   hashed,
				Nickname:   p.input.Nickname,
				Email:      p.input.Email,
				Department: p.input.Department,
				Position:   p.input.Position,
				Status:     1,
				IsMaster:   false,
			}
			users[i] = u
			passwords[i] = plain
			roleIDsByUserID[u.ID] = p.roleIDs
			mu.Unlock()
		}
	}

	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			worker()
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return nil, nil, nil, fmt.Errorf("生成初始密码失败: %w", firstErr)
	}
	return users, passwords, roleIDsByUserID, nil
}

// validateDefaultRoles 校验表单传入的默认角色，并返回去重后的角色ID与展示用编码。
func (s *UserService) validateDefaultRoles(ctx context.Context, roleIDs []string) ([]string, []string, error) {
	if len(roleIDs) == 0 {
		return nil, nil, nil
	}

	unique := make([]string, 0, len(roleIDs))
	codes := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, nil, nil
	}

	if err := s.validateRoleIDs(ctx, unique, rbacModel.RoleScopeTenant); err != nil {
		return nil, nil, err
	}
	for _, id := range unique {
		role, err := s.roleRepo.GetByID(ctx, id)
		if err != nil {
			return nil, nil, newValidationErr("角色 '%s' 不存在", id)
		}
		codes = append(codes, role.Code)
	}
	return unique, codes, nil
}

// resolveRowRoles 解析单行角色：文件列优先，缺省回落到表单选择的默认角色。
func (s *UserService) resolveRowRoles(
	ctx context.Context,
	tenantID string,
	rowCodes []string,
	defaultRoleIDs []string,
	defaultRoleCodes []string,
	cache map[string]resolvedRole,
) ([]string, []string, []string) {
	if len(rowCodes) == 0 {
		if len(defaultRoleIDs) == 0 {
			return nil, nil, []string{"未指定角色：请在文件 roles 列填写角色编码，或在导入时选择默认角色"}
		}
		return defaultRoleIDs, defaultRoleCodes, nil
	}

	roleIDs := make([]string, 0, len(rowCodes))
	roleCodes := make([]string, 0, len(rowCodes))
	var errs []string
	for _, code := range rowCodes {
		resolved, ok := cache[code]
		if !ok {
			resolved = s.resolveRoleCode(ctx, tenantID, code)
			cache[code] = resolved
		}
		if resolved.errMsg != "" {
			errs = append(errs, resolved.errMsg)
			continue
		}
		roleIDs = append(roleIDs, resolved.id)
		roleCodes = append(roleCodes, resolved.code)
	}
	if len(errs) > 0 {
		return nil, nil, errs
	}
	return roleIDs, roleCodes, nil
}

// resolveRoleCode 按编码解析角色：先找本租户角色，再回落到全平台角色（scope=all）。
func (s *UserService) resolveRoleCode(ctx context.Context, tenantID, code string) resolvedRole {
	role, err := s.roleRepo.GetByCode(ctx, tenantID, code)
	if err != nil {
		role, err = s.roleRepo.GetByCode(ctx, "", code)
	}
	if err != nil {
		return resolvedRole{errMsg: fmt.Sprintf("角色编码 '%s' 不存在", code)}
	}
	if role.Status == 0 {
		return resolvedRole{errMsg: fmt.Sprintf("角色 '%s' 已禁用", role.Code)}
	}
	if role.Scope != rbacModel.RoleScopeAll && role.Scope != rbacModel.RoleScopeTenant {
		return resolvedRole{errMsg: fmt.Sprintf("角色 '%s' 不允许分配给租户用户", role.Code)}
	}
	return resolvedRole{id: role.ID, code: role.Code}
}

// checkImportQuota 校验套餐剩余额度是否放得下整批导入。
func (s *UserService) checkImportQuota(ctx context.Context, tenantID string, count int) error {
	if s.quotaVerifier == nil {
		return nil
	}
	overLimit, users, limit, err := s.quotaVerifier.CheckUserLimit(ctx, tenantID)
	if err != nil {
		// 套餐到期 / 已达上限等语义复用单条创建的错误映射；映射不到时原样上抛
		if mapped := s.checkUserQuota(ctx, tenantID); mapped != nil {
			return mapped
		}
		return err
	}
	remaining := limit - int(users)
	// limit < 0 是套餐的「不限量」语义（-1）。此时 remaining 恒为负数，
	// 不能参与「剩余是否够」的比较，否则不限量的租户反而一行都导不进来。
	if limit < 0 && !overLimit {
		return nil
	}
	if overLimit || remaining < count {
		if remaining < 0 {
			remaining = 0
		}
		return newValidationErr("套餐用户数上限 %d，当前已用 %d，剩余可导入 %d 个，本次提交 %d 行", limit, users, remaining, count)
	}
	return nil
}

// generateInitialPassword 按密码策略生成随机初始密码。
func (s *UserService) generateInitialPassword() (string, error) {
	length := defaultInitialPasswordLength
	if s.passwordPolicy.Enabled {
		if want := s.passwordPolicy.MinLength + 2; want > length {
			length = want
		}
		if s.passwordPolicy.MaxLength > 0 && length > s.passwordPolicy.MaxLength {
			length = s.passwordPolicy.MaxLength
		}
	}
	if length > 32 {
		length = 32 // 与 dto.CreateUserReq 的 max=32 对齐，避免下发给用户后无法登录后修改
	}
	if length < 8 {
		length = 8
	}
	return security.GeneratePassword(length)
}

// sortedKeys 返回有序的 map 键，保证同一行内的错误顺序稳定（便于测试与阅读）。
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
