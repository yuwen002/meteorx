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
      <el-button type="primary" :loading="submitting" @click="submitComment">
        {{ t('wiki.comment.submit') }}
      </el-button>
    </div>

    <el-divider />

    <div class="comment-list">
      <div v-for="comment in rootComments" :key="comment.id" class="comment-item">
        <div class="comment-header">
          <span class="user-name">{{ comment.user_name || t('wiki.comment.anonymous') }}</span>
          <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
        </div>
        <div class="comment-content">{{ comment.content }}</div>
        <div class="comment-actions">
          <el-button link type="primary" @click="replyTo(comment)">{{ t('wiki.comment.reply') }}</el-button>
          <el-button link type="danger" @click="handleDelete(comment.id)">{{ t('common.delete') }}</el-button>
        </div>

        <!-- 回复列表：后端以 replies 嵌套返回 -->
        <div v-for="reply in comment.replies || []" :key="reply.id" class="reply-item">
          <div class="comment-header">
            <span class="user-name">{{ reply.user_name || t('wiki.comment.anonymous') }}</span>
            <span class="comment-time">{{ formatTime(reply.created_at) }}</span>
          </div>
          <div class="comment-content">{{ reply.content }}</div>
          <div class="comment-actions">
            <el-button link type="danger" @click="handleDelete(reply.id)">{{ t('common.delete') }}</el-button>
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
  type Comment
} from '@/api/modules/wiki'
import { useI18n } from '@/composables/useI18n'

const { t, locale } = useI18n()

const props = defineProps<{
  documentId: string
}>()

const comments = ref<Comment[]>([])
const newComment = ref('')
const replyContent = ref('')
const replyingTo = ref<string | null>(null)
const submitting = ref(false)

watch(
  () => props.documentId,
  async () => {
    await loadComments()
  },
  { immediate: true }
)

// 后端 ListComments 仅返回顶层评论，回复嵌套在顶层评论的 replies 中
const rootComments = computed(() =>
  comments.value.filter((c) => !c.parent_id)
)

async function loadComments() {
  try {
    comments.value = await listComments(props.documentId)
  } catch (error) {
    ElMessage.error(t('wiki.comment.loadFailed'))
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
      content: newComment.value
    })
    ElMessage.success(t('wiki.comment.success'))
    newComment.value = ''
    await loadComments()
  } catch (error) {
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
  } catch (error) {
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

.comment-header {
  display: flex;
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