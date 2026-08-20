package model

import (
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// File 文件模型
type File struct {
	ID          string    `gorm:"primaryKey;type:varchar(26)" json:"id"`
	TenantID    string    `gorm:"type:varchar(26);index;not null" json:"tenant_id"`
	UserID      string    `gorm:"type:varchar(26);index;not null" json:"user_id"`
	FileName    string    `gorm:"type:varchar(255);not null" json:"file_name"`
	OriginalName string   `gorm:"type:varchar(255);not null" json:"original_name"`
	FilePath    string    `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize    int64     `gorm:"not null" json:"file_size"`
	MimeType    string    `gorm:"type:varchar(100);not null" json:"mime_type"`
	FileType    string    `gorm:"type:varchar(50);not null" json:"file_type"` // image, document, video, audio, other
	StorageType string    `gorm:"type:varchar(50);default:'local'" json:"storage_type"` // local, oss, s3
	MD5         string    `gorm:"type:varchar(32);index" json:"md5"`
	Status      int       `gorm:"type:tinyint;default:1" json:"status"` // 1: active, 0: deleted
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (File) TableName() string {
	return "files"
}

// BeforeCreate GORM hook - 在创建前生成ULID
func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = ulid.Make().String()
	}
	return nil
}
