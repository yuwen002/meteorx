<template>
  <div class="comment-section">
    <h3>{{ t('wiki.comment.title', rootComments.length) }}</h3>

    <div class="comment-input">
      <el-input
        v-model="newComment"
        type="textarea"
        :rows="3"
        :placeholder="t('wiki.comment.inputPlaceholder')"
      />
      <el-select
        v-model="mentionPicked"
        multiple
        filterable
        remote
        reserve-keyword
        collapse-tags
        collapse-tags-tooltip
        :multiple-limit="10"
        :remote-method="searchMentionables"
        :loading="mentionLoading"
        :placeholder="t('wiki.comment.mentionPlaceholder')"
        class="mention-select"
      >
        <el-option
          v-for="u in mentionOptions"
          :key="u.id"
          :label="u.nickname || u.username"
          :value="u.id"
        />
      </el-select>
      <el-button type="primary" :loading="submitting" @click="submitComment">
        {{ t('wiki.comment.submit') }}
      </el-button>
    </div>

    <el-divider />

    <div class="comment-list">
      <div
        v-for="comment in rootComments"
        :key="comment.id"
        class="comment-item"
        :class="{ resolved: isResolved(comment) }"
      >
        <div class="comment-header">
          <span class="user-name">{{ comment.user_name || t('wiki.comment.anonymous') }}</span>
          <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
          <el-tag v-if="isResolved(comment)" type="success" size="small" effect="light">
            {{ t('wiki.comment.resolved') }}
          </el-tag>
        </div>
        <div class="comment-content">{{ comment.content }}</div>

        <!-- 表情回应 -->
        <div class="reactions">
          <span
            v-for="r in comment.reactions || []"
            :key="r.emoji"
            class="rchip"
            :class="{ mine: r.mine }"
            @click="toggleReaction(comment, r.emoji)"
          >{{ r.emoji }} {{ r.count }}</span>
          <el-popover trigger="click" placement="top" :width="200">
            <template #reference>
              <button class="radd" :title="t('wiki.comment.addReaction')">＋😊</button>
            </template>
            <div class="emoji-picker">
              <span v-for="e in EMOJIS" :key="e" class="emoji" @click="toggleReaction(comment, e)">{{ e }}</span>
            </div>
          </el-popover>
        </div>

        <div class="comment-actions">
          <el-button link type="primary" @click="replyTo(comment)">{{ t('wiki.comment.reply') }}</el-button>
          <el-button v-if="isAuthor(comment)" link type="primary" @click="handleEdit(comment)">{{ t('wiki.comment.edit') }}</el-button>
          <el-button link type="success" @click="toggleResolve(comment)">
            {{ isResolved(comment) ? t('wiki.comment.reopen') : t('wiki.comment.resolve') }}
          </el-button>
          <el-button v-if="isAuthor(comment) || canModerate" link type="danger" @click="handleDelete(comment.id)">{{ t('common.delete') }}</el-button>
        </div>

        <!-- 回复列表：后端以 replies 嵌套返回 -->
        <div v-for="reply in comment.replies || []" :key="reply.id" class="reply-item">
          <div class="comment-header">
            <span class="user-name">{{ reply.user_name || t('wiki.comment.anonymous') }}</span>
            <span class="comment-time">{{ formatTime(reply.created_at) }}</span>
          </div>
          <div class="comment-content">{{ reply.content }}</div>

          <div class="reactions">
            <span
              v-for="r in reply.reactions || []"
              :key="r.emoji"
              class="rchip"
              :class="{ mine: r.mine }"
              @click="toggleReaction(reply, r.emoji)"
            >{{ r.emoji }} {{ r.count }}</span>
            <el-popover trigger="click" placement="top" :width="200">
              <template #reference>
                <button class="radd" :title="t('wiki.comment.addReaction')">＋😊</button>
              </template>
              <div class="emoji-picker">
                <span v-for="e in EMOJIS" :key="e" class="emoji" @click="toggleReaction(reply, e)">{{ e }}</span>
              </div>
            </el-popover>
          </div>

          <div class="comment-actions">
            <el-button v-if="isAuthor(reply)" link type="primary" @click="handleEdit(reply)">{{ t('wiki.comment.edit') }}</el-button>
            <el-button v-if="isAuthor(reply) || canModerate" link type="danger" @click="handleDelete(reply.id)">{{ t('common.delete') }}</el-button>
          </div>
        </div>

        <!-- 回复输入框 -->
        <div v-if="replyingTo === comment.id" class="reply-input">
          <el-input
            v-model="replyContent"
            type="textarea"
            :rows="2"
            :placeholder="t('wiki.comment.replyPlaceholder')"
          />
          <el-button size="small" @click="cancelReply">{{ t('common.cancel') }}</el-button>
          <el-button size="small" type="primary" :loading="submitting" @click="submitReply(comment.id)">
            {{ t('wiki.comment.submitReply') }}
          </el-button>
        </div>
      </div>

      <el-empty v-if="rootComments.length === 0" :description="t('wiki.comment.empty')" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listComments,
  createComment,
  deleteComment,
  updateComment,
  toggleCommentReaction,
  resolveComment,
  type Comment,
  type ReactionItem
} from '@/api/modules/wiki'
import { getUserList, type UserItem } from '@/api/modules/user'
import { toPageResult, asArray } from '@/types/pagination'
import { useUserStore } from '@/stores/user'
import { useI18n } from '@/composables/useI18n'

