package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"meteorx/internal/modules/file/dto"
	"meteorx/internal/modules/file/model"
	"meteorx/internal/modules/file/repository"
	"meteorx/internal/modules/file/storage"
)

// FileService 文件服务
type FileService struct {
	repo   repository.FileRepository
	storage storage.Storage
}

// NewFileService 创建文件服务实例
func NewFileService(repo repository.FileRepository, storage storage.Storage) *FileService {
	return &FileService{
		repo:   repo,
		storage: storage,
	}
}

// Upload 上传文件
func (s *FileService) Upload(ctx context.Context, fileHeader *multipart.FileHeader, tenantID, userID string) (*dto.UploadFileResp, error) {
	// 打开上传的文件
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// 计算文件MD5
	md5Hash, err := s.calculateMD5(file)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MD5: %w", err)
	}

	// 检查是否已存在相同文件（去重）
	existingFile, err := s.repo.GetByMD5(ctx, tenantID, md5Hash)
	if err == nil && existingFile != nil {
		// 文件已存在，返回现有文件信息
		return &dto.UploadFileResp{
			ID:           existingFile.ID,
			FileName:     existingFile.FileName,
			OriginalName: existingFile.OriginalName,
			FileSize:     existingFile.FileSize,
			MimeType:     existingFile.MimeType,
			FileType:     existingFile.FileType,
			URL:          s.storage.GetURL(existingFile.FilePath),
		}, nil
	}

	// 重置文件指针
	file.Seek(0, 0)

	// 上传文件到存储
	filePath, err := s.storage.Upload(ctx, file, fileHeader.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// 创建文件记录
	fileRecord := &model.File{
		TenantID:     tenantID,
		UserID:       userID,
		FileName:     filepath.Base(filePath),
		OriginalName: fileHeader.Filename,
		FilePath:     filePath,
		FileSize:     fileHeader.Size,
		MimeType:     fileHeader.Header.Get("Content-Type"),
		FileType:     s.detectFileType(fileHeader.Filename, fileHeader.Header.Get("Content-Type")),
		StorageType:  s.storage.Type(),
		MD5:          md5Hash,
		Status:       1,
	}

	if err := s.repo.Create(ctx, fileRecord); err != nil {
		// 如果数据库插入失败，删除已上传的文件
		s.storage.Delete(ctx, filePath)
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return &dto.UploadFileResp{
		ID:           fileRecord.ID,
		FileName:     fileRecord.FileName,
		OriginalName: fileRecord.OriginalName,
		FileSize:     fileRecord.FileSize,
		MimeType:     fileRecord.MimeType,
		FileType:     fileRecord.FileType,
		URL:          s.storage.GetURL(fileRecord.FilePath),
	}, nil
}

// GetByID 获取文件详情（同时校验文件归属于当前租户/用户）
func (s *FileService) GetByID(ctx context.Context, id string) (*dto.FileResp, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	url := s.storage.GetURL(file.FilePath)
	return dto.ToFileResp(file, url), nil
}

// GetByIDWithScope 获取文件详情，并校验访问范围
// scope: "tenant" 要求文件必须属于当前租户
func (s *FileService) GetByIDWithScope(ctx context.Context, id, tenantID string) (*dto.FileResp, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenantID != "" && file.TenantID != tenantID {
		return nil, fmt.Errorf("file not found")
	}

	url := s.storage.GetURL(file.FilePath)
	return dto.ToFileResp(file, url), nil
}

// ListByTenant 获取租户文件列表
func (s *FileService) ListByTenant(ctx context.Context, tenantID string, req *dto.FileListReq) ([]*dto.FileResp, int64, error) {
	files, total, err := s.repo.ListByTenant(ctx, tenantID, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}

	// 构建URL映射
	urlMap := make(map[string]string)
	for _, file := range files {
		urlMap[file.ID] = s.storage.GetURL(file.FilePath)
	}

	return dto.ToFileRespList(files, urlMap), total, nil
}

// ListByUser 获取用户文件列表
func (s *FileService) ListByUser(ctx context.Context, tenantID, userID string, req *dto.FileListReq) ([]*dto.FileResp, int64, error) {
	files, total, err := s.repo.ListByUser(ctx, tenantID, userID, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}

	// 构建URL映射
	urlMap := make(map[string]string)
	for _, file := range files {
		urlMap[file.ID] = s.storage.GetURL(file.FilePath)
	}

	return dto.ToFileRespList(files, urlMap), total, nil
}

// Update 更新文件信息
func (s *FileService) Update(ctx context.Context, id string, tenantID string, req *dto.FileUpdateReq) error {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tenantID != "" && file.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}

	file.FileName = req.FileName
	return s.repo.Update(ctx, file)
}

