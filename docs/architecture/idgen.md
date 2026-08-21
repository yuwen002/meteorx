# ID 生成架构

## 概述

使用 ULID（通用唯一词典有序标识符）为所有业务实体提供统一的 ID 生成。`pkg/idgen` 包提供单一入口点，替代分散的 `ulid` 和 `uuid` 调用。

## 包路径：`pkg/idgen`

### API

```go
// 生成新的 ULID（26 个字符，按时间有序）
id := idgen.New()
// 示例：01H7K3N5P8R2S4T6V8W0X2Y4Z6

// 生成 UUID（与 New() 相同，为语义清晰的别名）
id := idgen.NewUUID()

// 解析已有的 ULID 字符串
ulid, err := idgen.Parse("01H7K3N5P8R2S4T6V8W0X2Y4Z6")

// MustParse（对无效输入 panic，用于已知有效的字符串）
ulid := idgen.MustParse("01H7K3N5P8R2S4T6V8W0X2Y4Z6")
```

### ULID 格式

```
26 个字符，Crockford Base32 编码：

 01H7K3N5P8R2S4T6V8W0X2Y4Z6
 |--------|------------------|
  时间戳         随机数
  (10 字符)     (16 字符)
  (48 bit)      (80 bit)

按时间有序：较早时间戳的 ID 排在较晚的之前
```

### 为什么选择 ULID 而不是 UUID？

| 特性 | ULID | UUID v4 | UUID v7 |
|------|------|---------|---------|
| 长度 | 26 字符 | 36 字符（含连字符） | 36 字符 |
| 按时间有序 | 是（前缀） | 否 | 是（前缀） |
| 数据库索引效率 | 高（B+Tree 局部性） | 低 | 中 |
| 编码方式 | Crockford Base32 | 十六进制 + 连字符 | 十六进制 + 连字符 |
| 碰撞概率 | 极低 | 极低 | 极低 |
| 可读性 | 较好（无连字符） | 较差 | 较差 |

### 核心规则

```
1. 所有业务实体必须使用 idgen.New() 生成 ID
2. 禁止直接使用 ulid.Generate()、ulid.Make()、uuid.New()
3. pkg/ulid 和 pkg/uuid 包已废弃（仅向后兼容）
4. 数据库主键使用 VARCHAR(26) 存储 ULID
5. API DTO 使用 string 类型传递 ID（不是 ULID 类型）
```

### 迁移指南

#### 旧代码（已废弃）

```go
// 分散的导入
import "meteorx/pkg/ulid"
id := ulid.Generate()

// 或者
import "meteorx/pkg/uuid"
id := uuid.New()
```

#### 新代码（必须使用）

```go
import "meteorx/pkg/idgen"
id := idgen.New()
```

### 使用 ULID 的实体

| 模块 | 实体 | ID 字段 | 备注 |
|------|------|----------|------|
| auth | User | id | 主键 |
| auth | RefreshToken | id | 主键 |
| rbac | Role | id | 主键 |
| rbac | Permission | id | 主键 |
| rbac | RolePermission | id | 主键 |
| rbac | UserRole | id | 主键 |
| tenant | Tenant | id | 主键 |
| tenant | TenantSettings | id | 主键 |
| tenant | CancelRequest | id | 主键 |
| plan | Plan | id | 主键 |
| plan | Subscription | id | 主键 |
| audit | AuditLog | id | 主键 |
| file | File | id | 主键 |
| wiki | WikiSpace | id | 主键 |
| wiki | WikiNode | id | 主键 |
| wiki | Document | id | 主键 |
| wiki | DocumentRevision | id | 主键 |
| wiki | WikiSpaceMember | id | 主键 |
| notification | Announcement | id | 主键 |

### GORM 模型集成

```go
// 带 ULID 主键的基础模型
type BaseModel struct {
    ID        string         `gorm:"primaryKey;type:varchar(26)" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 通过 BeforeCreate 钩子在创建时自动生成 ID
func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
    if base.ID == "" {
        base.ID = idgen.New()
    }
    return nil
}
```

### 查询安全

```go
// 在数据库查找前始终校验 ID 格式
func (r *repository) GetByID(ctx context.Context, id string) (*model.Entity, error) {
    // 校验 ULID 格式
    if _, err := idgen.Parse(id); err != nil {
        return nil, apperrors.ErrBadRequest("invalid id format")
    }
    return r.getByID(ctx, id)
}
```

### 测试

```go
func TestIDGeneration(t *testing.T) {
    id1 := idgen.New()
    id2 := idgen.New()

    // 两者都是有效的 ULID
    _, err := idgen.Parse(id1)
    assert.NoError(t, err)

    // 唯一性
    assert.NotEqual(t, id1, id2)

    // 可排序性（按顺序生成时 id1 应 <= id2）
    assert.LessOrEqual(t, id1, id2)

    // 长度
    assert.Equal(t, 26, len(id1))
}
```

### 最佳实践

1. **始终**使用 `idgen.New()` — 绝不要分散地生成 ID
2. **使用** `idgen.Parse()` 在查询数据库之前校验 ID 格式
3. **仅** 对可信的、预校验的 ID 使用 `idgen.MustParse()`
4. **在** MySQL 中将 ID 存储为 `VARCHAR(26)` 以优化索引
5. **在**所有新实体文件中引入 `idgen` 包
6. **在**重构时替换旧的 `ulid.` 和 `uuid.` 调用