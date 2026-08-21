# Audit Architecture

## Current State (Baseline)

### Audit Log Fields

| Field | Description |
|-------|-------------|
| id | Log ID (ULID) |
| user_id | Operator user ID |
| username | Operator username |
| tenant_id | Tenant ID |
| module | Module name (user, tenant, auth, wiki...) |
| action | Action type (create, update, delete, login...) |
| resource | Resource path |
| resource_id | Resource ID |
| method | HTTP method |
| path | Request path |
| request_body | Request payload |
| response_body | Response data |
| status_code | HTTP status code |
| result | Operation result (success/failure) |
| error_message | Error details if failed |
| client_ip | Client IP |
| user_agent | Browser UA |
| duration | Request duration (ms) |
| created_at | Timestamp |

### Audit Flow

```
Request → AuditMiddleware → CreateLog (batch) → DB
```

- Batch processing via `AuditBatchProcessor` (configured interval)
- Async: writes are buffered and flushed periodically

### Known Gaps

- [ ] No before/after value tracking for mutations
- [ ] No context-aware audit (auto vs manual)
- [ ] No unified action naming convention
- [ ] Some manual audit calls may bypass middleware

## Target State (P0-6)

```
Context-aware audit:
  - Middleware auto-captures standard CRUD
  - Service layer can enrich with before/after
  - Action naming follows {MODULE}_{ACTION} convention
  - All audit entries include tenant_id for isolation
```