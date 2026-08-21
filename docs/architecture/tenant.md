# Tenant Isolation Architecture

## Current State (Baseline)

### Tenant ID Flow

```
HTTP Request
    ↓
Auth Middleware (JWT Parse)
    ↓
contextx.SetVars(ctx, tenantID, userID, roles)
    ↓
Handler gets tenantID via contextx.GetTenantID(r.Context())
    ↓
Service passes tenantID to Repository
    ↓
Repository uses tenantID in WHERE clause
```

### Tenant Context Source

1. **JWT Token Claims** — Primary source for authenticated requests
   - `tenant_id` claim in JWT payload
2. **Domain-based Tenant Resolution** — For public routes
   - `tenant_id` resolved from request domain via middleware
3. **System Tenant** — For super admin operations
   - Special tenant ID constant (`system` or `master`)

### Models with tenant_id

| Model | Table | tenant_id field | Notes |
|-------|-------|-----------------|-------|
| User | users | Yes | Partitioned by tenant |
| Role | roles | Yes | Per-tenant roles |
| Permission | permissions | Yes | May be shared |
| AuditLog | audit_logs | Yes | All operations |
| Plan | plans | Yes | Mostly system-level |
| Subscription | subscriptions | Yes | Per-tenant subscription |
| TenantSettings | tenant_settings | Yes | Config per tenant |
| WikiSpace | wiki_spaces | Yes | New Wiki module |

### Known Gaps

- [ ] Some queries may not filter by tenant_id in Repository layer
- [ ] No cross-tenant access test exists
- [ ] No centralized tenant context validation
- [ ] System admin vs tenant admin not consistently distinguished

## Target State (P0-1 Hardening)

```
HTTP Request
    ↓
JWT / Domain Resolution
    ↓
TenantContext Middleware (validates, injects)
    ↓
tenantctx.From(ctx) — single access point
    ↓
Service layer — no raw tenant_id handling
    ↓
Repository — automatic tenant scoping
    ↓
DB Query — tenant_id always included
```

### Key Rules

1. **All tenant-owned models MUST have tenant_id in queries**
2. **Super admin (master) can bypass tenant isolation**
3. **Cross-tenant access is forbidden by default**
4. **Repository layer enforces tenant scoping, not just Service layer**