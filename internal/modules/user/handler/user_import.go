package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/user/importer"
	"meteorx/internal/modules/user/service"
	"meteorx/internal/pkg/apperrors"
)

const (
	// maxImportFormMemory multipart 表单在内存中缓存的字节数，超出部分由标准库落到临时文件。
	maxImportFormMemory = 8 << 20
	// importFormOverhead 除文件本体外的表单边界与普通字段占用的余量。
	importFormOverhead = 1 << 20
)

// ImportUsers 租户内批量导入用户
// POST /api/v1/users/import
// 权限码由 AutoRequirePermission 推导为 user:create（与逐个新建保持同一门槛）。
func (h *UserHandler) ImportUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}
	// 系统管理员的登录态租户是虚拟租户 SYSTEM_ROOT，不是真实租户：
	// 走本接口只会把账号导入到这个谁也登录不进去的租户里，跨租户导入必须走 /admin/tenant-users/import。
	if tenantID == contextx.SystemTenantID {
		response.Fail(w, http.StatusBadRequest, "系统管理员请使用跨租户接口为指定租户导入用户")
		return
	}
	h.handleImport(w, r, tenantID)
}

// AdminImportTenantUsers 平台管理员为指定租户批量导入用户
// POST /api/v1/admin/tenant-users/import
// 租户ID 取自表单字段 tenant_id；权限码推导为 admin:tenant_user:create。
func (h *UserHandler) AdminImportTenantUsers(w http.ResponseWriter, r *http.Request) {
	h.handleImport(w, r, "")
}

// handleImport 两种导入入口的共用流程：解析表单 → 解析文件 → 交给服务层。
// tenantID 为空时从表单的 tenant_id 字段读取（管理端入口）。
func (h *UserHandler) handleImport(w http.ResponseWriter, r *http.Request, tenantID string) {
	filename, data, roleIDs, err := readImportForm(w, r)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if tenantID == "" {
		tenantID = strings.TrimSpace(r.FormValue("tenant_id"))
		if tenantID == "" {
			response.BadRequest(w, "租户ID不能为空")
			return
		}
		// 角色ID 属于某个具体租户，跨租户使用会把别的租户的角色挂到目标租户用户上，
		// 因此管理端入口只认文件里的角色编码，忽略表单的 role_ids。
		roleIDs = nil
	}

	rows, fileWarnings, err := importer.Parse(filename, data)
	if err != nil {
		// importer 的错误都是"整份文件不可用"级别，且文案面向使用者
		response.BadRequest(w, err.Error())
		return
	}

	result, rowErrs, err := h.svc.ImportUsers(r.Context(), tenantID, rows, roleIDs)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.BadRequest(w, err.Error())
			return
		}
		response.Fail(w, http.StatusInternalServerError, "批量导入失败")
		return
	}

	// 全量事务语义：任一行不合法则一条都没写，把逐行明细回传给前端定位
	if len(rowErrs) > 0 {
		response.WriteJSON(w, http.StatusBadRequest, response.ErrorResponse{
			Code:    string(apperrors.ErrInvalidParam),
			Message: fmt.Sprintf("导入校验未通过，共 %d 处问题，未写入任何用户", len(rowErrs)),
			Details: rowErrs,
		})
		return
	}

	if len(fileWarnings) > 0 {
		result.Warnings = append(result.Warnings, fileWarnings...)
	}
	response.Success(w, result)
}

// readImportForm 读取 multipart 表单，返回文件名、文件内容与默认角色ID。
func readImportForm(w http.ResponseWriter, r *http.Request) (string, []byte, []string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, importer.MaxFileSize+importFormOverhead)
	if err := r.ParseMultipartForm(maxImportFormMemory); err != nil {
		return "", nil, nil, fmt.Errorf("文件解析失败：请确认上传的是 CSV 或 XLSX 文件，且不超过 %d MB", importer.MaxFileSize>>20)
	}
	if r.MultipartForm != nil {
		// 超出内存阈值的部分已落临时文件，处理完立刻清理
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return "", nil, nil, errors.New("未找到上传文件（表单字段名应为 file）")
	}
	defer func() { _ = file.Close() }()

	// 多读一个字节用于判断是否越过上限（MaxBytesReader 同时兜住了请求体）
	data, err := io.ReadAll(io.LimitReader(file, importer.MaxFileSize+1))
	if err != nil {
		return "", nil, nil, errors.New("读取上传文件失败")
	}
	if len(data) > importer.MaxFileSize {
		return "", nil, nil, fmt.Errorf("文件大小超过 %d MB 限制", importer.MaxFileSize>>20)
	}
	return header.Filename, data, parseRoleIDs(r), nil
}

// parseRoleIDs 读取表单中的默认角色：支持重复字段与逗号分隔两种写法。
func parseRoleIDs(r *http.Request) []string {
	if r.MultipartForm == nil {
		return nil
	}
	raw := r.MultipartForm.Value["role_ids"]
	if len(raw) == 0 {
		return nil
	}

	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			id := strings.TrimSpace(part)
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
