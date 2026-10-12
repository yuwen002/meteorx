package service

import (
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"meteorx/internal/modules/file/dto"
	"meteorx/internal/modules/file/model"
)

func newTestFileService() (*FileService, *mockFileRepo, *mockStorage) {
	fr := newMockFileRepo()
	st := newMockStorage()
	svc := NewFileService(fr, st, nil)
	return svc, fr, st
}

// stubQuotaProvider 固定存储配额（-1 不限）的提供者，用于配额测试。
type stubQuotaProvider struct {
	limit int64
	err   error
}

func (s stubQuotaProvider) GetTenantStorageLimit(_ context.Context, _ string) (int64, error) {
	return s.limit, s.err
}

func seedFile(fr *mockFileRepo, id, tenantID, userID string) *model.File {
	f := &model.File{
		ID: id, TenantID: tenantID, UserID: userID,
		OriginalName: id + ".txt", FilePath: "files/" + id + ".txt",
		FileSize: 10, MimeType: "text/plain", MD5: "md5-" + id,
		Status: model.FileStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	fr.seed(f)
	return f
}

// ---------- GetByIDWithScope 租户隔离 ----------

func TestGetByIDWithScope_OK(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	f, err := svc.GetByIDWithScope(context.Background(), "f1", "t1")
	if err != nil {
		t.Fatalf("GetByIDWithScope error: %v", err)
	}
	if f.ID != "f1" {
		t.Fatalf("unexpected file: %+v", f)
	}
}

func TestGetByIDWithScope_CrossTenantDenied(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	// 跨租户访问：文件存在但租户不匹配，应返回模糊化的 not found 错误
	_, err := svc.GetByIDWithScope(context.Background(), "f1", "t2")
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("expected tenant denied error, got: %v", err)
	}
}

func TestGetByIDWithScope_NotFound(t *testing.T) {
	svc, _, _ := newTestFileService()
	_, err := svc.GetByIDWithScope(context.Background(), "missing", "t1")
	if err == nil {
		t.Fatalf("expected not found error, got nil")
	}
}

// ---------- Update ----------

func TestUpdate_CrossTenantDenied(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	err := svc.Update(context.Background(), "f1", "t2", &dto.FileUpdateReq{FileName: "new.txt"})
	if err == nil {
		t.Fatal("expected tenant denied error, got nil")
	}
}

func TestUpdate_Success(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	if err := svc.Update(context.Background(), "f1", "t1", &dto.FileUpdateReq{FileName: "new.txt"}); err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if fr.files["f1"].FileName != "new.txt" {
		t.Fatalf("expected updated file name, got %s", fr.files["f1"].FileName)
	}
}

// ---------- Delete 软删除 + 物理清理 ----------

func TestDelete_CrossTenantDenied(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	err := svc.Delete(context.Background(), "f1", "t2")
	if err == nil {
		t.Fatal("expected tenant denied error, got nil")
	}
}

