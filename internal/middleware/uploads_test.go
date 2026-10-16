package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"meteorx/internal/common/signedurl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignedUploadsHandler 真实校验 /uploads/* 签名下载的安全属性：
// 合法签名放行、篡改签名拒绝、过期拒绝、无签名（已配置密钥时）拒绝、目录穿越拒绝、密钥缺失时降级公开。
func TestSignedUploadsHandler(t *testing.T) {
	// 准备一个临时上传目录与文件
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.pdf"), []byte("TOP-SECRET-CONTENT"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("plain"), 0o600))

	const secret = "test-upload-sign-key"
	handler := SignedUploadsHandler(dir, []string{secret})

	sign := func(name string, expires int64) string {
		return signedurl.Sign(name, secret, expires)
	}
	urlFor := func(name string, expires int64) string {
		return "/uploads/" + name + "?" + signedurl.QueryExpires + "=" + strconv.FormatInt(expires, 10) + "&" + signedurl.QuerySig + "=" + sign(name, expires)
	}

	t.Run("合法签名放行并返回文件内容", func(t *testing.T) {
		exp := signedurl.Expires(signedurl.DefaultTTL)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, urlFor("secret.pdf", exp), nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "TOP-SECRET-CONTENT", w.Body.String())
	})

	t.Run("篡改签名被拒", func(t *testing.T) {
		exp := signedurl.Expires(signedurl.DefaultTTL)
		badURL := "/uploads/secret.pdf?" + signedurl.QueryExpires + "=" + strconv.FormatInt(exp, 10) + "&" + signedurl.QuerySig + "=deadbeef"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, badURL, nil))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("过期签名被拒", func(t *testing.T) {
		exp := signedurl.Expires(-time.Hour) // 已过期
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, urlFor("secret.pdf", exp), nil))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("已配置密钥但缺少签名被拒", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/secret.pdf", nil))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("目录穿越/子路径被拒（单段文件名约束）", func(t *testing.T) {
		// 原始 HTTP 路径含斜杠，handler 取出 name 后检测到分隔符 -> 404
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/foo/../secret.pdf", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("未配置密钥时降级为完全公开", func(t *testing.T) {
		openHandler := SignedUploadsHandler(dir, nil)
		w := httptest.NewRecorder()
		openHandler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uploads/ok.txt", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "plain", w.Body.String())
	})
}

// TestSignedUploadsHandlerRotationKeyChain 验证签名密钥平滑轮换：切换到新密钥后，
// 存量旧密钥签发的链接在「新旧共链」期间仍可访问；从链中移除旧密钥后旧链接被拒（轮换宽限期结束）。
func TestSignedUploadsHandlerRotationKeyChain(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "demo.txt"), []byte("rotation demo"), 0o600))

	const oldKey = "old-upload-sign-key"
	const newKey = "new-upload-sign-key"

	urlFor := func(secret string) string {
		exp := signedurl.Expires(time.Minute)
		sig := signedurl.Sign("demo.txt", secret, exp)
		return "/uploads/demo.txt?" + signedurl.QueryExpires + "=" + strconv.FormatInt(exp, 10) + "&" + signedurl.QuerySig + "=" + sig
	}

	t.Run("共链期旧密钥链接放行", func(t *testing.T) {
		rotated := SignedUploadsHandler(dir, []string{newKey, oldKey})
		w := httptest.NewRecorder()
		rotated.ServeHTTP(w, httptest.NewRequest(http.MethodGet, urlFor(oldKey), nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("新密钥签发链接放行", func(t *testing.T) {
		rotated := SignedUploadsHandler(dir, []string{newKey, oldKey})
		w := httptest.NewRecorder()
		rotated.ServeHTTP(w, httptest.NewRequest(http.MethodGet, urlFor(newKey), nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("移除旧密钥后旧链接被拒", func(t *testing.T) {
		onlyNew := SignedUploadsHandler(dir, []string{newKey})
		w := httptest.NewRecorder()
		onlyNew.ServeHTTP(w, httptest.NewRequest(http.MethodGet, urlFor(oldKey), nil))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
