# Error / Response Architecture

## Current State (Baseline)

### Response Format

**Success:**
```json
{
  "code": 0,
  "data": {},
  "message": "success"
}
```

**Error:**
```json
{
  "code": 40001,
  "data": null,
  "message": "错误描述"
}
```

### Error Handling

- Handler-level error responses (`response.Fail()`, `response.BadRequest()`)
- No centralized error codes
- No request ID tracking
- Inconsistent HTTP status mapping

### Known Gaps

- [ ] No unified error code registry
- [ ] No request_id for tracing
- [ ] Pagination response format inconsistent across modules
- [ ] No global error handler

## Target State (P0-3 Hardening)

```json
{
  "data": {},
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100
  },
  "request_id": "req_xxx"
}
```

**Error:**
```json
{
  "code": "WIKI_DOCUMENT_NOT_FOUND",
  "message": "Document not found",
  "request_id": "req_xxx"
}
```

### Key Principles

1. All errors use machine-readable codes
2. HTTP status maps to error categories
3. Every response includes request_id
4. Pagination is consistent across all list endpoints