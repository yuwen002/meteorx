package storage

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// TestIsOSSNoSuchKey 校验「对象不存在」识别：404 直接命中、包装后仍可识别、
// 非 404 与普通错误均不误判。保证 OSS Delete 的幂等容错只对 NoSuchKey 生效。
func TestIsOSSNoSuchKey(t *testing.T) {
	notFound := &oss.ServiceError{StatusCode: http.StatusNotFound, Code: "NoSuchKey"}
	if !isOSSNoSuchKey(notFound) {
		t.Fatal("404 ServiceError should be treated as NoSuchKey")
	}
	if !isOSSNoSuchKey(fmt.Errorf("oss delete: %w", notFound)) {
		t.Fatal("wrapped 404 should still be detected")
	}
	// 值形态（Error() 为值接收者）也应能识别
	if !isOSSNoSuchKey(oss.ServiceError{StatusCode: http.StatusNotFound}) {
		t.Fatal("value-form 404 should be detected")
	}
	if isOSSNoSuchKey(&oss.ServiceError{StatusCode: http.StatusForbidden, Code: "AccessDenied"}) {
		t.Fatal("non-404 must not be treated as NoSuchKey")
	}
	if isOSSNoSuchKey(errors.New("boom")) {
		t.Fatal("plain error must not be treated as NoSuchKey")
	}
}
