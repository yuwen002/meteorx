<template>
  <div class="comment-section">
    <h3>评论 ({{ comments.length }})</h3>

    <div class="comment-input">
      <el-input
        v-model="newComment"
        type="textarea"
        :rows="3"
        placeholder="输入评论内容... 使用 @用户名 可以提及他人"
      />
      <el-button type="primary" :loading="submitting" @click="submitComment">
        发表评论
      </el-button>
    </div>

    <el-divider />

    <div class="comment-list">
      <div v-for="comment in rootComments" :key="comment.id" class="comment-item">
        <div class="comment-header">
          <span class="user-name">{{ comment.user_name || '匿名用户' }}</span>
          <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
        </div>
        <div class="comment-content">{{ comment.content }}</div>
        <div class="comment-actions">
          <el-button link type="primary" @click="replyTo(comment)">回复</el-button>
          <el-button link type="danger" @click="handleDelete(comment.id)">删除</el-button>
        </div>

        <!-- 回复列表 -->
        <div v-for="reply in getReplies(comment.id)" :key="reply.id" class="reply-item">
          <div class="comment-header">
            <span class="user-name">{{ reply.user_name || '匿名用户' }}</span>
            <span class="comment-time">{{ formatTime(reply.created_at) }}</span>
          </div>
          <div class="comment-content">{{ reply.content }}</div>
          <div class="comment-actions">
            <el-button link type="danger" @click="handleDelete(reply.id)">删除</el-button>
          </div>
        </div>

        <!-- 回复输入框 -->
        <div v-if="replyingTo === comment.id" class="reply-input">
          <el-input
            v-model="replyContent"
            type="textarea"
            :rows="2"
            placeholder="输入回复内容..."
          />
          <el-button size="small" @click="cancelReply">取消</el-button>
          <el-button size="small" type="primary" :loading="submitting" @click="submitReply(comment.id)">
            发表回复
          </el-button>
        </div>
      </div>

      <el-empty v-if="comments.length === 0" description="暂无评论" />
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

const rootComments = computed(() =>
  comments.value.filter((c) => !c.parent_id)
)

function getReplies(parentId: string) {
  return comments.value.filter((c) => c.parent_id === parentId)
}

async function loadComments() {
  try {
    comments.value = await listComments(props.documentId)
  } catch (error) {
    ElMessage.error('加载评论失败')
  }
}

async function submitComment() {
  if (!newComment.value.trim()) {
    ElMessage.warning('请输入评论内容')
    return
  }

  submitting.value = true
  try {
    await createComment({
      document_id: props.documentId,
      content: newComment.value
    })
    ElMessage.success('评论成功')
    newComment.value = ''
    await loadComments()
  } catch (error) {
    ElMessage.error('评论失败')
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
    ElMessage.warning('请输入回复内容')
    return
  }

  submitting.value = true
  try {
    await createComment({
      document_id: props.documentId,
      parent_id: parentId,
      content: replyContent.value
    })
    ElMessage.success('回复成功')
    replyContent.value = ''
    replyingTo.value = null
    await loadComments()
  } catch (error) {
    ElMessage.error('回复失败')
  } finally {
    submitting.value = false
  }
}

function handleDelete(id: string) {
  ElMessageBox.confirm('确定要删除此评论吗？', '提示', {
    type: 'warning'
  })
    .then(async () => {
      await deleteComment(id)
      ElMessage.success('删除成功')
      await loadComments()
    })
    .catch(() => {})
}

function formatTime(time: string) {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  return date.toLocaleDateString('zh-CN')
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