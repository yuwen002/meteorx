<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('nav.export') }}</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>{{ t('exportPage.create') }}
          </el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-model="resourceFilter" :placeholder="t('exportPage.colResource')" clearable style="width: 150px; margin-right: 12px;" @change="loadList">
          <el-option v-for="opt in resourceOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
        <el-select v-model="statusFilter" :placeholder="t('exportPage.colStatus')" clearable style="width: 130px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('exportPage.status.pending')" value="pending" />
          <el-option :label="t('exportPage.status.running')" value="running" />
          <el-option :label="t('exportPage.status.completed')" value="completed" />
          <el-option :label="t('exportPage.status.failed')" value="failed" />
        </el-select>
        <el-button type="primary" @click="loadList">{{ t('common.query') }}</el-button>
        <el-button @click="loadList">{{ t('common.refresh') }}</el-button>
      </div>

      <el-alert
      type="info"
      :closable="false"
      show-icon
      :title="t('exportPage.alert')"
      style="margin-bottom: 16px;"
      />

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;">
        <el-table-column :label="t('exportPage.colResource')" width="130">
          <template #default="{ row }">{{ resourceLabel(row.resource) }}</template>
        </el-table-column>
        <el-table-column :label="t('exportPage.colFormat')" width="90" align="center">
          <template #default="{ row }">{{ (row.format || 'csv').toUpperCase() }}</template>
        </el-table-column>
        <el-table-column :label="t('exportPage.colStatus')" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('exportPage.colRows')" width="110" align="center">
          <template #default="{ row }">{{ row.status === 'completed' ? row.rows : '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('exportPage.colCreatedAt')" min-width="180" />
        <el-table-column prop="expires_at" :label="t('exportPage.colExpiresAt')" min-width="180" />
        <el-table-column :label="t('exportPage.colDesc')" min-width="160">
          <template #default="{ row }">
            <el-text v-if="row.status === 'failed'" type="danger">{{ row.error_msg || t('exportPage.failFallback') }}</el-text>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('exportPage.colAction')" width="160" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              link
              size="small"
              :disabled="row.status !== 'completed'"
              @click="handleDownload(row)"
            >{{ t('exportPage.download') }}</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog v-model="createDialogVisible" :title="t('exportPage.create')" width="480px" :close-on-click-modal="false">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="90px">
        <el-form-item :label="t('exportPage.colResource')" prop="resource">
          <el-select v-model="createForm.resource" :placeholder="t('exportPage.resourcePlaceholder')" style="width: 100%;">
            <el-option v-for="opt in resourceOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('exportPage.formatLabel')" prop="format">
          <el-select v-model="createForm.format" :placeholder="t('exportPage.formatPlaceholder')" style="width: 100%;">
            <el-option v-for="opt in formatOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">{{ t('exportPage.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  getExportList,
  createExport,
  downloadExport,
  deleteExport,
  type ExportJobItem,
  type ExportResource,
  type ExportFormat
} from '@/api/modules/export'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const resourceOptions = computed<{ label: string; value: ExportResource }[]>(() => [
  { label: t('exportPage.resUser'), value: 'user' },
  { label: t('exportPage.resTask'), value: 'task' },
  { label: t('exportPage.resAudit'), value: 'audit_log' },
  { label: t('exportPage.resWiki'), value: 'wiki_document' }
])

const formatOptions = computed<{ label: string; value: ExportFormat }[]>(() => [
  { label: t('exportPage.fmtCsv'), value: 'csv' },
  { label: t('exportPage.fmtXlsx'), value: 'xlsx' },
  { label: 'JSON', value: 'json' },
  { label: t('exportPage.fmtPdf'), value: 'pdf' }
])

function resourceLabel(resource: string) {
  return resourceOptions.value.find(o => o.value === resource)?.label || resource
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    pending: t('exportPage.status.pending'),
    running: t('exportPage.status.running'),
    completed: t('exportPage.status.completed'),
    failed: t('exportPage.status.failed')
  }
  return map[status] || status
}

function statusTagType(status: string) {
  const map: Record<string, string> = { pending: 'info', running: 'warning', completed: 'success', failed: 'danger' }
  return map[status] || 'info'
}

const loading = ref(false)
const list = ref<ExportJobItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const resourceFilter = ref('')
const statusFilter = ref('')

async function loadList() {
  loading.value = true
  try {
    const raw = await getExportList({
      page: page.value,
      page_size: pageSize.value,
      resource: resourceFilter.value || undefined,
      status: statusFilter.value || undefined
    })
    const result = toPageResult<ExportJobItem>(raw)
    list.value = result.list
    total.value = result.total
    scheduleAutoRefresh()
  } catch {
    ElMessage.error(t('exportPage.loadFailed'))
  } finally {
    loading.value = false
  }
}

// 存在进行中的任务时轮询刷新，全部结束后停止
let refreshTimer: ReturnType<typeof setTimeout> | null = null
function scheduleAutoRefresh() {
  if (refreshTimer) {
    clearTimeout(refreshTimer)
    refreshTimer = null
  }
  const hasActive = list.value.some(j => j.status === 'pending' || j.status === 'running')
  if (hasActive) {
    refreshTimer = setTimeout(loadList, 3000)
  }
}
onUnmounted(() => {
  if (refreshTimer) clearTimeout(refreshTimer)
})

const createDialogVisible = ref(false)
const createFormRef = ref<FormInstance>()
const createLoading = ref(false)
const createForm = reactive({ resource: '' as ExportResource | '', format: 'csv' as ExportFormat })
const createRules = computed<FormRules>(() => ({
  resource: [{ required: true, message: t('exportPage.resRequired'), trigger: 'change' }]
}))

function openCreateDialog() {
  createForm.resource = ''
  createDialogVisible.value = true
}

async function submitCreate() {
  if (!createFormRef.value) return
  await createFormRef.value.validate(async (valid) => {
    if (!valid) return
    createLoading.value = true
    try {
      await createExport({ resource: createForm.resource as ExportResource, format: createForm.format })
      ElMessage.success(t('exportPage.created'))
      createDialogVisible.value = false
      page.value = 1
      loadList()
    } catch (e) {
      ElMessage.error((e as Error).message || t('exportPage.createFailed'))
    } finally {
      createLoading.value = false
    }
  })
}

async function handleDownload(row: ExportJobItem) {
  try {
    const blob = await downloadExport(row.id)
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${row.resource}-${row.id}.${row.format || 'csv'}`
    link.click()
    window.URL.revokeObjectURL(url)
  } catch {
    ElMessage.error(t('exportPage.downloadFailed'))
  }
}

async function handleDelete(row: ExportJobItem) {
  try {
    await ElMessageBox.confirm(t('exportPage.deleteConfirm'), t('exportPage.deleteTitle'), { type: 'warning' })
    await deleteExport(row.id)
    ElMessage.success(t('exportPage.deleted'))
    loadList()
  } catch { /* user cancelled, or the request failed (error toast is shown centrally by the request.ts interceptor) */ }
}

onMounted(loadList)
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
