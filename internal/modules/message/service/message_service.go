// Package service 实现站内消息的读写编排与通知落库。
package service

import (
	"context"
	"errors"
	"meteorx/internal/modules/message/dto"
	"meteorx/internal/modules/message/model"
	messagerepo "meteorx/internal/modules/message/repository"
	"meteorx/internal/notify"
	"meteorx/internal/ws"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"
	"strings"
	"time"
)

// ErrNotFound 消息不存在或不属于当前收件人。
var ErrNotFound = messagerepo.ErrMessageNotFound

// 偏好写入的拒绝原因。用哨兵错误区分「入参不合法」与「存储失败」，
// 供 handler 映射成 400 而非 500。
var (
	// ErrInvalidCategory 偏好里出现了未知分类
	ErrInvalidCategory = errors.New("invalid message category")
	// ErrMandatoryCategory 试图关闭强制接收的分类（安全告警）
	ErrMandatoryCategory = errors.New("mandatory message category")
)

// maxTenantFanout 单条租户定向消息最多展开的收件人数。
// 超过上限直接截断并告警：站内信落库是按人写行，无上限会让一条公告的写放大失控。
const maxTenantFanout = 500

// maxBatchIDs 单次批量已读 / 删除允许的消息数上限（handler 的校验标签使用同一数值）。
// 批量化的是「多条独立行」，一条 SQL 即可完成，上限只用于挡住超长 IN 列表。
const maxBatchIDs = 100

// unreadPusher 未读角标推送的最小依赖面（*ws.Hub 天然满足）。
type unreadPusher interface {
	SendToUser(userID string, msg *ws.Message)
}

// MessageService 站内消息服务。
type MessageService struct {
	repo  messagerepo.MessageRepository
	users messagerepo.UserDirectory
	prefs messagerepo.PreferenceRepository
	hub   unreadPusher
}

// NewMessageService 创建站内消息服务。
// prefs 决定「谁不收哪类消息」，为 nil 时按全部接收处理（测试桩可省略）；
// hub 可为 nil（WS 关闭时不推送角标，收件箱仍可正常读写）。
func NewMessageService(repo messagerepo.MessageRepository, users messagerepo.UserDirectory,
	prefs messagerepo.PreferenceRepository, hub unreadPusher) *MessageService {
	svc := &MessageService{repo: repo, users: users, prefs: prefs}
	if hub == nil {
		return svc
	}
	// 预防 typed nil：调用方传入 nil *ws.Hub 时，装箱后的接口不为 nil，
	// 直接赋值会让后续 s.hub == nil 检查失效并在 SendToUser 上崩溃
	if p, ok := hub.(*ws.Hub); ok && p == nil {
		return svc
	}
	svc.hub = hub
	return svc
}

// List 分页查询当前用户的收件箱。
func (s *MessageService) List(ctx context.Context, f messagerepo.ListFilter) ([]*dto.MessageResp, int64, error) {
	msgs, total, err := s.repo.ListByUser(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.MessageResp, len(msgs))
	for i, m := range msgs {
		items[i] = dto.ToResp(m)
	}
	return items, total, nil
}

// UnreadCount 返回当前用户的未读总数与分类明细。
func (s *MessageService) UnreadCount(ctx context.Context, userID string) (*dto.UnreadCountResp, error) {
	byCategory, err := s.repo.CountUnreadByCategory(ctx, userID)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, n := range byCategory {
		total += n
	}
	return &dto.UnreadCountResp{Total: total, ByCategory: byCategory}, nil
}

// Preferences 返回全部分类的接收状态。
// 未设置过的分类按「接收」返回，前端无需区分「未设置」与「已开启」。
func (s *MessageService) Preferences(ctx context.Context, userID string) (*dto.PreferencesResp, error) {
	muted := map[string]bool{}
	if s.prefs != nil {
		m, err := s.prefs.MutedCategories(ctx, userID)
		if err != nil {
			return nil, err
		}
		muted = m
	}
	categories := model.AllCategories()
	items := make([]dto.PreferenceItem, len(categories))
	for i, c := range categories {
		items[i] = dto.PreferenceItem{
			Category:  c,
			Muted:     muted[c],
			Mandatory: model.IsMandatoryCategory(c),
		}
	}
	return &dto.PreferencesResp{Items: items}, nil
}

