<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input v-model="query.keyword" placeholder="搜索套餐名/编码" clearable style="width: 240px" @keyup.enter="loadList" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 130px">
          <el-option label="启用" :value="1" />
          <el-option label="停用" :value="0" />
        </el-select>
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog"><el-icon><Plus /></el-icon>新增套餐</el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="name" label="套餐名称" min-width="130" />
        <el-table-column prop="code" label="编码" min-width="140">
          <template #default="{ row }">
            <el-tag type="primary" effect="plain" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="用户数上限" width="120">
          <template #default="{ row }">
            <span v-if="row.user_limit === -1">不限</span>
            <span v-else>{{ row.user_limit }} 人</span>
          </template>
        </el-table-column>
        <el-table-column label="存储配额" width="120">
          <template #default="{ row }">
            <span v-if="row.storage_limit === -1">不限</span>
            <span v-else>{{ formatSize(row.storage_limit) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="API速率" width="120">
          <template #default="{ row }">
            <span v-if="row.api_rate_limit === -1">不限</span>
            <span v-else>{{ row.api_rate_limit }} 次/分</span>
          </template>
        </el-table-column>
        <el-table-column label="Wiki空间" width="110">
          <template #default="{ row }">
            <span v-if="row.wiki_space_limit === -1">不限</span>
            <span v-else>{{ row.wiki_space_limit }} 个</span>
          </template>
        </el-table-column>
        <el-table-column label="月费" width="110">
          <template #default="{ row }">
            <span v-if="row.price === 0">
              <el-tag type="success" size="small" effect="plain">免费</el-tag>
            </span>
            <span v-else>¥{{ row.price }}</span>
          </template>
        </el-table-column>
        <el-table-column label="使用租户数" width="110">
          <template #default="{ row }">
            {{ row.subscriber_cnt ?? 0 }}
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="180" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link :type="row.status === 1 ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.status === 1 ? '停用' : '启用' }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增套餐' : '编辑套餐'" width="560px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="套餐名称" prop="name"><el-input v-model="form.name" placeholder="如：标准版" /></el-form-item>
        <el-form-item label="套餐编码" prop="code">
          <el-input v-model="form.code" :disabled="dialogMode === 'edit'" placeholder="如：standard" />
        </el-form-item>
        <el-form-item label="用户数上限" prop="user_limit">
          <el-input-number v-model="form.user_limit" :min="-1" :max="999999" />
          <span class="tip">-1 表示不限</span>
        </el-form-item>
        <el-form-item label="存储配额(GB)" prop="storage_limit">
          <el-input-number v-model="form.storage_limit" :min="-1" :max="102400" :precision="0" />
          <span class="tip">-1 表示不限，单位为 GB</span>
        </el-form-item>
        <el-form-item label="API速率(分)" prop="api_rate_limit">
          <el-input-number v-model="form.api_rate_limit" :min="-1" :max="1000000" :step="60" />
          <span class="tip">每分钟请求数上限，-1 表示不限</span>
        </el-form-item>
        <el-form-item label="Wiki空间数" prop="wiki_space_limit">
          <el-input-number v-model="form.wiki_space_limit" :min="-1" :max="100000" />
          <span class="tip">-1 表示不限</span>
        </el-form-item>
        <el-form-item label="月费价格" prop="price">
          <el-input-number v-model="form.price" :min="0" :precision="2" :step="10" />
          <span class="tip">0 表示免费</span>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import {
  getPlanList,
  createPlan,
  updatePlan,
  deletePlan,
  type PlanItem
} from '@/api/modules/plan'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<PlanItem, { keyword: string; status?: number }>({
  fetchList: async (params) =>
    toPageResult(
      await getPlanList({
        page: params.page,
        page_size: params.page_size,
        keyword: params.keyword || undefined,
        status: params.status
      })
    ),
  initialQuery: { keyword: '', status: undefined }
})
const loadList = reload

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({
  name: '',
  code: '',
  user_limit: -1,
  storage_limit: -1, // 表单内以 GB 为单位存储，-1 表示不限
  api_rate_limit: -1,
  wiki_space_limit: -1,
  price: 0,
  status: 1,
  description: ''
})

const GB = 1024 * 1024 * 1024

// formatSize 将字节数格式化为可读大小
function formatSize(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let val = bytes
  let i = 0
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return `${val.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入套餐名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入套餐编码', trigger: 'blur' }],
  user_limit: [{ required: true, message: '请输入用户数上限', trigger: 'blur' }]
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.name = ''
  form.code = ''
  form.user_limit = -1
  form.storage_limit = -1
  form.api_rate_limit = -1
  form.wiki_space_limit = -1
  form.price = 0
  form.status = 1
  form.description = ''
  dialogVisible.value = true
}

function openEditDialog(row: PlanItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.name = row.name
  form.code = row.code
  form.user_limit = row.user_limit
  // 后端存储为字节，回填表单时换算为 GB（-1 保持不限）
  form.storage_limit = row.storage_limit === -1 ? -1 : Math.round(row.storage_limit / GB)
  form.api_rate_limit = row.api_rate_limit
  form.wiki_space_limit = row.wiki_space_limit
  form.price = row.price
  form.status = row.status
  form.description = row.description || ''
  dialogVisible.value = true
}

// buildPayload 将表单（GB 单位）转换为后端需要的字节配额载荷
function buildPayload() {
  return {
    name: form.name,
    description: form.description,
    user_limit: form.user_limit,
    storage_limit: form.storage_limit === -1 ? -1 : form.storage_limit * GB,
    api_rate_limit: form.api_rate_limit,
    wiki_space_limit: form.wiki_space_limit,
    price: form.price,
    status: form.status
  }
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      const payload = buildPayload()
      if (dialogMode.value === 'create') {
        await createPlan({ ...payload, code: form.code })
        ElMessage.success('新增成功')
      } else if (editingId.value) {
        await updatePlan(editingId.value, payload)
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function handleDelete(row: PlanItem) {
  ElMessageBox.confirm(`确定要删除套餐 "${row.name}" 吗？`, '提示', { type: 'warning' })
    .then(async () => {
      if (!row.id) return
      await deletePlan(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

function resetSearch() {
  reset()
}

async function handleToggleStatus(row: PlanItem) {
  if (!row.id) return
  const newStatus = row.status === 1 ? 0 : 1
  const actionText = newStatus === 1 ? '启用' : '停用'
  try {
    await ElMessageBox.confirm(`确定要${actionText}套餐 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await updatePlan(row.id, { status: newStatus })
    ElMessage.success(`${actionText}成功`)
    loadList()
  } catch (e) {
    // 用户取消或请求失败
  }
}
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 16px; }
.search-bar { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; }
.flex-1 { flex: 1; }
.pagination { display: flex; justify-content: flex-end; padding-top: 16px; }
.tip { margin-left: 8px; color: #999; font-size: 12px; }
</style>