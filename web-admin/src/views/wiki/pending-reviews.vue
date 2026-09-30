<template>
  <div class="pending-reviews-page">
    <div class="page-header">
      <h2>待审核文档</h2>
      <el-button :icon="Refresh" @click="loadData">刷新</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="items"
      stripe
      style="width: 100%"
    >
      <el-table-column prop="title" label="文档标题" min-width="200">
        <template #default="{ row }">
          <el-link type="primary" @click="goToDocument(row)">{{ row.title }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="space_name" label="所属空间" width="160" />
      <el-table-column prop="publish_status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.publish_status)" size="small">
            {{ statusText(row.publish_status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="submitted_by" label="提交人" width="120" />
      <el-table-column prop="submitted_at" label="提交时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.submitted_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="success" @click="handleApprove(row)">通过</el-button>
          <el-button size="small" type="danger" @click="handleReject(row)">驳回</el-button>
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
      <p>文档：<strong>{{ actionDoc?.title }}</strong></p>
      <el-input
        v-model="actionComment"
        type="textarea"
        :rows="4"
        placeholder="输入审核意见（可选）"
        style="margin-top: 12px"
      />
      <template #footer>
        <el-button @click="actionDialogVisible = false">取消</el-button>
        <el-button
          :type="actionType === 'approve' ? 'success' : 'danger'"
          :loading="submitting"
          @click="submitAction"
        >
          确认
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

const router = useRouter()

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
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
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
  actionDialogTitle.value = '审核通过'
  actionComment.value = ''
  actionDialogVisible.value = true
}

function handleReject(row: PendingReviewItem) {
  actionDoc.value = row
  actionType.value = 'reject'
  actionDialogTitle.value = '驳回文档'
  actionComment.value = ''
  actionDialogVisible.value = true
}

async function submitAction() {
  if (!actionDoc.value) return
  submitting.value = true
  try {
    const fn = actionType.value === 'approve' ? approveDocument : rejectDocument
    await fn(actionDoc.value.document_id, actionComment.value)
    ElMessage.success(actionType.value === 'approve' ? '已通过' : '已驳回')
    actionDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

function statusText(s: PublishStatus): string {
  const map: Record<string, string> = {
    draft: '草稿',
    pending_review: '待审核',
    published: '已发布',
    rejected: '已驳回',
    archived: '已归档'
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

function formatTime(t?: string): string {
  if (!t) return '-'
  return t.replace('T', ' ').substring(0, 19)
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