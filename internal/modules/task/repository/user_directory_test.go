package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserDirectory_ResolveUserNames 验证批量解析：昵称优先、空昵称回退用户名，
// 且未命中的 ID 不出现在结果中。
func TestUserDirectory_ResolveUserNames(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	d := NewUserDirectory(gormDB)

	rows := sqlmock.NewRows([]string{"id", "nickname", "username"}).
		AddRow("u1", "张三", "zhangsan").
		AddRow("u2", "", "lisi")
	mock.ExpectQuery("SELECT .* FROM `users`").WillReturnRows(rows)

	names, err := d.ResolveUserNames(context.Background(), []string{"u1", "u2", "u3"})
	require.NoError(t, err)
	assert.Equal(t, "张三", names["u1"])   // 昵称优先
	assert.Equal(t, "lisi", names["u2"]) // 昵称空回退用户名
	assert.NotContains(t, names, "u3")   // 未命中
}

// TestUserDirectory_EmptyIDs 空 ID 列表应直接返回空映射且不触发查询。
func TestUserDirectory_EmptyIDs(t *testing.T) {
	gormDB, _ := newTaskTestDB(t)
	d := NewUserDirectory(gormDB)
	names, err := d.ResolveUserNames(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, names)
}

// TestUserDirectory_ListAssignable 验证存活成员列表：昵称优先回退用户名，且保留排序。
func TestUserDirectory_ListAssignable(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	d := NewUserDirectory(gormDB)

	rows := sqlmock.NewRows([]string{"id", "nickname", "username"}).
		AddRow("u1", "张三", "zhangsan").
		AddRow("u2", "", "lisi")
	mock.ExpectQuery("SELECT .* FROM `users` WHERE tenant_id").
		WithArgs("t1").WillReturnRows(rows)

	list, err := d.ListAssignable(context.Background(), "t1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, AssigneeBrief{ID: "u1", Name: "张三"}, list[0])
	assert.Equal(t, AssigneeBrief{ID: "u2", Name: "lisi"}, list[1])
}

// TestUserDirectory_ListAssignable_EmptyTenant 空租户不触发查询，直接返回空列表。
func TestUserDirectory_ListAssignable_EmptyTenant(t *testing.T) {
	gormDB, _ := newTaskTestDB(t)
	d := NewUserDirectory(gormDB)
	list, err := d.ListAssignable(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, list)
}
