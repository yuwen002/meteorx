<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.pending }}</div>
          <div class="stat-label">待办</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.in_progress }}</div>
          <div class="stat-label">进行中</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.completed }}</div>
          <div class="stat-label">已完成</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card stat-card--clickable" @click="filterOverdue">
          <div class="stat-num danger">{{ stats.overdue }}</div>
          <div class="stat-label">已逾期<span class="drill-hint">› 下钻</span></div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ (stats.completion_rate * 100).toFixed(1) }}%</div>
          <div class="stat-label">完成率</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ formatDuration(stats.avg_handle_seconds) }}</div>
          <div class="stat-label">平均耗时</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>任务管理</span>
          <div>
            <el-tooltip :disabled="aiEnabled" content="AI 能力未启用，请在服务端配置后开启" placement="bottom">
              <span>
                <el-button :disabled="!aiEnabled" @click="openSummary">
                  <el-icon><MagicStick /></el-icon>AI 摘要
                </el-button>
              </span>
            </el-tooltip>
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
          <el-option label="已逾期" value="overdue" />
        </el-select>
        <el-select v-model="priorityFilter" placeholder="优先级" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option label="低" value="low" />
          <el-option label="普通" value="normal" />
          <el-option label="高" value="high" />
          <el-option label="紧急" value="urgent" />
        </el-select>
        <el-input v-model="tagFilter" placeholder="按标签筛选" clearable style="width: 150px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-date-picker
          v-model="dueRange"
          type="daterange"
          range-separator="至"
          start-placeholder="截止起"
          end-placeholder="截止止"
          value-format="YYYY-MM-DD"
          clearable
          style="width: 240px; margin-right: 12px;"
          @change="loadList"
        />
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
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag v-if="row.recurrence" size="small" type="success" effect="plain" style="margin-right: 6px;">{{ recurrenceLabel(row.recurrence) }}</el-tag>
            <span>{{ row.title }}</span>
          </template>
        </el-table-column>
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
        <el-table-column label="负责人" min-width="100">
          <template #default="{ row }">{{ row.assignee_name || '-' }}</template>
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
        <el-form-item v-if="!editingId && aiEnabled" label="AI 解析">
          <div class="ai-parse">
            <el-input v-model="nlText" type="textarea" :rows="2" maxlength="500" placeholder="用自然语言描述任务，如“下周五前高优先级写完周报并加标签 运营”，AI 会预填下方表单" />
            <el-button class="ai-parse__btn" type="primary" plain :loading="aiParsing" @click="handleAIParse">
              <el-icon><MagicStick /></el-icon>解析预填
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" placeholder="请输入任务标题" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="任务详细描述（可选）" maxlength="2000" show-word-limit />
          <el-button v-if="aiEnabled" class="ai-inline-btn" size="small" plain :loading="aiExpanding" @click="handleAIExpand">
            <el-icon><MagicStick /></el-icon>AI 扩写/拆解
          </el-button>
        </el-form-item>
        <el-form-item label="可见范围" prop="visibility">
          <el-radio-group v-model="form.visibility">
            <el-radio-button label="personal">个人待办</el-radio-button>
            <el-radio-button label="tenant">团队协作</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="优先级">
          <div class="ai-field-row">
            <el-select v-model="form.priority" style="flex: 1;">
              <el-option label="低" value="low" />
              <el-option label="普通" value="normal" />
              <el-option label="高" value="high" />
              <el-option label="紧急" value="urgent" />
            </el-select>
            <el-button v-if="aiEnabled" size="small" plain :loading="aiSuggesting" @click="handleAISuggest">
              <el-icon><MagicStick /></el-icon>AI 建议
            </el-button>
          </div>
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
        <el-form-item label="提醒提前量">
          <el-input v-model="form.remind_before" placeholder="如 2h / 30m / 1d，留空使用全局默认" clearable />
          <div class="form-tip">仅在设置截止时间时生效；为空则按系统默认提醒窗口提醒。</div>
        </el-form-item>
        <el-form-item label="重复周期">
          <el-select v-model="form.recurrence" placeholder="不重复" style="width: 100%;">
            <el-option label="不重复" value="" />
            <el-option label="每日" value="daily" />
            <el-option label="每周" value="weekly" />
            <el-option label="每月" value="monthly" />
          </el-select>
          <div class="form-tip">选择周期需先设置截止时间；完成该任务后会自动按周期生成下一条待办。</div>
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

    <el-dialog v-model="summaryVisible" title="AI 任务摘要 / 周报" width="560px">
      <div class="summary-toolbar">
        <el-radio-group v-model="summaryScope" size="small" @change="loadSummary">
          <el-radio-button label="today">今日</el-radio-button>
          <el-radio-button label="week">本周</el-radio-button>
          <el-radio-button label="overdue">逾期</el-radio-button>
          <el-radio-button label="all">全部</el-radio-button>
        </el-radio-group>
        <span v-if="summary" class="summary-tip">共 {{ summary.total }} 项</span>
      </div>
      <div v-loading="summaryLoading" class="summary-body">
        <template v-if="summary">
          <p class="summary-text">{{ summary.text }}</p>
          <div v-if="summary.highlights?.length" class="summary-block">
            <div class="summary-block__title">重点</div>
            <ul><li v-for="(h, i) in summary.highlights" :key="'h'+i">{{ h }}</li></ul>
          </div>
          <div v-if="summary.suggested_focus?.length" class="summary-block">
            <div class="summary-block__title">优先处理建议</div>
            <ul><li v-for="(f, i) in summary.suggested_focus" :key="'f'+i">{{ f }}</li></ul>
          </div>
        </template>
        <el-empty v-else-if="!summaryLoading" description="暂无摘要" />
      </div>
      <template #footer>
        <el-button @click="summaryVisible = false">关闭</el-button>
        <el-button type="primary" :loading="summaryLoading" @click="loadSummary">重新生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus, Files, MagicStick } from '@element-plus/icons-vue'
