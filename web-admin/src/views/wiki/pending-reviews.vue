<template>
  <div class="pending-reviews-page">
    <div class="page-header">
      <h2>{{ t('wiki.pending.title') }}</h2>
      <el-button :icon="Refresh" @click="loadData">{{ t('common.refresh') }}</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="items"
      stripe
      style="width: 100%"
    >
      <el-table-column prop="title" :label="t('wiki.pending.col.docTitle')" min-width="200">
        <template #default="{ row }">
          <el-link type="primary" @click="goToDocument(row)">{{ row.title }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="space_name" :label="t('wiki.pending.col.space')" width="160" />
      <el-table-column prop="publish_status" :label="t('common.status')" width="120">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.publish_status)" size="small">
            {{ statusText(row.publish_status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="submitted_by" :label="t('wiki.pending.col.submitter')" width="120" />
      <el-table-column prop="submitted_at" :label="t('wiki.pending.col.submittedAt')" width="180">
        <template #default="{ row }">
          {{ formatTime(row.submitted_at) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="success" @click="handleApprove(row)">{{ t('wiki.pending.approve') }}</el-button>
          <el-button size="small" type="danger" @click="handleReject(row)">{{ t('wiki.pub.menu.reject') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="loadData"
      />
    </div>

    <el-dialog v-model="actionDialogVisible" :title="actionDialogTitle" width="480px">
      <p>{{ t('wiki.pending.docLabel') }}<strong>{{ actionDoc?.title }}</strong></p>
      <el-input
        v-model="actionComment"
        type="textarea"
        :rows="4"
        :placeholder="t('wiki.pub.commentPlaceholder')"
        style="margin-top: 12px"
      />
      <template #footer>
        <el-button @click="actionDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button
          :type="actionType === 'approve' ? 'success' : 'danger'"
          :loading="submitting"
          @click="submitAction"
        >
          {{ t('wiki.pub.confirmOk') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import {
  type PendingReviewItem,
  type PublishStatus,
  listPendingReviews,
  approveDocument,
  rejectDocument
} from '@/api/modules/wiki'
import { useI18n } from '@/composables/useI18n'

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const items = ref<PendingReviewItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const actionDialogVisible = ref(false)
const actionDialogTitle = ref('')
const actionComment = ref('')
const actionType = ref<'approve' | 'reject'>('approve')
const actionDoc = ref<PendingReviewItem | null>(null)
const submitting = ref(false)

onMounted(() => loadData())

async function loadData() {
  loading.value = true
  try {
    const res = await listPendingReviews(page.value, pageSize.value)
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('wiki.pending.loadFailed'))
  } finally {
    loading.value = false
  }
}

function goToDocument(row: PendingReviewItem) {
  router.push(`/wiki/spaces/${row.space_id}`)
}

function handleApprove(row: PendingReviewItem) {
  actionDoc.value = row
  actionType.value = 'approve'
  actionDialogTitle.value = t('wiki.pub.menu.approve')
  actionComment.value = ''
  actionDialogVisible.value = true
}

function handleReject(row: PendingReviewItem) {
  actionDoc.value = row
  actionType.value = 'reject'
  actionDialogTitle.value = t('wiki.pub.rejectDoc')
  actionComment.value = ''
  actionDialogVisible.value = true
}

async function submitAction() {
  if (!actionDoc.value) return
  submitting.value = true
  try {
    const fn = actionType.value === 'approve' ? approveDocument : rejectDocument
    await fn(actionDoc.value.document_id, actionComment.value)
    ElMessage.success(actionType.value === 'approve' ? t('wiki.pending.approved') : t('wiki.pending.rejected'))
    actionDialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error((e as Error)?.message || t('common.failed'))
  } finally {
    submitting.value = false
  }
}

function statusText(s: PublishStatus): string {
  const map: Record<string, string> = {
    draft: t('wiki.pub.status.draft'),
    pending_review: t('wiki.pub.status.pending'),
    published: t('wiki.pub.status.published'),
    rejected: t('wiki.pub.status.rejected'),
    archived: t('wiki.pub.status.archived')
  }
  return map[s] || s
}

function statusTagType(s: PublishStatus): string {
  const map: Record<string, string> = {
    draft: 'info',
    pending_review: 'warning',
    published: 'success',
    rejected: 'danger',
    archived: ''
  }
  return map[s] || 'info'
}

function formatTime(value?: string): string {
  if (!value) return '-'
  return value.replace('T', ' ').substring(0, 19)
}
</script>

<style scoped>
.pending-reviews-page {
  padding: 20px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 18px;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>