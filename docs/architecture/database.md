# Database Architecture

## Current State (Baseline)

### ORM: GORM

### Connection Management
- Single DB connection pool managed in `config`
- AutoMigrate called at startup for all modules

### ID Generation
- ULID (Crockford's Base32, 26 chars) via `pkg/ulid`
- Some UUID usage in older modules via `pkg/uuid`

### Models
- All models use GORM struct tags
- Soft delete via `DeletedAt *time.Time`
- Timestamps via `CreatedAt time.Time`, `UpdatedAt time.Time`

### Migration Order
1. User/Auth → Tenant → RBAC → Plan → Audit → File → Notification → Wiki

### Known Gaps

- [ ] Mixed ID strategies (ULID + UUID)
- [ ] No transaction helper (manual Begin/Commit)
- [ ] No repository interface enforcement for transactions
- [ ] No connection pooling configuration exposed

## Target State (P0-5, P1-1)

```go
// Transaction helper
db.WithTx(ctx, func(tx *gorm.DB) error {
    // all operations within transaction
})

// Unified ID generation
id := idgen.New()  // always ULID
```