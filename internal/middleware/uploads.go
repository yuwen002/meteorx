package middleware

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"meteorx/internal/common/signedurl"
)

// SignedUploadsHandler 提供 /uploads/* 文件访问（替代裸 http.FileServer）：
//  1. 仅接受"单段文件名"，拒绝目录列举、子路径与路径穿越；
//  2. 必须携带由后端签发的短时效签名参数（e=过期时间&s=HMAC）；
//  3. secrets 为签名密钥链（首项当前签发密钥，后续为轮换期保留的历史密钥），
//     任一密钥验证通过即放行；secrets 为空时降级为完全公开（便于本地调试/未启用签名场景）。
func SignedUploadsHandler(uploadPath string, secrets []string) http.Handler {
	baseDir := filepath.Clean(uploadPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 取出 /uploads 之后的文件路径
		name := strings.TrimPrefix(r.URL.Path, "/uploads")
		name = strings.TrimPrefix(name, "/")

		// 安全校验：必须为单段文件名（ULID.扩展名），杜绝列目录 / 路径穿越
		if name == "" || name == "." ||
			strings.ContainsAny(name, `/\`) ||
			strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}

		// 签名校验（未配置密钥时跳过，保持原公开行为）
		if len(secrets) > 0 {
			q := r.URL.Query()
			signature := q.Get(signedurl.QuerySig)
			expiresStr := q.Get(signedurl.QueryExpires)
			expires, err := strconv.ParseInt(expiresStr, 10, 64)
			if err != nil || !signedurl.VerifyAny(name, secrets, expires, signature) ||
				time.Now().Unix() > expires {
				http.Error(w, "链接无效或已过期，请通过文件接口获取最新链接", http.StatusForbidden)
				return
			}
		}

		// 直接提供文件（避免列出目录）
		http.ServeFile(w, r, filepath.Join(baseDir, name))
	})
}
