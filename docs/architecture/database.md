# Database Architecture

## Overview

GORM-based ORM with unified ID generation, transaction management, and tenant-scoped queries.

## ORM: GORM

### Connection Management

```go
// Bootstrap handles connection
// internal/bootstrap/database.go
func InitDatabase(cfg *config.Config) (*gorm.DB, error) {
    // MySQL connection with connection pooling
    db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{...})
    // AutoMigrate all module models
    // Ping check on startup
}
```

### Connection Pool

```go
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)
```

## ID Generation

### Package: `pkg/idgen`

Unified ID generation using ULID (Universally Unique Lexicographically Sortable Identifier):

```go
// Always use idgen.New() for business IDs
id := idgen.New()  // 26-character ULID, e.g., 01H7K3N5P8R2S4T6V8W0X2Y4Z6

// Parse existing IDs
ulid, err := idgen.Parse("01H7K3N5P8R2S4T6V8W0X2Y4Z6")
```

### Why ULID?

| Feature | ULID | UUID |
|---------|------|------|
| Length | 26 chars | 36 chars (with hyphens) |
| Sortable | Yes (time-ordered) | No |
| Encoding | Crockford Base32 | Hex + hyphens |
| Collision | Very low | Very low |
| DB Index | Efficient | Less efficient |

### ID Rule

```
ALL business entities → idgen.New() → ULID
NEVER use uuid.New() or ulid.Generate() directly
```

Migration: old `pkg/ulid` and `pkg/uuid` packages are maintained for backward compatibility only. New code MUST use `pkg/idgen`.

## Transaction Management

### Package: `internal/pkg/db`

```go
// Create transaction manager
txManager := db.NewTxManager(db)

// Execute operations within a transaction
err := txManager.WithTx(ctx, func(ctx context.Context, tx *gorm.DB) error {
    // All repository operations should use tx-aware context
    // Repository.GetDB(ctx) returns tx when inside transaction
    repo.SetTx(tx)
    _, err := repo.Create(ctx, entity)
    if err != nil {
        return err  // triggers rollback
    }
    return nil  // triggers commit
})
```

### Repository Pattern with Transaction Support

```go
type BaseRepository struct {
    db *gorm.DB
}

func (r *BaseRepository) SetTx(tx *gorm.DB) {
    r.db = tx
}

func (r *BaseRepository) GetDB(ctx context.Context) *gorm.DB {
    if tx := db.GetTx(ctx); tx != nil {
        return tx
    }
    return r.db
}
```

### Transaction Flow

```
Handler
    ↓
Service
    ↓
TxManager.WithTx(ctx, fn)
    ↓
Repository operations (use tx)
    ↓
Error? → Rollback
Success? → Commit
```

### Rules

1. **Transaction boundary is Service layer**, not Handler or Repository
2. **NEVER** open transactions in handlers
3. **NEVER** call `Begin()`/`Commit()`/`Rollback()` manually
4. **Repository** should accept `*gorm.DB` parameter or have `SetTx()` method
5. **One transaction per business operation** — keep them short

## Soft Delete

GORM's built-in soft delete via `gorm.DeletedAt`:

```go
type BaseModel struct {
    ID        string         `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
```

## Timestamps

```go
// Auto-set by GORM
CreatedAt → set on first insert
UpdatedAt → set on every update
```

## Migration Order

```
Phase 1 (Core):
  1. users → roles → permissions → role_permissions → user_roles
  2. tenants → tenant_settings → cancel_requests → plans → subscriptions

Phase 2 (Business):
  3. audit_logs
  4. files
  5. announcements
  6. wiki_spaces → wiki_nodes → documents → document_revisions → wiki_space_members
```

## Query Builder Pattern

### Package: `pkg/pagination`

```go
// Pagination
req := pagination.NewPageRequest(page, pageSize)

// Sorting (with whitelist protection)
allowedSortFields := map[string]string{
    "name": "name",
    "created_at": "created_at",
    "updated_at": "updated_at",
}
query = pagination.ApplySort(query, req.SortBy, req.SortOrder, allowedSortFields)

// Keyword search
query = pagination.ApplyKeyword(query, req.Keyword, "name", "description")

// Pagination
query = pagination.ApplyPagination(query, req.Page, req.PageSize)

// Execute
var items []T
var total int64
query.Count(&total).Find(&items)
result := pagination.NewPageResult(items, total, req.Page, req.PageSize)
```

## Tenant-Scoped Queries

### Package: `internal/common/tenantctx`

```go
// All tenant-owned models MUST use FilterQuery
query := tenantctx.FilterQuery(ctx, db, "tenant_id")
// Automatically adds WHERE tenant_id = ? (or skips for master admin)
```

## Data Integrity

### Constraints

- Foreign keys enforced at DB level
- `tenant_id` indexed for isolation queries
- `deleted_at` indexed for soft delete queries
- `created_at` indexed for time-range queries

### Validation

- Server-side validation in handlers via `validator.ValidateJSON()`
- Business rule validation in service layer
- NEVER trust client-side validation alone

## Best Practices

1. **Always** use `idgen.New()` for new entity IDs
2. **Always** use transactions for multi-step operations
3. **Always** use `FilterQuery()` for tenant-owned queries
4. **NEVER** raw SQL string concatenation (use GORM's parameter binding)
5. **Index** frequently filtered columns (`tenant_id`, `deleted_at`, `created_at`)
6. **Use** `gorm.DeletedAt` for soft delete, not manual `deleted_at` field