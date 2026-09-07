package contextx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetVars_AndGetters(t *testing.T) {
	ctx := SetVars(context.Background(), "tenant-1", "user-1", []string{"admin", "viewer"})

	assert.Equal(t, "tenant-1", GetTenantID(ctx))
	assert.Equal(t, "user-1", GetUserID(ctx))
	assert.Equal(t, []string{"admin", "viewer"}, GetRoles(ctx))
}

func TestGetters_OnEmptyContext(t *testing.T) {
	assert.Empty(t, GetTenantID(context.Background()))
	assert.Empty(t, GetUserID(context.Background()))
	assert.Equal(t, []string{}, GetRoles(context.Background()))
}

func TestHasRole(t *testing.T) {
	ctx := SetVars(context.Background(), "t", "u", []string{"admin"})
	assert.True(t, HasRole(ctx, "admin"))
	assert.False(t, HasRole(ctx, "owner"))
	assert.False(t, HasRole(context.Background(), "admin"))
}

func TestSetVars_PreservesOtherValues(t *testing.T) {
	type markerKey struct{}
	base := context.WithValue(context.Background(), markerKey{}, "kept")
	ctx := SetVars(base, "t", "u", nil)

	assert.Equal(t, "kept", ctx.Value(markerKey{}))
	assert.Equal(t, "t", GetTenantID(ctx))
}