// UpdatePreference 覆盖写入单条分类偏好，返回写入后的完整视图。
// 强制接收的分类拒绝关闭而不是静默忽略：静默会让前端以为已经生效，
// 用户以为关掉了告警，实际仍会收到。
func (s *MessageService) UpdatePreference(ctx context.Context, userID string, req dto.UpdatePreferenceReq) (*dto.PreferencesResp, error) {
	if !model.IsValidCategory(req.Category) {
		return nil, ErrInvalidCategory
	}
	if req.Muted && model.IsMandatoryCategory(req.Category) {
		return nil, ErrMandatoryCategory
	}
	if s.prefs == nil {
		return nil, errors.New("message preference repository not configured")
	}
	if err := s.prefs.Upsert(ctx, userID, []model.CategoryPreference{{
		Category: req.Category,
		Muted:    req.Muted,
	}}); err != nil {
		return nil, err
	}
	return s.Preferences(ctx, userID)
}

// MarkRead 标记单条消息已读（幂等：重复标记不报错）。
func (s *MessageService) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

// MarkAllRead 标记全部已读（可按分类）。
func (s *MessageService) MarkAllRead(ctx context.Context, userID, category string) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID, category)
}

// MarkReadBatch 批量标记已读，返回实际更新条数。
// 已经读过的、已被别处删掉的、不属于本人的 ID 都静默跳过：
// 用户选中的是一批可见消息，其中一条失效不应让整批回滚。
func (s *MessageService) MarkReadBatch(ctx context.Context, userID string, ids []string) (int64, error) {
	return s.repo.MarkReadBatch(ctx, userID, normalizeIDs(ids))
}

// Delete 删除单条消息。
func (s *MessageService) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, userID, id)
}

// DeleteBatch 批量删除，返回实际删除条数（语义同 MarkReadBatch）。
func (s *MessageService) DeleteBatch(ctx context.Context, userID string, ids []string) (int64, error) {
	return s.repo.DeleteBatch(ctx, userID, normalizeIDs(ids))
}

// normalizeIDs 去空白、去重，并按 maxBatchIDs 截断。
// 去重复用收件人展开的 dedupeIDs（同一 ID 在 IN 列表里重复既多占位，也让「实际更新条数」难以对账）；
// 截断是兜底：handler 已经按同一上限校验，但服务层不能依赖调用方守规矩，
// 否则内部调用仍可能拼出超长 SQL。
func normalizeIDs(ids []string) []string {
	out := dedupeIDs(ids)
	if len(out) > maxBatchIDs {
		out = out[:maxBatchIDs]
	}
	return out
}

