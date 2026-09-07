package uuid

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerate_ValidUUIDv4(t *testing.T) {
	id := Generate()
	parsed, err := uuid.Parse(id)
	assert.NoError(t, err)
	assert.Equal(t, uuid.Version(4), parsed.Version())
	parts := strings.Split(id, "-")
	assert.Len(t, parts, 5)
}

func TestGenerate_Unique(t *testing.T) {
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		id := Generate()
		if seen[id] {
			t.Fatalf("duplicate uuid %s", id)
		}
		seen[id] = true
	}
}
