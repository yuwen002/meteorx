<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>数据导出</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>新建导出任务
          </el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-model="resourceFilter" placeholder="资源类型" clearable style="width: 150px; margin-right: 12px;" @change="loadList">
          <el-option v-for="opt in resourceOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
        <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 130px; margin-right: 12px;" @change="loadList">
          <el-option label="排队中" value="pending" />
          <el-option label="导出中" value="running" />
          <el-option label="已完成" value="completed" />
          <el-option label="失败" value="failed" />
        </el-select>
        <el-button type="primary" @click="loadList">查询</el-button>
        <el-button @click="loadList">刷新</el-button>
      </div>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="导出为异步任务，创建后在此列表查看进度；完成后即可下载 CSV 文件（UTF-8 带 BOM，Excel 直接打开不乱码）。"
        style="margin-bottom: 16px;"
      />

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;">
        <el-table-column label="资源类型" width="130">
          <template #default="{ row }">{{ resourceLabel(row.resource) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="导出行数" width="110" align="center">
          <template #default="{ row }">{{ row.status === 'completed' ? row.rows : '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="180" />
        <el-table-column prop="expires_at" label="过期时间" min-width="180" />
        <el-table-column label="说明" min-width="160">
          <template #default="{ row }">
            <el-text v-if="row.status === 'failed'" type="danger">{{ row.error_msg || '导出失败' }}</el-text>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              link
              size="small"
              :disabled="row.status !== 'completed'"
              @click="handleDownload(row)"
            >下载</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
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

    <el-dialog v-model="createDialogVisible" title="新建导出任务" width="480px" :close-on-click-modal="false">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="90px">
        <el-form-item label="资源类型" prop="resource">
          <el-select v-model="createForm.resource" placeholder="请选择要导出的数据" style="width: 100%;">
            <el-option v-for="opt in resourceOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  getExportList,
  createExport,
  downloadExport,
  deleteExport,
  type ExportJobItem,
  type ExportResource
} from '@/api/modules/export'
import { toPageResult } from '@/types/pagination'

const resourceOptions: { label: string; value: ExportResource }[] = [
  { label: '用户', value: 'user' },
  { label: '任务', value: 'task' },
  { label: '审计日志', value: 'audit_log' },
  { label: 'Wiki 文档', value: 'wiki_document' }
]

function resourceLabel(resource: string) {
  return resourceOptions.find(o => o.value === resource)?.label || resource
}

function statusLabel(status: string) {
  const map: Record<string, string> = { pending: '排队中', running: '导出中', completed: '已完成', failed: '失败' }
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
    ElMessage.error('获取导出任务列表失败')
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
const createForm = reactive({ resource: '' as ExportResource | '' })
const createRules: FormRules = {
  resource: [{ required: true, message: '请选择资源类型', trigger: 'change' }]
}

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
      await createExport({ resource: createForm.resource as ExportResource })
      ElMessage.success('导出任务已创建，正在后台生成')
      createDialogVisible.value = false
      page.value = 1
      loadList()
    } catch (e) {
      ElMessage.error((e as Error).message || '创建导出任务失败')
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
    link.download = `${row.resource}-${row.id}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
  } catch {
    ElMessage.error('下载失败，文件可能已过期')
  }
}

async function handleDelete(row: ExportJobItem) {
  try {
    await ElMessageBox.confirm('确定删除此导出任务及其文件？', '删除确认', { type: 'warning' })
    await deleteExport(row.id)
    ElMessage.success('已删除')
    loadList()
  } catch { /* 用户取消 */ }
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
