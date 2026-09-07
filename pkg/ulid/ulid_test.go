package ulid

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestGenerate_ReturnsValidULID(t *testing.T) {
	id := Generate()
	assert.Len(t, id, 26)
	if _, err := ulid.Parse(id); err != nil {
		t.Fatalf("invalid ulid %q: %v", id, err)
	}
}

func TestGenerate_Unique(t *testing.T) {
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		id := Generate()
		if seen[id] {
			t.Fatalf("duplicate ulid %s", id)
		}
		seen[id] = true
	}
}

func TestGenerate_TimeOrderedAcrossTimestamps(t *testing.T) {
	a := Generate()
	time.Sleep(30 * time.Millisecond)
	b := Generate()
	if b <= a {
		t.Fatalf("expected %s > %s", b, a)
	}
}
