# ID Generation Architecture

## Overview

Unified ID generation using ULID (Universally Unique Lexicographically Sortable Identifier) for all business entities. The `pkg/idgen` package provides a single entry point for ID generation, replacing scattered `ulid` and `uuid` usage.

## Package: `pkg/idgen`

### API

```go
// Generate new ULID (26 characters, time-ordered)
id := idgen.New()
// Example: 01H7K3N5P8R2S4T6V8W0X2Y4Z6

// Generate UUID (same as New(), alias for semantic clarity)
id := idgen.NewUUID()

// Parse existing ULID string
ulid, err := idgen.Parse("01H7K3N5P8R2S4T6V8W0X2Y4Z6")

// MustParse (panics on invalid input, for known-good strings)
ulid := idgen.MustParse("01H7K3N5P8R2S4T6V8W0X2Y4Z6")
```

### ULID Format

```
26 characters, Crockford Base32 encoding:

 01H7K3N5P8R2S4T6V8W0X2Y4Z6
 |--------|------------------|
  Timestamp      Randomness
  (10 chars)     (16 chars)
  (48 bits)      (80 bits)

Time-ordered: IDs from earlier timestamps sort before later ones
```

### Why ULID Over UUID?

| Feature | ULID | UUID v4 | UUID v7 |
|---------|------|---------|---------|
| Length | 26 chars | 36 chars (with hyphens) | 36 chars |
| Time-ordered | Yes (prefix) | No | Yes (prefix) |
| DB Index Efficiency | High (B+Tree locality) | Low | Medium |
| Encoding | Crockford Base32 | Hex + hyphens | Hex + hyphens |
| Collision Probability | Extremely low | Extremely low | Extremely low |
| Readability | Better (no hyphens) | Poor | Poor |

### Core Rules

```
1. ALL business entities MUST use idgen.New() for ID generation
2. NEVER use ulid.Generate(), ulid.Make(), uuid.New() directly
3. The pkg/ulid and pkg/uuid packages are DEPRECATED (backward-compat only)
4. Database primary keys use VARCHAR(26) for ULID storage
5. API DTOs use string type for IDs (not ULID type)
```

### Migration Guide

#### Old Code (Deprecated)

```go
// scattered imports
import "meteorx/pkg/ulid"
id := ulid.Generate()

// or
import "meteorx/pkg/uuid"
id := uuid.New()
```

#### New Code (Required)

```go
import "meteorx/pkg/idgen"
id := idgen.New()
```

### Entities Using ULID

| Module | Entity | ID Field | Notes |
|--------|--------|----------|-------|
| auth | User | id | Primary key |
| auth | RefreshToken | id | Primary key |
| rbac | Role | id | Primary key |
| rbac | Permission | id | Primary key |
| rbac | RolePermission | id | Primary key |
| rbac | UserRole | id | Primary key |
| tenant | Tenant | id | Primary key |
| tenant | TenantSettings | id | Primary key |
| tenant | CancelRequest | id | Primary key |
| plan | Plan | id | Primary key |
| plan | Subscription | id | Primary key |
| audit | AuditLog | id | Primary key |
| file | File | id | Primary key |
| wiki | WikiSpace | id | Primary key |
| wiki | WikiNode | id | Primary key |
| wiki | Document | id | Primary key |
| wiki | DocumentRevision | id | Primary key |
| wiki | WikiSpaceMember | id | Primary key |
| notification | Announcement | id | Primary key |

### GORM Model Integration

```go
// Base model with ULID primary key
type BaseModel struct {
    ID        string         `gorm:"primaryKey;type:varchar(26)" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Auto-generate ID on create via BeforeCreate hook
func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
    if base.ID == "" {
        base.ID = idgen.New()
    }
    return nil
}
```

### Query Safety

```go
// Always validate ID format before database lookup
func (r *repository) GetByID(ctx context.Context, id string) (*model.Entity, error) {
    // Validate ULID format
    if _, err := idgen.Parse(id); err != nil {
        return nil, apperrors.ErrBadRequest("invalid id format")
    }
    return r.getByID(ctx, id)
}
```

### Testing

```go
func TestIDGeneration(t *testing.T) {
    id1 := idgen.New()
    id2 := idgen.New()

    // Both are valid ULIDs
    _, err := idgen.Parse(id1)
    assert.NoError(t, err)

    // Unique
    assert.NotEqual(t, id1, id2)

    // Sortable (id1 should be <= id2 when generated in sequence)
    assert.LessOrEqual(t, id1, id2)

    // Length
    assert.Equal(t, 26, len(id1))
}
```

### Best Practices

1. **Always** use `idgen.New()` — never scattered ID generation
2. **Validate** ID format with `idgen.Parse()` before DB queries
3. **Use** `idgen.MustParse()` only for trusted, pre-validated IDs
4. **Store** IDs as `VARCHAR(26)` in MySQL for optimal indexing
5. **Include** `idgen` import in all new entity files
6. **Replace** old `ulid.` and `uuid.` calls during refactoring