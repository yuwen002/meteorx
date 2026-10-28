<template>
  <div class="page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.pending }}</div>
          <div class="stat-label">{{ t('taskPage.stat.pending') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.in_progress }}</div>
          <div class="stat-label">{{ t('taskPage.stat.in_progress') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ stats.completed }}</div>
          <div class="stat-label">{{ t('taskPage.stat.completed') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card stat-card--clickable" @click="filterOverdue">
          <div class="stat-num danger">{{ stats.overdue }}</div>
          <div class="stat-label">{{ t('taskPage.stat.overdue') }}<span class="drill-hint">{{ t('taskPage.stat.drillHint') }}</span></div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ (stats.completion_rate * 100).toFixed(1) }}%</div>
          <div class="stat-label">{{ t('taskPage.stat.completionRate') }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="never" class="stat-card">
          <div class="stat-num">{{ formatDuration(stats.avg_handle_seconds) }}</div>
          <div class="stat-label">{{ t('taskPage.stat.avgHandle') }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('taskPage.title') }}</span>
          <div>
            <el-tooltip :disabled="aiEnabled" :content="t('taskPage.aiDisabledTip')" placement="bottom">
              <span>
                <el-button :disabled="!aiEnabled" @click="openSummary">
                  <el-icon><MagicStick /></el-icon>{{ t('taskPage.aiSummary') }}
                </el-button>
              </span>
            </el-tooltip>
            <el-button @click="openTrash">
              <el-icon><Files /></el-icon>{{ t('taskPage.trash') }}
            </el-button>
            <el-button type="primary" @click="openDialog()">
              <el-icon><Plus /></el-icon>{{ t('taskPage.create') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-model="visibilityFilter" :placeholder="t('taskPage.filter.scopePlaceholder')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('taskPage.scope.all')" value="" />
          <el-option :label="t('taskPage.scope.personal')" value="personal" />
          <el-option :label="t('taskPage.scope.tenant')" value="tenant" />
        </el-select>
        <el-select v-model="statusFilter" :placeholder="t('taskPage.filter.statusPlaceholder')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('taskPage.stat.pending')" value="pending" />
          <el-option :label="t('taskPage.stat.in_progress')" value="in_progress" />
          <el-option :label="t('taskPage.stat.completed')" value="completed" />
          <el-option :label="t('taskPage.stat.overdue')" value="overdue" />
        </el-select>
        <el-select v-model="priorityFilter" :placeholder="t('taskPage.filter.priorityPlaceholder')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('taskPage.priority.low')" value="low" />
          <el-option :label="t('taskPage.priority.normal')" value="normal" />
          <el-option :label="t('taskPage.priority.high')" value="high" />
          <el-option :label="t('taskPage.priority.urgent')" value="urgent" />
        </el-select>
        <el-input v-model="tagFilter" :placeholder="t('taskPage.filter.tagPlaceholder')" clearable style="width: 150px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-date-picker
          v-model="dueRange"
          type="daterange"
          :range-separator="t('taskPage.filter.rangeSeparator')"
          :start-placeholder="t('taskPage.filter.dueStart')"
          :end-placeholder="t('taskPage.filter.dueEnd')"
          value-format="YYYY-MM-DD"
          clearable
          style="width: 240px; margin-right: 12px;"
          @change="loadList"
        />
        <el-input v-model="keyword" :placeholder="t('taskPage.filter.keywordPlaceholder')" clearable style="width: 200px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList">{{ t('common.query') }}</el-button>
      </div>

      <div v-if="selected.length" class="batch-bar">
        <span class="batch-tip">{{ t('taskPage.batch.selected', selected.length) }}</span>
        <el-dropdown trigger="click" class="batch-dropdown" @command="handleBatchStatus">
          <el-button plain size="small">{{ t('taskPage.batch.changeStatus') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="pending">{{ t('taskPage.stat.pending') }}</el-dropdown-item>
              <el-dropdown-item command="in_progress">{{ t('taskPage.stat.in_progress') }}</el-dropdown-item>
              <el-dropdown-item command="completed">{{ t('taskPage.stat.completed') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button size="small" @click="openBatchAssign">{{ t('taskPage.batch.assign') }}</el-button>
        <el-button type="success" size="small" @click="handleBatchComplete">{{ t('taskPage.batch.complete') }}</el-button>
        <el-button type="danger" size="small" @click="handleBatchDelete">{{ t('taskPage.batch.delete') }}</el-button>
      </div>

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="45" />
        <el-table-column prop="title" :label="t('taskPage.col.title')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag v-if="row.recurrence" size="small" type="success" effect="plain" style="margin-right: 6px;">{{ recurrenceLabel(row.recurrence) }}</el-tag>
            <span>{{ row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.scope')" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.visibility === 'tenant' ? 'warning' : 'info'">
              {{ row.visibility === 'tenant' ? t('taskPage.scope.tagTenant') : t('taskPage.scope.tagPersonal') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.status')" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.priority')" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="priorityTagType(row.priority)" size="small" effect="plain">{{ priorityLabel(row.priority) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.tags')" min-width="140">
          <template #default="{ row }">
            <el-tag v-for="tag in row.tags" :key="tag" size="small" type="info" style="margin-right: 4px;">{{ tag }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.assignee')" min-width="100">
          <template #default="{ row }">{{ row.assignee_name || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('taskPage.col.dueDate')" min-width="150">
          <template #default="{ row }">{{ row.due_date || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('taskPage.col.createdAt')" min-width="150" />
        <el-table-column :label="t('taskPage.col.action')" width="220" align="center" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'completed'" type="success" link size="small" @click="handleComplete(row)">{{ t('taskPage.action.complete') }}</el-button>
            <el-button v-else type="warning" link size="small" @click="handleReopen(row)">{{ t('taskPage.action.reopen') }}</el-button>
            <el-button type="primary" link size="small" @click="openDialog(row)">{{ t('taskPage.action.edit') }}</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('taskPage.action.delete') }}</el-button>
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

    <el-dialog v-model="dialogVisible" :title="editingId ? t('taskPage.dialog.editTitle') : t('taskPage.dialog.createTitle')" width="520px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item v-if="!editingId && aiEnabled" :label="t('taskPage.form.aiParseLabel')">
          <div class="ai-parse">
            <el-input v-model="nlText" type="textarea" :rows="2" maxlength="500" :placeholder="t('taskPage.form.aiParsePlaceholder')" />
            <el-button class="ai-parse__btn" type="primary" plain :loading="aiParsing" @click="handleAIParse">
              <el-icon><MagicStick /></el-icon>{{ t('taskPage.form.aiParseBtn') }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.titleLabel')" prop="title">
          <el-input v-model="form.title" :placeholder="t('taskPage.form.titlePlaceholder')" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('taskPage.form.descLabel')">
          <el-input v-model="form.description" type="textarea" :rows="3" :placeholder="t('taskPage.form.descPlaceholder')" maxlength="2000" show-word-limit />
          <el-button v-if="aiEnabled" class="ai-inline-btn" size="small" plain :loading="aiExpanding" @click="handleAIExpand">
            <el-icon><MagicStick /></el-icon>{{ t('taskPage.form.aiExpandBtn') }}
          </el-button>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.visibilityLabel')" prop="visibility">
          <el-radio-group v-model="form.visibility">
            <el-radio-button label="personal">{{ t('taskPage.scope.personal') }}</el-radio-button>
            <el-radio-button label="tenant">{{ t('taskPage.scope.tenant') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.priorityLabel')">
          <div class="ai-field-row">
            <el-select v-model="form.priority" style="flex: 1;">
              <el-option :label="t('taskPage.priority.low')" value="low" />
              <el-option :label="t('taskPage.priority.normal')" value="normal" />
              <el-option :label="t('taskPage.priority.high')" value="high" />
              <el-option :label="t('taskPage.priority.urgent')" value="urgent" />
            </el-select>
            <el-button v-if="aiEnabled" size="small" plain :loading="aiSuggesting" @click="handleAISuggest">
              <el-icon><MagicStick /></el-icon>{{ t('taskPage.form.aiSuggestBtn') }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item v-if="form.visibility === 'tenant'" :label="t('taskPage.form.assigneeLabel')">
          <el-input v-model="form.assignee_id" :placeholder="t('taskPage.form.assigneePlaceholder')" clearable />
        </el-form-item>
        <el-form-item :label="t('taskPage.form.dueLabel')">
          <el-date-picker
            v-model="form.due_date"
            type="datetime"
            :placeholder="t('taskPage.form.duePlaceholder')"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%;"
          />
        </el-form-item>
        <el-form-item :label="t('taskPage.form.remindLabel')">
          <el-input v-model="form.remind_before" :placeholder="t('taskPage.form.remindPlaceholder')" clearable />
          <div class="form-tip">{{ t('taskPage.form.remindTip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.recurrenceLabel')">
          <el-select v-model="form.recurrence" :placeholder="t('taskPage.form.recurrencePlaceholder')" style="width: 100%;">
            <el-option :label="t('taskPage.recurrence.none')" value="" />
            <el-option :label="t('taskPage.recurrence.daily')" value="daily" />
            <el-option :label="t('taskPage.recurrence.weekly')" value="weekly" />
            <el-option :label="t('taskPage.recurrence.monthly')" value="monthly" />
          </el-select>
          <div class="form-tip">{{ t('taskPage.form.recurrenceTip') }}</div>
        </el-form-item>
        <el-form-item :label="t('taskPage.form.tagsLabel')">
          <el-select v-model="form.tags" multiple filterable allow-create default-first-option :placeholder="t('taskPage.form.tagsPlaceholder')" style="width: 100%;">
            <el-option v-for="tag in form.tags" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitLoading" @click="submitForm">{{ editingId ? t('taskPage.form.submitSave') : t('taskPage.form.submitCreate') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="assignVisible" :title="t('taskPage.assign.title')" width="440px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item :label="t('taskPage.assign.idLabel')">
          <el-input v-model="assignUserId" :placeholder="t('taskPage.assign.idPlaceholder')" clearable />
          <div class="form-tip">{{ t('taskPage.assign.tip') }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="assignLoading" @click="submitBatchAssign">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="trashVisible" :title="t('taskPage.trash.title')" size="600px">
      <el-table v-loading="trashLoading" :data="trashList" stripe style="width: 100%;">
        <el-table-column prop="title" :label="t('taskPage.trash.colTitle')" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('taskPage.trash.colDeletedAt')" min-width="150">
          <template #default="{ row }">{{ row.deleted_at || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('taskPage.trash.colAction')" width="150" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleRestore(row)">{{ t('taskPage.trash.restore') }}</el-button>
            <el-button type="danger" link size="small" @click="handlePermanentDelete(row)">{{ t('taskPage.trash.permanentDelete') }}</el-button>
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

    <el-dialog v-model="summaryVisible" :title="t('taskPage.summary.title')" width="560px">
      <div class="summary-toolbar">
        <el-radio-group v-model="summaryScope" size="small" @change="loadSummary">
          <el-radio-button label="today">{{ t('taskPage.summary.scope.today') }}</el-radio-button>
          <el-radio-button label="week">{{ t('taskPage.summary.scope.week') }}</el-radio-button>
          <el-radio-button label="overdue">{{ t('taskPage.summary.scope.overdue') }}</el-radio-button>
          <el-radio-button label="all">{{ t('taskPage.summary.scope.all') }}</el-radio-button>
        </el-radio-group>
        <span v-if="summary" class="summary-tip">{{ t('taskPage.summary.totalCount', summary.total) }}</span>
      </div>
      <div v-loading="summaryLoading" class="summary-body">
        <template v-if="summary">
          <p class="summary-text">{{ summary.text }}</p>
          <div v-if="summary.highlights?.length" class="summary-block">
            <div class="summary-block__title">{{ t('taskPage.summary.highlights') }}</div>
            <ul><li v-for="(h, i) in summary.highlights" :key="'h'+i">{{ h }}</li></ul>
          </div>
          <div v-if="summary.suggested_focus?.length" class="summary-block">
            <div class="summary-block__title">{{ t('taskPage.summary.suggestedFocus') }}</div>
            <ul><li v-for="(f, i) in summary.suggested_focus" :key="'f'+i">{{ f }}</li></ul>
          </div>
        </template>
        <el-empty v-else-if="!summaryLoading" :description="t('taskPage.summary.empty')" />
      </div>
      <template #footer>
        <el-button @click="summaryVisible = false">{{ t('taskPage.summary.close') }}</el-button>
        <el-button type="primary" :loading="summaryLoading" @click="loadSummary">{{ t('taskPage.summary.regenerate') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus, Files, MagicStick, ArrowDown } from '@element-plus/icons-vue'
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
  batchUpdateTaskStatus,
  batchAssignTasks,
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
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
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
  return ({ pending: t('taskPage.stat.pending'), in_progress: t('taskPage.stat.in_progress'), completed: t('taskPage.stat.completed') } as Record<string, string>)[s] || s
}
function statusTagType(s: string) {
  return ({ pending: 'info', in_progress: 'warning', completed: 'success' } as Record<string, string>)[s] || 'info'
}
function priorityLabel(p: string) {
  return ({ low: t('taskPage.priority.low'), normal: t('taskPage.priority.normal'), high: t('taskPage.priority.high'), urgent: t('taskPage.priority.urgent') } as Record<string, string>)[p] || p
}
function priorityTagType(p: string) {
  return ({ low: 'info', normal: '', high: 'warning', urgent: 'danger' } as Record<string, string>)[p] || ''
}
function recurrenceLabel(r: string) {
  return ({ daily: t('taskPage.recurrence.daily'), weekly: t('taskPage.recurrence.weekly'), monthly: t('taskPage.recurrence.monthly') } as Record<string, string>)[r] || r
}
// formatDuration 将秒数格式化为“X天/X小时/X分钟”可读时长；0 或无效返回 “-”。
function formatDuration(seconds: number) {
  if (!seconds || seconds <= 0) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return t('taskPage.duration.days', d, h)
  if (h > 0) return t('taskPage.duration.hours', h, m)
  if (m > 0) return t('taskPage.duration.minutes', m)
  return t('taskPage.duration.seconds', Math.floor(seconds))
}

async function loadStats() {
  try {
    const data = await getTaskStats(visibilityFilter.value ? { visibility: visibilityFilter.value } : undefined)
    Object.assign(stats, data)
  } catch (e) {
    console.error(t('taskPage.loadStatsFailed'), e)
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
    ElMessage.error(t('taskPage.loadListFailed'))
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
const rules = computed<FormRules>(() => ({
  title: [{ required: true, message: t('taskPage.form.titleRequired'), trigger: 'blur' }],
}))

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
        ElMessage.success(t('taskPage.updated'))
      } else {
        await createTask(payload)
        ElMessage.success(t('taskPage.created'))
      }
      dialogVisible.value = false
      loadList()
    } catch (e) {
      ElMessage.error((e as Error).message || t('taskPage.opFailed'))
    } finally {
      submitLoading.value = false
    }
  })
}

async function handleComplete(row: TaskItem) {
  try {
    await completeTask(row.id)
    ElMessage.success(t('taskPage.completed'))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.opFailed'))
  }
}

async function handleReopen(row: TaskItem) {
  try {
    await reopenTask(row.id, 'pending')
    ElMessage.success(t('taskPage.reopened'))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.opFailed'))
  }
}

async function handleDelete(row: TaskItem) {
  try {
    await ElMessageBox.confirm(t('taskPage.deleteConfirm', row.title), t('taskPage.deleteTitle'), { type: 'warning' })
    await deleteTask(row.id)
    ElMessage.success(t('taskPage.deleted'))
    loadList()
  } catch { /* cancelled */ }
}

async function handleBatchComplete() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    const res = await batchCompleteTasks(ids)
    ElMessage.success(t('taskPage.batchCompleted', res.affected))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.batchCompleteFailed'))
  }
}

// 批量改状态/批量指派：复用后端已有的 /tasks/batch/status 与 /tasks/batch/assign，
// 两者均逐条做归属校验，只返回实际生效数，失败项不阻断整体。
const assignVisible = ref(false)
const assignLoading = ref(false)
const assignUserId = ref('')

async function handleBatchStatus(status: string) {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    const res = await batchUpdateTaskStatus({ ids, status: status as TaskStatus })
    ElMessage.success(t('taskPage.batchStatusChanged', statusLabel(status), res.affected))
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.batchStatusFailed'))
  }
}

function openBatchAssign() {
  assignUserId.value = ''
  assignVisible.value = true
}

async function submitBatchAssign() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  const assignee = assignUserId.value.trim()
  if (!assignee) {
    ElMessage.warning(t('taskPage.assigneeIdRequired'))
    return
  }
  assignLoading.value = true
  try {
    const res = await batchAssignTasks({ ids, assignee_id: assignee })
    ElMessage.success(t('taskPage.batchAssigned', res.affected))
    assignVisible.value = false
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.batchAssignFailed'))
  } finally {
    assignLoading.value = false
  }
}

async function handleBatchDelete() {
  const ids = selected.value.map((r) => r.id)
  if (!ids.length) return
  try {
    await ElMessageBox.confirm(t('taskPage.batchDeleteConfirm', ids.length), t('taskPage.batchDeleteTitle'), { type: 'warning' })
    const res = await batchDeleteTasks(ids)
    ElMessage.success(t('taskPage.batchDeleted', res.affected))
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
    ElMessage.error(t('taskPage.loadTrashFailed'))
  } finally {
    trashLoading.value = false
  }
}

async function handleRestore(row: TaskItem) {
  try {
    await restoreTask(row.id)
    ElMessage.success(t('taskPage.restored'))
    loadTrash()
    loadList()
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.restoreFailed'))
  }
}

async function handlePermanentDelete(row: TaskItem) {
  try {
    await ElMessageBox.confirm(t('taskPage.permanentDeleteConfirm', row.title), t('taskPage.permanentDeleteTitle'), { type: 'warning' })
    await permanentDeleteTask(row.id)
    ElMessage.success(t('taskPage.permanentDeleted'))
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
    ElMessage.warning(t('taskPage.aiDescRequired'))
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
    ElMessage.success(t('taskPage.aiPrefilled'))
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.aiParseFailed'))
  } finally {
    aiParsing.value = false
  }
}

// handleAISuggest 依据当前标题/描述给出优先级与标签建议。
async function handleAISuggest() {
  if (!form.title.trim()) {
    ElMessage.warning(t('taskPage.aiTitleRequired'))
    return
  }
  aiSuggesting.value = true
  try {
    const s = await suggestTaskFields(form.title, form.description)
    if (s.priority) form.priority = s.priority
    if (s.tags?.length) form.tags = Array.from(new Set([...form.tags, ...s.tags]))
    if (s.reason) ElMessage.info(t('taskPage.aiSuggestion', s.reason))
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.aiSuggestFailed'))
  } finally {
    aiSuggesting.value = false
  }
}

// handleAIExpand 扩写描述并追加拆解步骤。
async function handleAIExpand() {
  if (!form.title.trim()) {
    ElMessage.warning(t('taskPage.aiTitleRequired'))
    return
  }
  aiExpanding.value = true
  try {
    const e2 = await expandTaskDescription(form.title, form.description)
    let text = e2.description || ''
    if (e2.steps?.length) text += (text ? '\n\n' : '') + e2.steps.map((s, i) => `${i + 1}. ${s}`).join('\n')
    if (text) form.description = text
    else ElMessage.info(t('taskPage.aiNoContent'))
  } catch (e) {
    ElMessage.error((e as Error).message || t('taskPage.aiExpandFailed'))
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
    ElMessage.error((e as Error).message || t('taskPage.summaryFailed'))
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
.batch-dropdown {
  margin-right: 12px;
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
