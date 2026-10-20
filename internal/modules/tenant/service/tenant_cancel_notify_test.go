package service

import (
	"context"
	"errors"
	"testing"
	"time"

	tenantDto "meteorx/internal/modules/tenant/dto"
	tenantModel "meteorx/internal/modules/tenant/model"
	userModel "meteorx/internal/modules/user/model"
	"meteorx/internal/notify"

	"github.com/stretchr/testify/require"
)

// recNotifier 记录投递到某渠道的消息，用于断言收件人语义。
type recNotifier struct {
	typ   notify.ChannelType
	err   error
	msgs  []*notify.Message
	calls int
}

func (r *recNotifier) Type() notify.ChannelType { return r.typ }
func (r *recNotifier) Name() string             { return "record-" + string(r.typ) }
func (r *recNotifier) Send(_ context.Context, msg *notify.Message) error {
	r.calls++
	r.msgs = append(r.msgs, msg)
	return r.err
}

// useNotifyRecorders 把指定渠道注册为全局通知管理器，测试结束后恢复原值
func useNotifyRecorders(t *testing.T, channels ...*recNotifier) {
	t.Helper()
	prev := notify.GetGlobalManager()
	m := notify.NewManager(nil, "", "", nil)
	for _, c := range channels {
		m.AddChannel(c)
	}
	notify.SetGlobalManager(m)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
}

// seedCancelFixture 装配「一个待审批注销申请 + 含联系邮箱的租户 + 两名启用成员」
func seedCancelFixture(t *testing.T) (*TenantService, *mockTenantRepo, *mockUserRepo) {
	t.Helper()
	repo := newMockTenantRepo()
	userRepoMock := newMockUserRepo()
	svc := NewTenantService(repo, userRepoMock, newMockRoleRepo())

	email := "owner@acme.com"
	repo.seed(&tenantModel.Tenant{
		ID: "t-001", Name: "Acme", Domain: "acme",
		Status: tenantModel.StatusEnabled, ContactEmail: email,
	})
	repo.seedCancelRequest(&tenantModel.CancelRequest{
		ID: "c-001", TenantID: "t-001", TenantName: "Acme",
		Status: tenantModel.CancelRequestStatusPending, AppliedAt: time.Now(),
	})
	userRepoMock.users["t-001"] = []*userModel.User{
		{ID: "u-1", TenantID: "t-001", Status: 1},
		{ID: "u-2", TenantID: "t-001", Status: 1},
		{ID: "u-3", TenantID: "t-001", Status: 0}, // 已禁用成员不该收到续费/注销推送
	}
	return svc, repo, userRepoMock
}

// TestNotifyCancelRequest_WSTargetsMemberIDs 注销审批结果的 WS 推送收件人必须是租户
// 启用成员的用户 ID。旧实现把租户联系人邮箱当 userID 传入，而 hub.clients 以
// claims.UserID 为键，邮箱永远 miss 且不报错——审批结果通知实际一条也没送到，
// 与上一轮修掉的到期提醒是同一处病灶。
func TestNotifyCancelRequest_WSTargetsMemberIDs(t *testing.T) {
	svc, _, _ := seedCancelFixture(t)

	email := &recNotifier{typ: notify.ChannelEmail}
	wsC := &recNotifier{typ: notify.ChannelWebSocket}
	useNotifyRecorders(t, email, wsC)

	ctx := context.Background()
	// EffectiveDays>0 避免顺带执行注销（软删租户），聚焦通知本身
	_, err := svc.ApproveCancellation(ctx, "c-001", "approver-1", tenantDto.AdminApproveCancelReq{
		AdminReviewCancelReq: tenantDto.AdminReviewCancelReq{ReviewRemark: "已确认"},
		EffectiveDays:        7,
	})
	require.NoError(t, err)

	require.Equal(t, 1, wsC.calls, "有启用成员时应走 WS 推送")
	require.Equal(t, []string{"u-1", "u-2"}, wsC.msgs[0].Recipients, "WS 收件人应为启用成员的用户 ID")
	require.NotContains(t, wsC.msgs[0].Recipients, "owner@acme.com", "邮箱不得混入 WS 收件人")

	require.Equal(t, 1, email.calls)
	require.Equal(t, []string{"owner@acme.com"}, email.msgs[0].Recipients)
	require.Equal(t, notify.ChannelEmail, email.msgs[0].Channel)
	require.Equal(t, notify.ChannelWebSocket, wsC.msgs[0].Channel, "两渠道的消息对象不得相互覆盖")
}

// TestNotifyCancelRequest_RejectedAlsoNotifies 驳回路径同样走这套通知，收件人口径必须一致。
func TestNotifyCancelRequest_RejectedAlsoNotifies(t *testing.T) {
	svc, _, _ := seedCancelFixture(t)

	email := &recNotifier{typ: notify.ChannelEmail}
	wsC := &recNotifier{typ: notify.ChannelWebSocket}
	useNotifyRecorders(t, email, wsC)

	_, err := svc.RejectCancellation(context.Background(), "c-001", "approver-1",
		tenantDto.AdminRejectCancelReq{AdminReviewCancelReq: tenantDto.AdminReviewCancelReq{ReviewRemark: "不符合政策"}})
	require.NoError(t, err)

	require.Equal(t, 1, wsC.calls)
	require.Equal(t, []string{"u-1", "u-2"}, wsC.msgs[0].Recipients)
	require.Contains(t, wsC.msgs[0].Content, "已被驳回")
	require.Contains(t, wsC.msgs[0].Content, "不符合政策")
}

// TestNotifyCancelRequest_MemberLookupFailureStillSendsEmail 成员查询失败只该降级掉 WS，
// 不能连带吞掉邮件——邮件才是租户联系人必定可达的渠道。
func TestNotifyCancelRequest_MemberLookupFailureStillSendsEmail(t *testing.T) {
	svc, _, userRepoMock := seedCancelFixture(t)
	userRepoMock.listErr = errors.New("用户库不可用")

	email := &recNotifier{typ: notify.ChannelEmail}
	wsC := &recNotifier{typ: notify.ChannelWebSocket}
	useNotifyRecorders(t, email, wsC)

	_, err := svc.ApproveCancellation(context.Background(), "c-001", "approver-1",
		tenantDto.AdminApproveCancelReq{EffectiveDays: 7})
	require.NoError(t, err, "通知降级不得影响审批主流程")

	require.Equal(t, 1, email.calls, "成员查询失败仍应发出邮件通知")
	require.Equal(t, 0, wsC.calls, "拿不到成员 ID 时不应空转 WS")
}

// TestNotifyCancelRequest_NoMembersSkipsWS 租户没有启用成员时不应产生空转的 WS 结果，
// 只剩邮件渠道时也不该因此报错。
func TestNotifyCancelRequest_NoMembersSkipsWS(t *testing.T) {
	svc, _, userRepoMock := seedCancelFixture(t)
	userRepoMock.users["t-001"] = nil

	email := &recNotifier{typ: notify.ChannelEmail}
	wsC := &recNotifier{typ: notify.ChannelWebSocket}
	useNotifyRecorders(t, email, wsC)

	_, err := svc.ApproveCancellation(context.Background(), "c-001", "approver-1",
		tenantDto.AdminApproveCancelReq{EffectiveDays: 7})
	require.NoError(t, err)

	require.Equal(t, 1, email.calls)
	require.Equal(t, 0, wsC.calls)
}