import {
  getTaskList,
  getTask,
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
  type TaskRecurrence,
} from '@/api/modules/task'
import {
  getAICapabilities,
  parseTaskByNL,
  suggestTaskFields,
  expandTaskDescription,
  summarizeTasks,
  type TaskSummary,
  type TaskSummaryScope,
} from '@/api/modules/task-ai'
import { toPageResult } from '@/types/pagination'

const route = useRoute()

const loading = ref(false)
const list = ref<TaskItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref<'' | TaskStatus | 'overdue'>('')
const priorityFilter = ref<'' | TaskPriority>('')
const visibilityFilter = ref<'' | TaskVisibility>('')
const tagFilter = ref('')
const dueRange = ref<[string, string] | null>(null)

const stats = reactive<TaskStats>({ pending: 0, in_progress: 0, completed: 0, overdue: 0, total: 0, completion_rate: 0, avg_handle_seconds: 0 })

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
function recurrenceLabel(r: string) {
  return ({ daily: '每日', weekly: '每周', monthly: '每月' } as Record<string, string>)[r] || r
}
// formatDuration 将秒数格式化为“X天/X小时/X分钟”可读时长；0 或无效返回 “-”。
function formatDuration(seconds: number) {
  if (!seconds || seconds <= 0) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}天${h}小时`
  if (h > 0) return `${h}小时${m}分钟`
  if (m > 0) return `${m}分钟`
  return `${Math.floor(seconds)}秒`
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
      tag: tagFilter.value || undefined,
      due_from: dueRange.value?.[0] || undefined,
      due_to: dueRange.value?.[1] || undefined,
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
  remind_before: '',
  recurrence: '' as '' | TaskRecurrence,
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
    form.remind_before = row.remind_before || ''
    form.recurrence = (row.recurrence as '' | TaskRecurrence) || ''
    form.tags = [...(row.tags || [])]
  } else {
    form.title = ''
    form.description = ''
    form.visibility = 'personal'
    form.priority = 'normal'
    form.assignee_id = ''
    form.due_date = ''
    form.remind_before = ''
    form.recurrence = ''
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
        remind_before: form.remind_before || undefined,
        recurrence: form.recurrence,
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
    } catch (e) {
      ElMessage.error((e as Error).message || '操作失败')
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
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  }
}

async function handleReopen(row: TaskItem) {
  try {
    await reopenTask(row.id, 'pending')
    ElMessage.success('已重新打开')
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
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
  } catch (e) {
    ElMessage.error((e as Error).message || '批量完成失败')
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

// filterOverdue 点击"已逾期"统计卡：将状态筛选切为逾期伪状态并重查（再次点击取消）。
function filterOverdue() {
  statusFilter.value = statusFilter.value === 'overdue' ? '' : 'overdue'
  page.value = 1
  loadList()
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
  } catch (e) {
    ElMessage.error((e as Error).message || '恢复失败')
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

// ===== AI 辅助（可选叠加层，未启用时相关入口隐藏/禁用） =====
const aiEnabled = ref(false)
const aiModel = ref('')
const nlText = ref('')
const aiParsing = ref(false)
const aiSuggesting = ref(false)
const aiExpanding = ref(false)

const summaryVisible = ref(false)
const summaryLoading = ref(false)
const summaryScope = ref<TaskSummaryScope>('all')
const summary = ref<TaskSummary | null>(null)

// loadAICapabilities 探测后端 AI 可用性；失败静默降级为不可用（不阻断任务页）。
async function loadAICapabilities() {
  try {
    const cap = await getAICapabilities()
    aiEnabled.value = cap.enabled
    aiModel.value = cap.model
  } catch {
    aiEnabled.value = false
  }
}

// handleAIParse 自然语言解析并预填新建表单（仅新建态使用）。
async function handleAIParse() {
  const text = nlText.value.trim()
  if (!text) {
    ElMessage.warning('请先输入任务描述')
    return
  }
  aiParsing.value = true
  try {
    const draft = await parseTaskByNL(text)
    if (draft.title) form.title = draft.title
    if (draft.priority) form.priority = draft.priority
    if (draft.due_date) form.due_date = draft.due_date
    if (draft.recurrence) form.recurrence = draft.recurrence
    if (draft.tags?.length) form.tags = [...draft.tags]
    ElMessage.success('已根据描述预填，请确认后创建')
  } catch (e) {
    ElMessage.error((e as Error).message || 'AI 解析失败')
  } finally {
    aiParsing.value = false
  }
}

// handleAISuggest 依据当前标题/描述给出优先级与标签建议。
async function handleAISuggest() {
  if (!form.title.trim()) {
    ElMessage.warning('请先填写标题')
    return
  }
  aiSuggesting.value = true
  try {
    const s = await suggestTaskFields(form.title, form.description)
    if (s.priority) form.priority = s.priority
    if (s.tags?.length) form.tags = Array.from(new Set([...form.tags, ...s.tags]))
    if (s.reason) ElMessage.info(`建议：${s.reason}`)
  } catch (e) {
    ElMessage.error((e as Error).message || 'AI 建议失败')
  } finally {
    aiSuggesting.value = false
  }
}

// handleAIExpand 扩写描述并追加拆解步骤。
async function handleAIExpand() {
  if (!form.title.trim()) {
    ElMessage.warning('请先填写标题')
    return
  }
  aiExpanding.value = true
  try {
    const e2 = await expandTaskDescription(form.title, form.description)
    let text = e2.description || ''
    if (e2.steps?.length) text += (text ? '\n\n' : '') + e2.steps.map((s, i) => `${i + 1}. ${s}`).join('\n')
    if (text) form.description = text
    else ElMessage.info('AI 未返回可用内容')
  } catch (e) {
    ElMessage.error((e as Error).message || 'AI 扩写失败')
  } finally {
    aiExpanding.value = false
  }
}

// openSummary 打开摘要对话框并按当前范围生成。
function openSummary() {
  summaryVisible.value = true
  loadSummary()
}

async function loadSummary() {
  summaryLoading.value = true
  try {
    summary.value = await summarizeTasks(summaryScope.value)
  } catch (e) {
    summary.value = null
    ElMessage.error((e as Error).message || '生成摘要失败')
  } finally {
    summaryLoading.value = false
  }
}

onMounted(() => {
  loadList()
  loadAICapabilities()
})

/**
 * 从收件箱的任务提醒（link 带 task_id）定位任务。
 * 不依赖当前列表里恰好有它：筛选条件、分页与列表刷新都可能把它挡在外面。
 */
async function openTaskById(id: string) {
  try {
    openDialog(await getTask(id))
  } catch {
    // 失败提示由拦截器统一弹出（任务可能已删除，或已不属于当前用户）
  }
}

// 已在任务页时再点另一条提醒只会改 query（不重新 mount），所以用 watch
watch(
  () => route.query.task_id,
  id => {
    if (typeof id === 'string' && id) openTaskById(id)
  },
  { immediate: true }
)
</script>

<style scoped>
.stat-row {
  margin-bottom: 16px;
}
.form-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 2px;
}
.stat-card {
  text-align: center;
}
.stat-card--clickable {
  cursor: pointer;
  transition: box-shadow 0.2s;
}
.stat-card--clickable:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.12);
}
.drill-hint {
  margin-left: 4px;
  font-size: 11px;
  color: #c0c4cc;
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
.ai-parse {
  width: 100%;
}
.ai-parse__btn {
  margin-top: 8px;
}
.ai-inline-btn {
  margin-top: 6px;
}
.ai-field-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.summary-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.summary-tip {
  font-size: 12px;
  color: #909399;
}
.summary-body {
  min-height: 120px;
}
.summary-text {
  white-space: pre-wrap;
  line-height: 1.6;
  color: #303133;
  margin: 0 0 12px;
}
.summary-block__title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 4px;
  color: #606266;
}
.summary-block ul {
  margin: 0 0 12px;
  padding-left: 20px;
  color: #606266;
  line-height: 1.6;
}
</style>
