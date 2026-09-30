<template>
  <div class="publish-workflow">
    <el-tag :type="statusTagType" size="small" class="status-tag">
      {{ statusText }}
    </el-tag>
    <el-dropdown trigger="click" @command="handleCommand">
      <el-button size="small" :icon="MoreFilled" />
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-if="canSubmitReview"
            command="submit-review"
            :icon="Upload"
          >
            提交审核
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canApprove"
            command="approve"
            :icon="CircleCheck"
          >
            审核通过
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canReject"
            command="reject"
            :icon="CircleClose"
          >
            驳回
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canPublish"
            command="publish"
            :icon="Promotion"
          >
            直接发布
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canUnpublish"
            command="unpublish"
            :icon="RefreshLeft"
          >
            取消发布
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canArchive"
            command="archive"
            :icon="Box"
          >
            归档
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canBackToDraft"
            command="unpublish"
            :icon="RefreshLeft"
          >
            退回草稿
          </el-dropdown-item>
          <el-dropdown-item
            divided
            command="view-comments"
            :icon="ChatDotRound"
          >
            审核记录
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <el-dialog v-model="commentDialogVisible" :title="commentDialogTitle" width="480px">
      <el-input
        v-model="commentText"
        type="textarea"
        :rows="4"
        placeholder="输入审核意见（可选）"
      />
      <template #footer>
        <el-button @click="commentDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAction">确认</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="commentsDialogVisible" title="审核记录" width="600px">
      <el-timeline v-if="reviewComments.length">
        <el-timeline-item
          v-for="c in reviewComments"
          :key="c.id"
          :timestamp="formatTime(c.created_at)"
          placement="top"
          :type="actionTagType(c.action)"
        >
          <div class="comment-item">
            <el-tag :type="actionTagType(c.action)" size="small" class="action-tag">
              {{ actionText(c.action) }}
            </el-tag>
            <span class="reviewer-name">{{ c.reviewer_name || c.reviewer_id }}</span>
            <p v-if="c.content" class="comment-content">{{ c.content }}</p>
          </div>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-else description="暂无审核记录" :image-size="60" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  MoreFilled, Upload, CircleCheck, CircleClose,
  Promotion, RefreshLeft, Box, ChatDotRound
} from '@element-plus/icons-vue'
import {
  type PublishStatus,
  type ReviewComment,
  submitForReview,
  approveDocument,
  rejectDocument,
  publishDocument,
  unpublishDocument,
  archiveDocument,
  listReviewComments
} from '@/api/modules/wiki'

const props = defineProps<{
  documentId: string
  publishStatus: PublishStatus
  canReview: boolean
  canPublish: boolean
}>()

const emit = defineEmits<{
  (e: 'status-changed', status: PublishStatus): void
}>()

const status = computed(() => props.publishStatus || 'draft')

const statusText = computed(() => {
  const map: Record<string, string> = {
    draft: '草稿',
    pending_review: '待审核',
    published: '已发布',
    rejected: '已驳回',
    archived: '已归档'
  }
  return map[status.value] || status.value
})

const statusTagType = computed(() => {
  const map: Record<string, string> = {
    draft: 'info',
    pending_review: 'warning',
    published: 'success',
    rejected: 'danger',
    archived: ''
  }
  return map[status.value] || 'info'
})

const canSubmitReview = computed(() =>
  status.value === 'draft' || status.value === 'rejected'
)

const canApprove = computed(() =>
  props.canReview && status.value === 'pending_review'
)

const canReject = computed(() =>
  props.canReview && status.value === 'pending_review'
)

const canPublish = computed(() =>
  props.canPublish && status.value === 'draft'
)

const canUnpublish = computed(() =>
  props.canPublish && status.value === 'published'
)

const canArchive = computed(() =>
  props.canPublish && status.value === 'published'
)

const canBackToDraft = computed(() =>
  props.canPublish && (status.value === 'archived' || status.value === 'pending_review')
)

const commentDialogVisible = ref(false)
const commentDialogTitle = ref('')
const commentText = ref('')
const pendingCommand = ref('')
const submitting = ref(false)

const commentsDialogVisible = ref(false)
const reviewComments = ref<ReviewComment[]>([])

function handleCommand(cmd: string) {
  if (cmd === 'view-comments') {
    loadComments()
    return
  }

  if (['submit-review', 'approve', 'reject'].includes(cmd)) {
    pendingCommand.value = cmd
    const titleMap: Record<string, string> = {
      'submit-review': '提交审核',
      'approve': '审核通过',
      'reject': '驳回文档'
    }
    commentDialogTitle.value = titleMap[cmd]
    commentText.value = ''
    commentDialogVisible.value = true
    return
  }

  const confirmMap: Record<string, string> = {
    'publish': '确定直接发布该文档？',
    'unpublish': '确定取消发布该文档？',
    'archive': '确定归档该文档？'
  }

  ElMessageBox.confirm(confirmMap[cmd] || '确定执行此操作？', '提示', {
    type: 'warning'
  }).then(() => {
    executeAction(cmd)
  }).catch(() => {})
}

async function submitAction() {
  submitting.value = true
  try {
    await executeAction(pendingCommand.value, commentText.value)
    commentDialogVisible.value = false
  } finally {
    submitting.value = false
  }
}

async function executeAction(cmd: string, comment?: string) {
  try {
    let result
    switch (cmd) {
      case 'submit-review':
        result = await submitForReview(props.documentId, comment)
        break
      case 'approve':
        result = await approveDocument(props.documentId, comment)
        break
      case 'reject':
        result = await rejectDocument(props.documentId, comment)
        break
      case 'publish':
        result = await publishDocument(props.documentId)
        break
      case 'unpublish':
        result = await unpublishDocument(props.documentId)
        break
      case 'archive':
        result = await archiveDocument(props.documentId)
        break
    }
    if (result?.publish_status) {
      emit('status-changed', result.publish_status)
    }
    ElMessage.success('操作成功')
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  }
}

async function loadComments() {
  try {
    reviewComments.value = await listReviewComments(props.documentId)
    commentsDialogVisible.value = true
  } catch (e: any) {
    ElMessage.error(e?.message || '加载审核记录失败')
  }
}

function actionText(action: string): string {
  const map: Record<string, string> = {
    submit: '提交审核',
    approve: '审核通过',
    reject: '驳回',
    publish: '直接发布',
    archive: '归档'
  }
  return map[action] || action
}

function actionTagType(action: string): string {
  const map: Record<string, string> = {
    submit: 'warning',
    approve: 'success',
    reject: 'danger',
    publish: 'success',
    archive: 'info'
  }
  return map[action] || 'info'
}

function formatTime(t: string): string {
  if (!t) return ''
  return t.replace('T', ' ').substring(0, 19)
}
</script>

<style scoped>
.publish-workflow {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-tag {
  font-weight: 500;
}

.comment-item {
  line-height: 1.6;
}

.action-tag {
  margin-right: 8px;
}

.reviewer-name {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.comment-content {
  margin: 4px 0 0;
  color: var(--el-text-color-regular);
  font-size: 13px;
}
</style>