const { t, locale } = useI18n()

const props = defineProps<{
  documentId: string
  // 具备节点管理权限（可解决/删除他人评论）时由父组件置为 true
  canModerate?: boolean
}>()

const userStore = useUserStore()
const currentUserId = computed(() => userStore.userInfo?.id || '')

// 评论状态与后端 model 常量对齐
const STATUS_ACTIVE = 1
const STATUS_RESOLVED = 2

// 快捷表情集合（中性 Unicode，无需翻译）
const EMOJIS = ['👍', '❤️', '🎉', '😄', '🚀', '👀']

const comments = ref<Comment[]>([])
const newComment = ref('')
const replyContent = ref('')
const replyingTo = ref<string | null>(null)
const submitting = ref(false)

// @提及选择器
const mentionPicked = ref<string[]>([])
const mentionOptions = ref<UserItem[]>([])
const mentionLoading = ref(false)

watch(
  () => props.documentId,
  async () => {
    await loadComments()
  },
  { immediate: true }
)

// 后端 ListComments 仅返回顶层评论，回复嵌套在顶层评论的 replies 中
const rootComments = computed(() => comments.value.filter((c) => !c.parent_id))

function isAuthor(c: Comment): boolean {
  return !!currentUserId.value && c.created_by === currentUserId.value
}

function isResolved(c: Comment): boolean {
  return c.status === STATUS_RESOLVED
}

async function loadComments() {
  try {
    // 空评论时后端返回 { data: null }，经拦截器回退为信封对象，直接赋值会使 rootComments 计算中的 .filter 抛错，须 asArray 兜底
    comments.value = asArray<Comment>(await listComments(props.documentId))
  } catch {
    ElMessage.error(t('wiki.comment.loadFailed'))
  }
}

async function searchMentionables(keyword: string) {
  mentionLoading.value = true
  try {
    const res = await getUserList({ page: 1, page_size: 50, keyword: keyword || undefined })
    const self = currentUserId.value
    // getUserList 返回后端原生分页包裹 { data, pagination }，须经 toPageResult 归一化出 list
    mentionOptions.value = toPageResult<UserItem>(res).list.filter((u) => u.id !== self)
  } catch {
    mentionOptions.value = []
  } finally {
    mentionLoading.value = false
  }
}

async function submitComment() {
  if (!newComment.value.trim()) {
    ElMessage.warning(t('wiki.comment.contentRequired'))
    return
  }

  submitting.value = true
  try {
    await createComment({
      document_id: props.documentId,
      content: newComment.value,
      mention_ids: mentionPicked.value.join(',')
    })
    ElMessage.success(t('wiki.comment.success'))
    newComment.value = ''
    mentionPicked.value = []
    await loadComments()
  } catch {
    ElMessage.error(t('wiki.comment.failed'))
  } finally {
    submitting.value = false
  }
}

function replyTo(comment: Comment) {
  replyingTo.value = comment.id
  replyContent.value = ''
}

function cancelReply() {
  replyingTo.value = null
  replyContent.value = ''
}

