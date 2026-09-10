package testutil

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// RouterOption 路由配置选项
type RouterOption func(r chi.Router)

// WithMiddleware 添加中间件
func WithMiddleware(middlewares ...func(http.Handler) http.Handler) RouterOption {
	return func(r chi.Router) {
		for _, m := range middlewares {
			r.Use(m)
		}
	}
}

// WithRoutes 添加路由
func WithRoutes(routes func(r chi.Router)) RouterOption {
	return func(r chi.Router) {
		routes(r)
	}
}

// NewTestRouter 创建测试用路由器
func NewTestRouter(opts ...RouterOption) chi.Router {
	r := chi.NewRouter()
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// TestContext 创建包含认证信息的测试上下文
func TestContext(tenantID, userID string, roles []string) context.Context {
	ctx := context.Background()
	return contextx.SetVars(ctx, tenantID, userID, roles)
}

// ExecuteRequest 执行 HTTP 测试请求并返回响应
func ExecuteRequest(r http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ExecuteRequestWithContext 执行带认证上下文的 HTTP 测试请求
func ExecuteRequestWithContext(r http.Handler, method, path string, body io.Reader, ctx context.Context) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ParseResponse 解析 JSON 响应
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, target))
}

// AssertStatusCode 断言 HTTP 状态码
func AssertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	require.Equal(t, expected, w.Code, "HTTP 状态码不匹配，响应体: %s", w.Body.String())
}

// NewJSONBody 将结构体编码为 JSON 请求体
func NewJSONBody(t *testing.T, v interface{}) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return io.NopCloser(bytesReader(b))
}

type bytesReader struct {
	*bytesReaderImpl
}

type bytesReaderImpl struct {
	data []byte
	off  int
}

func (r *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

func (r *bytesReaderImpl) Close() error {
	return nil
}

func bytesReader(data []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: data, off: 0}
}