// Package repository 实现协作即时消息（Chat）模块的 GORM 仓储层。
package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/modules/chat/model"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

// ---- 持久化对象 ----

// ConversationPO 会话持久化对象，映射 chat_conversations 表。
// Key 列名为 conv_key（key 是 MySQL 保留字），(tenant_id, conv_key) 唯一以支持私聊去重。
// Name 仅群聊使用（direct 会话留空）。
type ConversationPO struct {
	ID        string         `gorm:"primaryKey;size:26;comment:会话ID"`
	TenantID  string         `gorm:"size:26;not null;uniqueIndex:idx_chat_conv_key;comment:租户ID"`
	Type      string         `gorm:"size:20;not null;comment:会话类型(direct/group)"`
	Key       string         `gorm:"column:conv_key;size:80;not null;uniqueIndex:idx_chat_conv_key;comment:会话去重键"`
	Name      string         `gorm:"size:100;comment:群名称(仅 group)"`
	CreatedAt time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

// TableName 返回会话表名。
func (ConversationPO) TableName() string { return "chat_conversations" }

// ConversationMemberPO 会话成员持久化对象，映射 chat_conversation_members 表。
type ConversationMemberPO struct {
	ID             string     `gorm:"primaryKey;size:26;comment:成员记录ID"`
	ConversationID string     `gorm:"size:26;not null;uniqueIndex:idx_chat_member;index;comment:会话ID"`
	TenantID       string     `gorm:"size:26;not null;index;comment:租户ID"`
	UserID         string     `gorm:"column:user_id;size:26;not null;uniqueIndex:idx_chat_member;index;comment:成员用户ID"`
	Role           string     `gorm:"size:20;default:member;comment:成员角色(owner/member)"`
	LastReadAt     *time.Time `gorm:"comment:最后已读时间"`
	JoinedAt       time.Time  `gorm:"autoCreateTime;comment:加入时间"`
}

// TableName 返回会话成员表名。
func (ConversationMemberPO) TableName() string { return "chat_conversation_members" }

// ChatMessagePO 聊天消息持久化对象，映射 chat_messages 表。
type ChatMessagePO struct {
	ID             string     `gorm:"primaryKey;size:26;comment:消息ID"`
	ConversationID string     `gorm:"size:26;not null;index:idx_chat_msg_conv;comment:会话ID"`
	TenantID       string     `gorm:"size:26;not null;index;comment:租户ID"`
	SenderID       string     `gorm:"column:user_id;size:26;not null;comment:发送人ID"`
	Content        string     `gorm:"type:text;comment:消息正文（撤回后置空）"`
	Type           string     `gorm:"size:20;default:text;comment:消息类型"`
	Deleted        bool       `gorm:"default:false;comment:是否已撤回"`
	DeletedAt      *time.Time `gorm:"comment:撤回时间"`
	CreatedAt      time.Time  `gorm:"autoCreateTime;index:idx_chat_msg_conv;comment:发送时间"`
}

// TableName 返回消息表名。
func (ChatMessagePO) TableName() string { return "chat_messages" }

// ---- PO <-> 领域模型转换 ----

func (p ConversationPO) toDomain() *model.Conversation {
	return &model.Conversation{
		ID:        p.ID,
		TenantID:  p.TenantID,
		Type:      p.Type,
		Key:       p.Key,
		Name:      p.Name,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func (p ConversationMemberPO) toDomain() *model.ConversationMember {
	return &model.ConversationMember{
		ID:             p.ID,
		ConversationID: p.ConversationID,
		TenantID:       p.TenantID,
		UserID:         p.UserID,
		Role:           p.Role,
		LastReadAt:     p.LastReadAt,
		JoinedAt:       p.JoinedAt,
	}
}

func (p ChatMessagePO) toDomain() *model.ChatMessage {
	return &model.ChatMessage{
		ID:             p.ID,
		ConversationID: p.ConversationID,
		TenantID:       p.TenantID,
		SenderID:       p.SenderID,
		Content:        p.Content,
		Type:           p.Type,
		Deleted:        p.Deleted,
		DeletedAt:      p.DeletedAt,
		CreatedAt:      p.CreatedAt,
	}
}

// ---- 迁移 ----

// Models 返回本模块全部待迁移模型，作为 AutoMigrate 与脏列巡检的单一模型来源。
func Models() []interface{} {
	return []interface{}{&ConversationPO{}, &ConversationMemberPO{}, &ChatMessagePO{}}
}

// AutoMigrate 自动迁移聊天模块表结构。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}

// ---- 仓储实现 ----

// chatRepository ChatRepository 的 GORM 实现。
type chatRepository struct {
	db *gorm.DB
}

// NewChatRepository 创建聊天仓储实例。
func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

// EnsureDirectConversation 获取或创建 1:1 私聊会话，并保证双方成员存在（事务内幂等）。
func (r *chatRepository) EnsureDirectConversation(ctx context.Context, tenantID, userA, userB string) (*model.Conversation, error) {
	key := model.DirectKey(userA, userB)
	now := time.Now()

	var conv ConversationPO
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先按 (tenant_id, conv_key) 查既有会话；不存在则创建
		q := tx.Where("tenant_id = ? AND conv_key = ?", tenantID, key)
		if err := q.First(&conv).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			conv = ConversationPO{ID: idgen.New(), TenantID: tenantID, Type: model.ConversationTypeDirect, Key: key}
			if err := tx.Create(&conv).Error; err != nil {
				return err
			}
		}
		// 为两位参与者补建成员记录（已存在则跳过，保证幂等）
		for _, uid := range []string{userA, userB} {
			var cnt int64
			if err := tx.Model(&ConversationMemberPO{}).
				Where("conversation_id = ? AND user_id = ?", conv.ID, uid).
				Count(&cnt).Error; err != nil {
				return err
			}
			if cnt == 0 {
				m := ConversationMemberPO{ID: idgen.New(), ConversationID: conv.ID, TenantID: tenantID, UserID: uid, Role: model.MemberRoleMember, JoinedAt: now}
				if err := tx.Create(&m).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return conv.toDomain(), nil
}

// CreateGroupConversation 创建群聊会话（事务内写会话行与全部成员行）。
// 去重键取新生成 ULID，与 direct 的规范化 key 命名空间天然不冲突。
func (r *chatRepository) CreateGroupConversation(ctx context.Context, tenantID, creatorID, name string, memberIDs []string) (*model.Conversation, error) {
	convPO := ConversationPO{
		ID:       idgen.New(),
		TenantID: tenantID,
		Type:     model.ConversationTypeGroup,
		Key:      idgen.New(),
		Name:     name,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&convPO).Error; err != nil {
			return err
		}
		// 创建人为 owner，被邀请者为 member
		if err := tx.Create(&ConversationMemberPO{
			ID: idgen.New(), ConversationID: convPO.ID, TenantID: tenantID,
			UserID: creatorID, Role: model.MemberRoleOwner,
		}).Error; err != nil {
			return err
		}
		if len(memberIDs) > 0 {
			rows := make([]ConversationMemberPO, 0, len(memberIDs))
			for _, uid := range memberIDs {
				rows = append(rows, ConversationMemberPO{
					ID: idgen.New(), ConversationID: convPO.ID, TenantID: tenantID,
					UserID: uid, Role: model.MemberRoleMember,
				})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return convPO.toDomain(), nil
}

// UpdateConversationName 重命名会话。
func (r *chatRepository) UpdateConversationName(ctx context.Context, conversationID, name string) error {
	return r.db.WithContext(ctx).Model(&ConversationPO{}).
		Where("id = ?", conversationID).Update("name", name).Error
}

// DeleteConversation 软删除会话。
func (r *chatRepository) DeleteConversation(ctx context.Context, conversationID string) error {
	return r.db.WithContext(ctx).Where("id = ?", conversationID).Delete(&ConversationPO{}).Error
}

// GetMemberRole 返回用户在会话中的角色与非成员标记。
func (r *chatRepository) GetMemberRole(ctx context.Context, conversationID, userID string) (string, bool, error) {
	var p ConversationMemberPO
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return p.Role, true, nil
}

// CountMembers 统计会话成员数。
func (r *chatRepository) CountMembers(ctx context.Context, conversationID string) (int64, error) {
	var cnt int64
	err := r.db.WithContext(ctx).Model(&ConversationMemberPO{}).
		Where("conversation_id = ?", conversationID).Count(&cnt).Error
	return cnt, err
}

// AddMembers 向会话添加成员，已存在的成员静默跳过；返回实际新增人数。
func (r *chatRepository) AddMembers(ctx context.Context, conversationID, tenantID string, userIDs []string) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	var existing []string
	if err := r.db.WithContext(ctx).Model(&ConversationMemberPO{}).
		Where("conversation_id = ?", conversationID).Pluck("user_id", &existing).Error; err != nil {
		return 0, err
	}
	has := make(map[string]struct{}, len(existing))
	for _, uid := range existing {
		has[uid] = struct{}{}
	}
	added := int64(0)
	for _, uid := range userIDs {
		if _, ok := has[uid]; ok {
			continue
		}
		has[uid] = struct{}{} // 同一请求内重复 ID 只加一次
		m := ConversationMemberPO{ID: idgen.New(), ConversationID: conversationID, TenantID: tenantID, UserID: uid, Role: model.MemberRoleMember}
		if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// RemoveMember 从会话移除指定成员。
func (r *chatRepository) RemoveMember(ctx context.Context, conversationID, userID string) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&ConversationMemberPO{}).Error
}

// GetConversation 按 ID 查询未删除会话。
func (r *chatRepository) GetConversation(ctx context.Context, id string) (*model.Conversation, error) {
	var p ConversationPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return p.toDomain(), nil
}

// IsMember 判断用户是否为指定会话（同租户）的成员。
func (r *chatRepository) IsMember(ctx context.Context, conversationID, tenantID, userID string) (bool, error) {
	if conversationID == "" || userID == "" {
		return false, nil
	}
	var cnt int64
	err := r.db.WithContext(ctx).Model(&ConversationMemberPO{}).
		Where("conversation_id = ? AND tenant_id = ? AND user_id = ?", conversationID, tenantID, userID).
		Count(&cnt).Error
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// ListMembers 列出会话全部成员。
func (r *chatRepository) ListMembers(ctx context.Context, conversationID string) ([]*model.ConversationMember, error) {
	var pos []ConversationMemberPO
	if err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Find(&pos).Error; err != nil {
		return nil, err
	}
	out := make([]*model.ConversationMember, 0, len(pos))
	for _, p := range pos {
		out = append(out, p.toDomain())
	}
	return out, nil
}

// ListConversations 列出某用户全部会话，按最后消息时间倒序，并富化对方/摘要/未读数。
func (r *chatRepository) ListConversations(ctx context.Context, tenantID, userID string) ([]*model.Conversation, error) {
	var myMembers []ConversationMemberPO
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Find(&myMembers).Error; err != nil {
		return nil, err
	}
	if len(myMembers) == 0 {
		return []*model.Conversation{}, nil
	}
	convIDs := make([]string, 0, len(myMembers))
	readAt := make(map[string]*time.Time, len(myMembers))
	for _, m := range myMembers {
		convIDs = append(convIDs, m.ConversationID)
		readAt[m.ConversationID] = m.LastReadAt
	}

	// 会话基本信息
	var convPos []ConversationPO
	if err := r.db.WithContext(ctx).Where("id IN ?", convIDs).Find(&convPos).Error; err != nil {
		return nil, err
	}

	// 全部成员，用于确定 direct 会话的对方与群会话成员数
	var allMembers []ConversationMemberPO
	if err := r.db.WithContext(ctx).Where("conversation_id IN ?", convIDs).Find(&allMembers).Error; err != nil {
		return nil, err
	}
	// 会话类型映射：仅 direct 会话需要取对方，group 会话展示群名
	typeOf := make(map[string]string, len(convPos))
	for _, cp := range convPos {
		typeOf[cp.ID] = cp.Type
	}
	peerOf := make(map[string]string, len(convIDs))     // conversationID -> 对方 userID（仅 direct）
	memberCount := make(map[string]int64, len(convIDs)) // conversationID -> 成员数
	myRole := make(map[string]string, len(convIDs))     // conversationID -> 当前用户角色
	for _, m := range allMembers {
		memberCount[m.ConversationID]++
		if m.UserID == userID {
			myRole[m.ConversationID] = m.Role
			continue
		}
		if typeOf[m.ConversationID] == model.ConversationTypeDirect {
			peerOf[m.ConversationID] = m.UserID
		}
	}

	out := make([]*model.Conversation, 0, len(convPos))
	for _, cp := range convPos {
		conv := cp.toDomain()
		conv.PeerUserID = peerOf[conv.ID]
		conv.MemberCount = memberCount[conv.ID]
		conv.MyRole = myRole[conv.ID]
		// 最后一条消息
		var last ChatMessagePO
		if err := r.db.WithContext(ctx).
			Where("conversation_id = ?", conv.ID).
			Order("created_at DESC").Limit(1).First(&last).Error; err == nil {
			conv.LastMessage = last.Content
			t := last.CreatedAt
			conv.LastAt = &t
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		// 未读数：非本人发送且晚于已读水位的消息数
		uq := r.db.WithContext(ctx).Model(&ChatMessagePO{}).
			Where("conversation_id = ? AND user_id <> ?", conv.ID, userID)
		if ra := readAt[conv.ID]; ra != nil {
			uq = uq.Where("created_at > ?", *ra)
		}
		var unread int64
		if err := uq.Count(&unread).Error; err != nil {
			return nil, err
		}
		conv.UnreadCount = unread
		out = append(out, conv)
	}

	// 按最后消息时间倒序（无消息的排在后面）
	sortConversationsByLastAt(out)
	return out, nil
}

// sortConversationsByLastAt 依据 LastAt 倒序稳定排序会话，nil 视为最旧。
func sortConversationsByLastAt(cs []*model.Conversation) {
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0; j-- {
			a, b := cs[j-1].LastAt, cs[j].LastAt
			// 期望降序：前一个更早则与后一个交换
			if (a == nil && b != nil) || (a != nil && b != nil && a.Before(*b)) {
				cs[j-1], cs[j] = cs[j], cs[j-1]
			} else {
				break
			}
		}
	}
}

// SendMessage 落库一条消息。
func (r *chatRepository) SendMessage(ctx context.Context, msg *model.ChatMessage) error {
	p := ChatMessagePO{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		TenantID:       msg.TenantID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		Type:           msg.Type,
		CreatedAt:      msg.CreatedAt,
	}
	return r.db.WithContext(ctx).Create(&p).Error
}

// GetMessage 按会话与消息 ID 查询一条消息（限未软删除行）。
func (r *chatRepository) GetMessage(ctx context.Context, conversationID, messageID string) (*model.ChatMessage, error) {
	var p ChatMessagePO
	if err := r.db.WithContext(ctx).
		Where("id = ? AND conversation_id = ?", messageID, conversationID).
		First(&p).Error; err != nil {
		return nil, err
	}
	return p.toDomain(), nil
}

// RecallMessage 软撤回消息：仅对 deleted=false 的行生效，重复撤回不命中（RowsAffected=0）。
func (r *chatRepository) RecallMessage(ctx context.Context, conversationID, messageID string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&ChatMessagePO{}).
		Where("id = ? AND conversation_id = ? AND deleted = ?", messageID, conversationID, false).
		Updates(map[string]interface{}{"deleted": true, "deleted_at": at, "content": ""}).Error
}

// ListMessages 分页查询会话历史消息：按时间倒序取分页，再将页内反转为升序返回。
func (r *chatRepository) ListMessages(ctx context.Context, conversationID string, page, pageSize int) ([]*model.ChatMessage, int64, error) {
	q := r.db.WithContext(ctx).Model(&ChatMessagePO{}).Where("conversation_id = ?", conversationID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var pos []ChatMessagePO
	if err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*model.ChatMessage, 0, len(pos))
	// 页内反转为升序，前端自上而下按时间顺序渲染
	for i := len(pos) - 1; i >= 0; i-- {
		out = append(out, pos[i].toDomain())
	}
	return out, total, nil
}

// MarkRead 更新成员已读水位。
func (r *chatRepository) MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&ConversationMemberPO{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_at", at).Error
}