async function submitReply(parentId: string) {
  if (!replyContent.value.trim()) {
    ElMessage.warning(t('wiki.comment.replyContentRequired'))
    return
  }

  submitting.value = true
  try {
    await createComment({
      document_id: props.documentId,
      parent_id: parentId,
      content: replyContent.value
    })
    ElMessage.success(t('wiki.comment.replySuccess'))
    replyContent.value = ''
    replyingTo.value = null
    await loadComments()
  } catch {
    ElMessage.error(t('wiki.comment.replyFailed'))
  } finally {
    submitting.value = false
  }
}

function handleDelete(id: string) {
  ElMessageBox.confirm(t('wiki.comment.deleteConfirm'), t('common.tip'), {
    type: 'warning'
  })
    .then(async () => {
      await deleteComment(id)
      ElMessage.success(t('wiki.comment.deleteSuccess'))
      await loadComments()
    })
    .catch(() => {})
}

// 编辑仅作者可见：弹出输入框预填原内容，提交后回拉列表
async function handleEdit(comment: Comment) {
  let value: string
  try {
    const r = await ElMessageBox.prompt(t('wiki.comment.editTitle'), t('wiki.comment.edit'), {
      inputValue: comment.content,
      inputType: 'textarea',
      inputValidator: (v: string) => (v && v.trim() ? true : t('wiki.comment.contentRequired'))
    })
    value = ((r as { value: string }).value || '').trim()
  } catch {
    return // cancelled
  }
  if (!value) return
  try {
    await updateComment(comment.id, value)
    ElMessage.success(t('wiki.comment.editSuccess'))
    await loadComments()
  } catch {
    ElMessage.error(t('wiki.comment.editFailed'))
  }
}

// 幂等切换表情回应：以后端返回的全量聚合覆盖本地，避免前端二次计算计数
async function toggleReaction(comment: Comment, emoji: string) {
  try {
    const res = await toggleCommentReaction(comment.id, emoji)
    comment.reactions = (res?.reactions || []) as ReactionItem[]
  } catch {
    /* 反应为轻量交互，失败静默回退，不阻断阅读 */
  }
}

async function toggleResolve(comment: Comment) {
  const next = !isResolved(comment)
  try {
    await resolveComment(comment.id, next)
    comment.status = next ? STATUS_RESOLVED : STATUS_ACTIVE
    ElMessage.success(t('wiki.comment.resolveSuccess'))
  } catch {
    ElMessage.error(t('wiki.comment.resolveFailed'))
  }
}

function formatTime(time: string) {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return t('wiki.time.justNow')
  if (diff < 3600000) return t('wiki.time.minutesAgo', Math.floor(diff / 60000))
  if (diff < 86400000) return t('wiki.time.hoursAgo', Math.floor(diff / 3600000))
  return date.toLocaleDateString(locale.value)
}
</script>

<style scoped>
.comment-section {
  padding: 16px;
  background: #fff;
  border-radius: 4px;
}

.comment-section h3 {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
}

.comment-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mention-select {
  width: 100%;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-right: 4px;
}

.comment-item {
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
}

.comment-item.resolved {
  background: #f6ffed;
  border-color: #b7eb8f;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.user-name {
  font-weight: 600;
  color: #1f2937;
}

.comment-time {
  font-size: 12px;
  color: #9ca3af;
}

.comment-content {
  margin-bottom: 8px;
  color: #374151;
  line-height: 1.6;
}

.reactions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.rchip {
  cursor: pointer;
  padding: 2px 8px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  font-size: 13px;
  background: #f9fafb;
  user-select: none;
}

.rchip.mine {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.radd {
  cursor: pointer;
  padding: 2px 8px;
  border: 1px dashed #d1d5db;
  border-radius: 12px;
  font-size: 13px;
  background: transparent;
  color: #6b7280;
}

.radd:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.emoji-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.emoji-picker .emoji {
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
  padding: 4px;
  border-radius: 6px;
}

.emoji-picker .emoji:hover {
  background: #f3f4f6;
}

.comment-actions {
  display: flex;
  gap: 8px;
}

.reply-item {
  margin-left: 24px;
  margin-top: 12px;
  padding: 8px;
  background: #f9fafb;
  border-radius: 4px;
}

.reply-input {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
