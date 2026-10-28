package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/chat/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newChatTestDB 基于 go-sqlmock 构造 MySQL 方言的测试连接。
func newChatTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectationsWereMet()
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)
	return gormDB, mock
}

// TestSortConversationsByLastAt 会话应按最后消息时间倒序排列，无消息（nil）排在最后。
func TestSortConversationsByLastAt(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)
	t2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.Local)
	t3 := time.Date(2026, 1, 3, 10, 0, 0, 0, time.Local)

	cs := []*model.Conversation{
		{ID: "a", LastAt: &t1},
		{ID: "empty"},
		{ID: "c", LastAt: &t3},
		{ID: "b", LastAt: &t2},
	}
	sortConversationsByLastAt(cs)

	got := make([]string, 0, len(cs))
	for _, c := range cs {
		got = append(got, c.ID)
	}
	want := []string{"c", "b", "a", "empty"}
	if len(got) != len(want) {
		t.Fatalf("排序结果长度不符: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("排序结果错误: got %v want %v", got, want)
		}
	}
}

// TestChatRepo_SendMessage 发送消息应落库到 chat_messages。
func TestChatRepo_SendMessage(t *testing.T) {
	gormDB, mock := newChatTestDB(t)
	repo := NewChatRepository(gormDB)

	mock.ExpectExec("INSERT INTO `chat_messages`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.SendMessage(context.Background(), &model.ChatMessage{
		ID:             "m1",
		ConversationID: "c1",
		TenantID:       "t1",
		SenderID:       "u1",
		Content:        "你好",
		Type:           model.MessageTypeText,
		CreatedAt:      time.Now(),
	})
	require.NoError(t, err)
}
