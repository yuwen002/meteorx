<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.pending }}</div>
          <div class="stat-label">待办</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.in_progress }}</div>
          <div class="stat-label">进行中</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.completed }}</div>
          <div class="stat-label">已完成</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num danger">{{ stats.overdue }}</div>
          <div class="stat-label">已逾期</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>任务管理</span>
          <div>
            <el-button @click="openTrash">
              <el-icon><Files /></el-icon>回收站
            </el-button>
            <el-button type="primary" @click="openDialog()">
              <el-icon><Plus /></el-icon>新建任务
            </el-button>
          </div>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-model="visibilityFilter" placeholder="范围" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option label="全部可见" value="" />
          <el-option label="个人待办" value="personal" />
          <el-option label="团队协作" value="tenant" />
        </el-select>
        <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option label="待办" value="pending" />
          <el-option label="进行中" value="in_progress" />
          <el-option label="已完成" value="completed" />
        </el-select>
        <el-select v-model="priorityFilter" placeholder="优先级" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option label="低" value="low" />
          <el-option label="普通" value="normal" />
          <el-option label="高" value="high" />
          <el-option label="紧急" value="urgent" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索标题/描述" clearable style="width: 200px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList">查询</el-button>
      </div>

      <div v-if="selected.length" class="batch-bar">
        <span class="batch-tip">已选择 {{ selected.length }} 项</span>
        <el-button type="success" size="small" @click="handleBatchComplete">批量完成</el-button>
        <el-button type="danger" size="small" @click="handleBatchDelete">批量删除</el-button>
      </div>

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="45" />
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column label="范围" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.visibility === 'tenant' ? 'warning' : 'info'">
              {{ row.visibility === 'tenant' ? '团队' : '个人' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="优先级" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="priorityTagType(row.priority)" size="small" effect="plain">{{ priorityLabel(row.priority) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="标签" min-width="140">
          <template #default="{ row }">
            <el-tag v-for="t in row.tags" :key="t" size="small" type="info" style="margin-right: 4px;">{{ t }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="截止时间" min-width="150">
          <template #default="{ row }">{{ row.due_date || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="150" />
        <el-table-column label="操作" width="220" align="center" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'completed'" type="success" link size="small" @click="handleComplete(row)">完成</el-button>
            <el-button v-else type="warning" link size="small" @click="handleReopen(row)">重开</el-button>
            <el-button type="primary" link size="small" @click="openDialog(row)">编辑</el-button>
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

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑任务' : '新建任务'" width="520px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" placeholder="请输入任务标题" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="任务详细描述（可选）" maxlength="2000" show-word-limit />
        </el-form-item>
        <el-form-item label="可见范围" prop="visibility">
          <el-radio-group v-model="form.visibility">
            <el-radio-button label="personal">个人待办</el-radio-button>
            <el-radio-button label="tenant">团队协作</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="优先级">
          <el-select v-model="form.priority" style="width: 100%;">
            <el-option label="低" value="low" />
            <el-option label="普通" value="normal" />
            <el-option label="高" value="high" />
            <el-option label="紧急" value="urgent" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.visibility === 'tenant'" label="负责人">
          <el-input v-model="form.assignee_id" placeholder="负责人用户 ID（可选，留空则自己负责）" clearable />
        </el-form-item>
        <el-form-item label="截止时间">
          <el-date-picker
            v-model="form.due_date"
            type="datetime"
            placeholder="选择截止时间（可选）"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%;"
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-select v-model="form.tags" multiple filterable allow-create default-first-option placeholder="输入后回车添加标签" style="width: 100%;">
            <el-option v-for="t in form.tags" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="submitForm">{{ editingId ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="trashVisible" title="回收站" size="600px">
      <el-table v-loading="trashLoading" :data="trashList" stripe style="width: 100%;">
        <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
        <el-table-column label="删除时间" min-width="150">
          <template #default="{ row }">{{ row.deleted_at || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleRestore(row)">恢复</el-button>
            <el-button type="danger" link size="small" @click="handlePermanentDelete(row)">彻底删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="trashPage"
          v-model:page-size="trashPageSize"
          :total="trashTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadTrash"
          @current-change="loadTrash"
        />
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus, Files } from '@element-plus/icons-vue'
import {
  getTaskList,
  createTask,
  updateTask,
  completeTask,
  reopenTask,
  deleteTask,
  getTaskStats,
  getTaskTrash,
  restoreTask,
  permanentDeleteTask,
  batchCompleteTasks,
  batchDeleteTasks,
  type TaskItem,
  type TaskStats,
  type TaskStatus,
  type TaskPriority,
  type TaskVisibility,
} from '@/api/modules/task'
import { toPageResult } from '@/types/pagination'

const loading = ref(false)
const list = ref<TaskItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref<'' | TaskStatus>('')
const priorityFilter = ref<'' | TaskPriority>('')
const visibilityFilter = ref<'' | TaskVisibility>('')

const stats = reactive<TaskStats>({ pending: 0, in_progress: 0, completed: 0, overdue: 0, total: 0 })

// 批量选择
const selected = ref<TaskItem[]>([])
function handleSelectionChange(rows: TaskItem[]) {
  selected.value = rows
}

// 回收站
const trashVisible = ref(false)
const trashLoading = ref(false)
const trashList = ref<TaskItem[]>([])
const trashPage = ref(1)
const trashPageSize = ref(10)
const trashTotal = ref(0)

function statusLabel(s: string) {
  return ({ pending: '待办', in_progress: '进行中', completed: '已完成' } as Record<string, string>)[s] || s
}
function statusTagType(s: string) {
  return ({ pending: 'info', in_progress: 'warning', completed: 'success' } as Record<string, string>)[s] || 'info'
}
function priorityLabel(p: string) {
  return ({ low: '低', normal: '普通', high: '高', urgent: '紧急' } as Record<string, string>)[p] || p
}
function priorityTagType(p: string) {
  return ({ low: 'info', normal: '', high: 'warning', urgent: 'danger' } as Record<string, string>)[p] || ''
}

async function loadStats() {
  try {
    const data = await getTaskStats(visibilityFilter.value ? { visibility: visibilityFilter.value } : undefined)
    Object.assign(stats, data)
  } catch (e) {
    console.error('加载任务统计失败', e)
  }
}

async function loadList() {
  loading.value = true
  try {
    const raw = await getTaskList({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
      status: statusFilter.value || undefined,
      priority: priorityFilter.value || undefined,
      visibility: visibilityFilter.value || undefined,
    })
    const result = toPageResult<TaskItem>(raw)
    list.value = result.list
    total.value = result.total
    loadStats()
  } catch (e) {
    ElMessage.error('获取任务列表失败')
  } finally {
    loading.value = false
  }
}

const dialogVisible = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const editingId = ref('')
const form = reactive({
  title: '',
  description: '',
  visibility: 'personal' as TaskVisibility,
  priority: 'normal' as TaskPriority,
  assignee_id: '',
  due_date: '',
  tags: [] as string[],
})
const rules: FormRules = {
  title: [{ required: true, message: '请输入任务标题', trigger: 'blur' }],
}

function openDialog(row?: TaskItem) {
  editingId.value = row?.id || ''
  if (row) {
    form.title = row.title
    form.description = row.description
    form.visibility = row.visibility
    form.priority = row.priority
    form.assignee_id = row.assignee_id === row.creator_id ? '' : row.assignee_id
    form.due_date = row.due_date || ''
    form.tags = [...(row.tags || [])]
  } else {
    form.title = ''
    form.description = ''
    form.visibility = 'personal'
    form.priority = 'normal'
    form.assignee_id = ''
    form.due_date = ''
    form.tags = []
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      const payload = {
        title: form.title,
        description: form.description,
        visibility: form.visibility,
        priority: form.priority,
        assignee_id: form.visibility === 'tenant' ? form.assignee_id : undefined,
        due_date: form.due_date || undefined,
        tags: form.tags,
      }
      if (editingId.value) {
        await updateTask(editingId.value, payload)
        ElMessage.success('任务已更新')
      } else {
        await createTask(payload)
        ElMessage.success('任务已创建')
      }
      dialogVisible.value = false
      loadList()
    } catch (e: any) {
      ElMessage.error(e.message || '操作失败')
    } finally {
      submitLoading.value = false
    }
  })
}

async function handleComplete(row: TaskItem) {
  try {
    await completeTask(row.id)
    ElMessage.success('已完成')
    loadList()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleReopen(row: TaskItem) {
  try {
    await reopenTask(row.id, 'pending')
    ElMessage.success('已重新打开')
    loadList()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDelete(row: TaskItem) {
  try {
    await ElMessageBox.confirm(`确定删除任务「${row.title}」？`, '删除确认', { type: 'warning' })
    await deleteTask(row.id)
    ElMessage.success('已删除')
    loadList()
  } catch { /* cancelled */ }
}

async function handleBatchComplete() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    const res = await batchCompleteTasks(ids)
    ElMessage.success(`已完成 ${res.affected} 项`)
    loadList()
  } catch (e: any) {
    ElMessage.error(e.message || '批量完成失败')
  }
}

async function handleBatchDelete() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${ids.length} 项任务？`, '批量删除确认', { type: 'warning' })
    const res = await batchDeleteTasks(ids)
    ElMessage.success(`已删除 ${res.affected} 项`)
    loadList()
  } catch { /* cancelled */ }
}

function openTrash() {
  trashVisible.value = true
  trashPage.value = 1
  loadTrash()
}

async function loadTrash() {
  trashLoading.value = true
  try {
    const raw = await getTaskTrash({ page: trashPage.value, page_size: trashPageSize.value })
    const result = toPageResult<TaskItem>(raw)
    trashList.value = result.list
    trashTotal.value = result.total
  } catch {
    ElMessage.error('获取回收站列表失败')
  } finally {
    trashLoading.value = false
  }
}

async function handleRestore(row: TaskItem) {
  try {
    await restoreTask(row.id)
    ElMessage.success('已恢复')
    loadTrash()
    loadList()
  } catch (e: any) {
    ElMessage.error(e.message || '恢复失败')
  }
}

async function handlePermanentDelete(row: TaskItem) {
  try {
    await ElMessageBox.confirm(`彻底删除任务「${row.title}」？此操作不可恢复。`, '确认', { type: 'warning' })
    await permanentDeleteTask(row.id)
    ElMessage.success('已彻底删除')
    loadTrash()
  } catch { /* cancelled */ }
}

onMounted(() => {
  loadList()
})
</script>

<style scoped>
.stat-row {
  margin-bottom: 16px;
}
.stat-card {
  text-align: center;
}
.stat-num {
  font-size: 28px;
  font-weight: 600;
  color: #3b82f6;
}
.stat-num.danger {
  color: #ef4444;
}
.stat-label {
  margin-top: 4px;
  color: #6b7280;
  font-size: 13px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.batch-bar {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.batch-tip {
  margin-right: 12px;
  color: #6b7280;
  font-size: 13px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
