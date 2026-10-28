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
            {{ t('wiki.pub.menu.submitReview') }}
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canApprove"
            command="approve"
            :icon="CircleCheck"
          >
            {{ t('wiki.pub.menu.approve') }}
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canReject"
            command="reject"
            :icon="CircleClose"
          >
            {{ t('wiki.pub.menu.reject') }}
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canPublish"
            command="publish"
            :icon="Promotion"
          >
            {{ t('wiki.pub.menu.publish') }}
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canUnpublish"
            command="unpublish"
            :icon="RefreshLeft"
          >
            {{ t('wiki.pub.menu.unpublish') }}
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canArchive"
            command="archive"
            :icon="Box"
          >
            {{ t('wiki.pub.menu.archive') }}
          </el-dropdown-item>
          <el-dropdown-item
            v-if="canBackToDraft"
            command="unpublish"
            :icon="RefreshLeft"
          >
            {{ t('wiki.pub.menu.backToDraft') }}
          </el-dropdown-item>
          <el-dropdown-item
            divided
            command="view-comments"
            :icon="ChatDotRound"
          >
            {{ t('wiki.pub.menu.records') }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <el-dialog v-model="commentDialogVisible" :title="commentDialogTitle" width="480px">
      <el-input
        v-model="commentText"
        type="textarea"
        :rows="4"
        :placeholder="t('wiki.pub.commentPlaceholder')"
      />
      <template #footer>
        <el-button @click="commentDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAction">{{ t('wiki.pub.confirmOk') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="commentsDialogVisible" :title="t('wiki.pub.menu.records')" width="600px">
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
      <el-empty v-else :description="t('wiki.pub.recordsEmpty')" :image-size="60" />
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
import { useI18n } from '@/composables/useI18n'
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

const { t } = useI18n()

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
    draft: t('wiki.pub.status.draft'),
    pending_review: t('wiki.pub.status.pending'),
    published: t('wiki.pub.status.published'),
    rejected: t('wiki.pub.status.rejected'),
    archived: t('wiki.pub.status.archived')
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
      'submit-review': t('wiki.pub.menu.submitReview'),
      'approve': t('wiki.pub.menu.approve'),
      'reject': t('wiki.pub.rejectDoc')
    }
    commentDialogTitle.value = titleMap[cmd]
    commentText.value = ''
    commentDialogVisible.value = true
    return
  }

  const confirmMap: Record<string, string> = {
    'publish': t('wiki.pub.confirmPublish'),
    'unpublish': t('wiki.pub.confirmUnpublish'),
    'archive': t('wiki.pub.confirmArchive')
  }

  ElMessageBox.confirm(confirmMap[cmd] || t('wiki.pub.confirmDefault'), t('common.tip'), {
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
    ElMessage.success(t('common.success'))
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('common.failed'))
  }
}

async function loadComments() {
  try {
    reviewComments.value = await listReviewComments(props.documentId)
    commentsDialogVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('wiki.pub.loadRecordsFailed'))
  }
}

function actionText(action: string): string {
  const map: Record<string, string> = {
    submit: t('wiki.pub.menu.submitReview'),
    approve: t('wiki.pub.menu.approve'),
    reject: t('wiki.pub.menu.reject'),
    publish: t('wiki.pub.menu.publish'),
    archive: t('wiki.pub.menu.archive')
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

function formatTime(time: string): string {
  if (!time) return ''
  return time.replace('T', ' ').substring(0, 19)
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