package idgen

import (
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestNew_ReturnsValidULID(t *testing.T) {
	id := New()
	assert.Len(t, id, 26)
	if _, err := ulid.Parse(id); err != nil {
		t.Fatalf("generated id %q is not a valid ULID: %v", id, err)
	}
}

func TestNew_Unique(t *testing.T) {
	const n = 500
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		id := New()
		if seen[id] {
			t.Fatalf("duplicate id generated: %s", id)
		}
		seen[id] = true
	}
}

// TestNew_TimeOrderedAcrossTimestamps 验证跨时间戳生成时字典序递增（ULID 按时间有序）
func TestNew_TimeOrderedAcrossTimestamps(t *testing.T) {
	first := New()
	time.Sleep(30 * time.Millisecond)
	second := New()
	if second <= first {
		t.Fatalf("expected %s > %s (later ULID should sort after earlier)", second, first)
	}
}

func TestNewULID_AliasOfNew(t *testing.T) {
	assert.Len(t, NewULID(), 26)
}

func TestNewUUID_Format(t *testing.T) {
	u := NewUUID()
	assert.Len(t, u, 36)
	parts := strings.Split(u, "-")
	assert.Len(t, parts, 5)
	// UUID v4: 版本位为 4
	assert.Equal(t, "4", parts[2][:1])
}

func TestParse_ValidRoundTrip(t *testing.T) {
	id := New()
	parsed, err := Parse(id)
	assert.NoError(t, err)
	assert.Equal(t, id, parsed)
}

func TestParse_InvalidInput(t *testing.T) {
	_, err := Parse("not-a-valid-ulid-!!")
	assert.Error(t, err)
}

func TestMustParse_Valid(t *testing.T) {
	id := New()
	assert.Equal(t, id, MustParse(id))
}

func TestMustParse_PanicsOnInvalid(t *testing.T) {
	assert.Panics(t, func() { MustParse("garbage") })
}