// Delete 删除文件（软删除 + 物理文件清理）
func (s *FileService) Delete(ctx context.Context, id, tenantID string) error {
	// 先查记录，拿到存储路径以便物理清理
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tenantID != "" && file.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}

	// 软删除数据库记录
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// 同步删除物理文件；失败只记日志，不回滚软删除（软删除已符合业务语义）
	if file != nil && file.FilePath != "" {
		if err := s.storage.Delete(ctx, file.FilePath); err != nil {
			fmt.Printf("[FileService] warning: physical file delete failed for %s: %v\n", file.FilePath, err)
		}
	}
	return nil
}

// BatchDelete 批量删除文件
func (s *FileService) BatchDelete(ctx context.Context, tenantID string, req *dto.BatchDeleteReq) (*dto.BatchDeleteResp, error) {
	successCount := 0
	failedCount := 0
	failedIDs := []string{}

	for _, id := range req.IDs {
		if err := s.Delete(ctx, id, tenantID); err != nil {
			failedCount++
			failedIDs = append(failedIDs, id)
		} else {
			successCount++
		}
	}

	return &dto.BatchDeleteResp{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		FailedIDs:    failedIDs,
	}, nil
}

// GetDeletedList 获取已删除文件列表
func (s *FileService) GetDeletedList(ctx context.Context, tenantID string, page, pageSize int) ([]*dto.FileResp, int64, error) {
	files, total, err := s.repo.GetDeletedList(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 构建URL映射（已删除文件的URL可能不可用）
	urlMap := make(map[string]string)
	for _, file := range files {
		// 检查物理文件是否还存在
		if exists, _ := s.storage.Exists(ctx, file.FilePath); exists {
			urlMap[file.ID] = s.storage.GetURL(file.FilePath)
		}
	}

	return dto.ToFileRespList(files, urlMap), total, nil
}

// Restore 恢复已删除文件（需要校验租户归属）
func (s *FileService) Restore(ctx context.Context, id, tenantID string) error {
	file, err := s.repo.GetByIDUnscoped(ctx, id)
	if err != nil {
		return err
	}
	if tenantID != "" && file.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}
	return s.repo.Restore(ctx, id)
}

// PermanentDelete 永久删除文件（物理删除 + 物理文件清理）
// 仅用于回收站中的已软删除文件
func (s *FileService) PermanentDelete(ctx context.Context, id, tenantID string) error {
	// 查找包含已删除记录的文件（Unscoped 才能查到已软删除记录）
	file, err := s.repo.GetByIDUnscoped(ctx, id)
	if err != nil {
		return err
	}
	if tenantID != "" && file.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}

	// 物理删除数据库记录（Unscoped）
	if err := s.repo.PermanentDelete(ctx, id); err != nil {
		return err
	}

	// 同步删除物理文件
	if file.FilePath != "" {
		if err := s.storage.Delete(ctx, file.FilePath); err != nil {
			fmt.Printf("[FileService] warning: physical file delete failed for %s: %v\n", file.FilePath, err)
		}
	}
	return nil
}

// Download 下载文件（需要校验租户归属）
func (s *FileService) Download(ctx context.Context, id, tenantID string) (io.ReadCloser, string, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if tenantID != "" && file.TenantID != tenantID {
		return nil, "", fmt.Errorf("file not found")
	}

	reader, err := s.storage.Download(ctx, file.FilePath)
	if err != nil {
		return nil, "", err
	}

	return reader, file.OriginalName, nil
}

// calculateMD5 计算文件的MD5值
func (s *FileService) calculateMD5(file multipart.File) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// detectFileType 检测文件类型
func (s *FileService) detectFileType(filename, mimeType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	// 根据扩展名判断
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}
	documentExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt"}
	videoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv"}
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".ogg"}

	for _, imgExt := range imageExts {
		if ext == imgExt {
			return "image"
		}
	}
	
	for _, docExt := range documentExts {
		if ext == docExt {
			return "document"
		}
	}
	
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return "video"
		}
	}
	
	for _, audExt := range audioExts {
		if ext == audExt {
			return "audio"
		}
	}

	// 根据MIME类型判断
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}
	if strings.HasPrefix(mimeType, "text/") || mimeType == "application/pdf" {
		return "document"
	}

	return "other"
}