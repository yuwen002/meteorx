package tenantctx

import (
	"context"
	"testing"

	"meteorx/internal/common/contextx"

	"github.com/stretchr/testify/assert"
)

func TestTenantContext(t *testing.T) {
	t.Run("tenant context missing returns error", func(t *testing.T) {
		ctx := context.Background()
		info, err := From(ctx)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("tenant context has valid tenant", func(t *testing.T) {
		ctx := contextx.SetVars(context.Background(), "tenant-001", "user-001", []string{"admin"})
		info, err := From(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "tenant-001", info.ID)
		assert.Equal(t, "user-001", info.UserID)
		assert.False(t, info.IsMaster)
	})

	t.Run("master admin is detected", func(t *testing.T) {
		ctx := contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", []string{"superadmin"})
		info, err := From(ctx)
		assert.NoError(t, err)
		assert.True(t, info.IsMaster)
	})
}

func TestCrossTenantAccess(t *testing.T) {
	t.Run("regular tenant cannot access different tenant", func(t *testing.T) {
		ctx := contextx.SetVars(context.Background(), "tenant-001", "user-001", []string{"admin"})
		assert.False(t, CanAccessTenant(ctx, "tenant-002"))
	})

	t.Run("regular tenant can access own tenant", func(t *testing.T) {
		ctx := contextx.SetVars(context.Background(), "tenant-001", "user-001", []string{"admin"})
		assert.True(t, CanAccessTenant(ctx, "tenant-001"))
	})

	t.Run("master admin can access any tenant", func(t *testing.T) {
		ctx := contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", []string{"superadmin"})
		assert.True(t, CanAccessTenant(ctx, "tenant-001"))
		assert.True(t, CanAccessTenant(ctx, "tenant-002"))
	})
}

func TestRequireTenant(t *testing.T) {
	t.Run("missing tenant returns error", func(t *testing.T) {
		ctx := context.Background()
		info, err := RequireTenant(ctx)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("valid tenant returns info", func(t *testing.T) {
		ctx := contextx.SetVars(context.Background(), "tenant-001", "user-001", []string{"admin"})
		info, err := RequireTenant(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "tenant-001", info.ID)
	})
}