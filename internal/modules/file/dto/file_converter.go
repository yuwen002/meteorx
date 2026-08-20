package dto

import (
	"meteorx/internal/modules/file/model"
	"time"
)

// ToFileResp 将模型转换为响应DTO
func ToFileResp(file *model.File, url string) *FileResp {
	return &FileResp{
		ID:           file.ID,
		TenantID:     file.TenantID,
		UserID:       file.UserID,
		FileName:     file.FileName,
		OriginalName: file.OriginalName,
		FileSize:     file.FileSize,
		MimeType:     file.MimeType,
		FileType:     file.FileType,
		URL:          url,
		MD5:          file.MD5,
		Status:       file.Status,
		CreatedAt:    file.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    file.UpdatedAt.Format(time.RFC3339),
	}
}

// ToFileRespList 将模型列表转换为响应DTO列表
func ToFileRespList(files []*model.File, urlMap map[string]string) []*FileResp {
	resps := make([]*FileResp, len(files))
	for i, file := range files {
		url := ""
		if urlMap != nil {
			url = urlMap[file.ID]
		}
		resps[i] = ToFileResp(file, url)
	}
	return resps
}