// Deliver 实现 notify.InboxWriter：把一条业务通知落成收件箱消息。
// 收件人为空且指定了租户时按租户成员展开（公告场景）；
// 两者都为空则视为「无可投递对象」直接返回，避免产生无人可见的行。
func (s *MessageService) Deliver(ctx context.Context, in *notify.InboxMessage) error {
	if in == nil {
		return nil
	}

	userIDs := dedupeIDs(in.UserIDs)
	if len(userIDs) == 0 && in.TenantID != "" {
		ids, err := s.users.ListActiveUserIDs(ctx, in.TenantID, maxTenantFanout)
		if err != nil {
			return err
		}
		userIDs = dedupeIDs(ids)
	}
	// 再截断一次：UserDirectory 是接口，实现方未必遵守传入上限；
	// 写放大必须在本方法内封死，不能依赖下游
	if len(userIDs) > maxTenantFanout {
		logger.Warnf("[Message] 收件人 %d 超过上限 %d，已截断 tenant=%s type=%s",
			len(userIDs), maxTenantFanout, in.TenantID, in.Type)
		userIDs = userIDs[:maxTenantFanout]
	}
	if len(userIDs) == 0 {
		return nil
	}

	category := in.Category
	if category == "" {
		category = CategoryForType(in.Type)
	}

	// 偏好过滤：关闭了该分类的收件人既不落库也不推角标。
	// 强制接收的分类（安全告警）无视存量偏好：即使库里被人为写入了 muted 行，
	// 告警也不能因此被吞掉。
	if s.prefs != nil && !model.IsMandatoryCategory(category) {
		muted, err := s.prefs.MutedUserIDs(ctx, category, userIDs)
		if err != nil {
			// 查询失败按默认接收投递：宁可多打扰，不可漏消息
			logger.Warnf("[Message] 消息偏好查询失败，按默认接收投递 type=%s category=%s: %v", in.Type, category, err)
		} else if len(muted) > 0 {
			kept := make([]string, 0, len(userIDs)-len(muted))
			for _, uid := range userIDs {
				if _, ok := muted[uid]; ok {
					continue
				}
				kept = append(kept, uid)
			}
			userIDs = kept
		}
	}
	if len(userIDs) == 0 {
		return nil
	}

	// 去重键：优先用调用方按「业务事件 + 时间粒度」构造的键；
	// 未提供时回退为「来源类型:来源ID」，宁可保守去重也不重复骚扰；
	// 两者都定位不到事件（如平台公告）则不去重，每次投递都是新行。
	dedupeKey := strings.TrimSpace(in.DedupeKey)
	if dedupeKey == "" && in.SourceType != "" && in.SourceID != "" {
		dedupeKey = in.SourceType + ":" + in.SourceID
	}

	now := time.Now()
	msgs := make([]*model.Message, 0, len(userIDs))
	for _, uid := range userIDs {
		msgs = append(msgs, &model.Message{
			ID:         idgen.New(),
			TenantID:   in.TenantID,
			UserID:     uid,
			Category:   category,
			Title:      in.Title,
			Content:    in.Content,
			Link:       in.Link,
			Priority:   int(in.Priority),
			SourceType: in.SourceType,
			SourceID:   in.SourceID,
			DedupeKey:  dedupeKey,
			CreatedAt:  now,
		})
	}
	if err := s.repo.BatchCreate(ctx, msgs); err != nil {
		return err
	}

	s.pushUnreadCounts(ctx, userIDs)
	return nil
}

// pushUnreadCounts 落库成功后推送未读数角标。
// 推送失败不影响落库结果：离线用户下次打开页面会主动拉一次未读数。
func (s *MessageService) pushUnreadCounts(ctx context.Context, userIDs []string) {
	if s.hub == nil {
		return
	}
	counts, err := s.repo.CountUnreadByUsers(ctx, userIDs)
	if err != nil {
		logger.Warnf("[Message] 未读数统计失败，跳过角标推送: %v", err)
		return
	}
	now := time.Now().UnixMilli()
	for _, uid := range userIDs {
		s.hub.SendToUser(uid, &ws.Message{
			Type:    ws.MsgTypeUnreadCount,
			Payload: map[string]interface{}{"count": counts[uid], "total": counts[uid]},
			Time:    now,
		})
	}
}

// CategoryForType 把通知业务类型映射为收件箱分类。
// 未知类型归入系统消息：宁可分类不准，也不要让消息丢失。
func CategoryForType(t string) string {
	switch {
	case t == "announcement":
		return model.CategoryAnnouncement
	case t == "subscription_expiry":
		return model.CategoryPlan
	case strings.HasPrefix(t, "task"):
		return model.CategoryTask
	case t == "alert":
		return model.CategoryAudit
	default:
		return model.CategorySystem
	}
}

// dedupeIDs 去重并剔除空串，保持入参顺序。
func dedupeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
