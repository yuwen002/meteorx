# Permission / RBAC Architecture

## Current State (Baseline)

### Permission Model

```
Role → Permission (many-to-many)
User → Role (many-to-many)
Permission → {resource}:{action} (e.g., user:create, wiki:document:edit)
```

### Middleware Chain

```
1. AuthMiddleware     → Parse JWT, inject user info
2. AuditMiddleware     → Log all requests
3. PermissionMiddleware → Check RBAC permissions (when AutoRequirePermission is used)
```

### Permission Resolution

1. **Auto Permission Derivation** — HTTP method + path → permission code
   - `GET /users` → `user:list`
   - `POST /users` → `user:create`
   - `PUT /users/{id}` → `user:update`
   - `DELETE /users/{id}` → `user:delete`

2. **Explicit Permission** — Route-level override
   - `r.Use(middleware.RequirePermission(checker, "custom:permission"))`

3. **Superadmin Bypass** — `superadmin` role skips all permission checks

### Permission Checker

```go
type PermissionChecker interface {
    GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)
}
```

### Known Gaps

- [ ] Permission codes not centrally defined (scattered in handlers/routes)
- [ ] No cache for permission lookups (DB query per request)
- [ ] No explicit system permission vs resource permission distinction
- [ ] Wiki module has no permission middleware yet

## Target State (P0-2 Hardening)

```
Permission Codes (centralized constants)
    ↓
Route Registration (automatic derivation or explicit)
    ↓
Permission Cache (Redis, keyed by userID)
    ↓
Middleware (permission check with cache invalidation on role change)
```