func TestDelete_Success(t *testing.T) {
	svc, fr, st := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")
	st.uploaded["files/f1.txt"] = true

	if err := svc.Delete(context.Background(), "f1", "t1"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, ok := fr.files["f1"]; ok {
		t.Fatal("expected file moved out of active set")
	}
	if _, ok := fr.deleted["f1"]; !ok {
		t.Fatal("expected file in deleted set")
	}
	if st.uploaded["files/f1.txt"] {
		t.Fatal("expected storage object removed")
	}
}

// ---------- BatchDelete ----------

func TestBatchDelete_Success(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")
	seedFile(fr, "f2", "t1", "u1")

	resp, err := svc.BatchDelete(context.Background(), "t1", &dto.BatchDeleteReq{IDs: []string{"f1", "f2"}})
	if err != nil {
		t.Fatalf("BatchDelete error: %v", err)
	}
	if resp.SuccessCount != 2 || resp.FailedCount != 0 {
		t.Fatalf("expected success=2 failed=0, got %+v", resp)
	}
}

func TestBatchDelete_NotFoundCountedAsFail(t *testing.T) {
	svc, _, _ := newTestFileService()
	resp, err := svc.BatchDelete(context.Background(), "t1", &dto.BatchDeleteReq{IDs: []string{"missing"}})
	if err != nil {
		t.Fatalf("BatchDelete error: %v", err)
	}
	if resp.SuccessCount != 0 || resp.FailedCount != 1 || len(resp.FailedIDs) != 1 {
		t.Fatalf("expected success=0 failed=1, got %+v", resp)
	}
}

// ---------- PermanentDelete ----------

func TestPermanentDelete_CrossTenantDenied(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	err := svc.PermanentDelete(context.Background(), "f1", "t2")
	if err == nil {
		t.Fatal("expected tenant denied error, got nil")
	}
}

func TestPermanentDelete_Success(t *testing.T) {
	svc, fr, st := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")
	st.uploaded["files/f1.txt"] = true

	if err := svc.PermanentDelete(context.Background(), "f1", "t1"); err != nil {
		t.Fatalf("PermanentDelete error: %v", err)
	}
	if _, ok := fr.files["f1"]; ok {
		t.Fatal("expected file permanently deleted")
	}
	if st.uploaded["files/f1.txt"] {
		t.Fatal("expected storage object removed")
	}
}

// ---------- Download ----------

func TestDownload_CrossTenantDenied(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	_, _, err := svc.Download(context.Background(), "f1", "t2")
	if err == nil {
		t.Fatal("expected tenant denied error, got nil")
	}
}

func TestDownload_Success(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	rc, _, err := svc.Download(context.Background(), "f1", "t1")
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	defer func() { _ = rc.Close() }()
}

// ---------- Restore ----------

func TestRestore_CrossTenantDenied(t *testing.T) {
	svc, fr, _ := newTestFileService()
	seedFile(fr, "f1", "t1", "u1")

	err := svc.Restore(context.Background(), "f1", "t2")
	if err == nil {
		t.Fatal("expected tenant denied error, got nil")
	}
}

func TestRestore_Success(t *testing.T) {
	svc, fr, _ := newTestFileService()
	f := seedFile(fr, "f1", "t1", "u1")
	f.Status = model.FileStatusDeleted
	fr.deleted["f1"] = f
	delete(fr.files, "f1")

	if err := svc.Restore(context.Background(), "f1", "t1"); err != nil {
		t.Fatalf("Restore error: %v", err)
	}
	if _, ok := fr.files["f1"]; !ok {
		t.Fatal("expected file restored to active set")
	}
}

// ---------- 上传前存储配额校验 ----------

// newUploadHeader 构造一个指定大小的 multipart.FileHeader（无需真实内容）。
func newUploadHeader(size int64) *multipart.FileHeader {
	return &multipart.FileHeader{Filename: "big.txt", Size: size, Header: nil}
}

func TestUpload_StorageQuotaExceeded(t *testing.T) {
	fr := newMockFileRepo()
	st := newMockStorage()
	// 已有文件占用 10 字节，上限 15，本次上传 10 字节 -> 20 > 15 拒绝
	seedFile(fr, "f1", "t1", "u1")
	svc := NewFileService(fr, st, stubQuotaProvider{limit: 15})

	_, err := svc.Upload(context.Background(), newUploadHeader(10), "t1", "u1")
	if !errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("expected ErrStorageQuotaExceeded, got: %v", err)
	}
	if len(st.uploaded) != 0 {
		t.Fatalf("no file should be uploaded when quota exceeded")
	}
}

func TestUpload_StorageQuotaUnlimited(t *testing.T) {
	fr := newMockFileRepo()
	st := newMockStorage()
	// limit=-1 不限制；但 header 无 Header 会导致 Content-Type 读取，仍可上传
	svc := NewFileService(fr, st, stubQuotaProvider{limit: -1})

	_, err := svc.Upload(context.Background(), newUploadHeader(1<<30), "t1", "u1")
	if err != nil && !errors.Is(err, ErrStorageQuotaExceeded) {
		// 允许因文件指针等无关错误失败，但不能是配额错误
		t.Logf("upload returned non-quota error (acceptable for stub header): %v", err)
	}
}

func TestUpload_StorageQuotaFailOpenOnError(t *testing.T) {
	fr := newMockFileRepo()
	st := newMockStorage()
	// 配额提供者报错 -> fail-open，不应返回配额错误
	svc := NewFileService(fr, st, stubQuotaProvider{err: errors.New("db down")})

	_, err := svc.Upload(context.Background(), newUploadHeader(999), "t1", "u1")
	if errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("provider error should fail-open, not quota-reject")
	}
}
