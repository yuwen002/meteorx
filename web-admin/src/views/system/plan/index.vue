<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input v-model="query.keyword" :placeholder="t('planPage.searchPlaceholder')" clearable style="width: 240px" @keyup.enter="loadList" />
        <el-select v-model="query.status" :placeholder="t('planPage.statusFilter')" clearable style="width: 130px">
          <el-option :label="t('planPage.status.enabled')" :value="1" />
          <el-option :label="t('planPage.status.disabled')" :value="0" />
        </el-select>
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>{{ t('common.query') }}</el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog"><el-icon><Plus /></el-icon>{{ t('planPage.create') }}</el-button>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column prop="name" :label="t('planPage.col.name')" min-width="130" />
        <el-table-column prop="code" :label="t('planPage.col.code')" min-width="140">
          <template #default="{ row }">
            <el-tag type="primary" effect="plain" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('planPage.col.userLimit')" width="120">
          <template #default="{ row }">
            <span v-if="row.user_limit === -1">{{ t('planPage.unlimited') }}</span>
            <span v-else>{{ t('planPage.people', row.user_limit) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('planPage.col.storage')" width="120">
          <template #default="{ row }">
            <span v-if="row.storage_limit === -1">{{ t('planPage.unlimited') }}</span>
            <span v-else>{{ formatSize(row.storage_limit) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('planPage.col.apiRate')" width="120">
          <template #default="{ row }">
            <span v-if="row.api_rate_limit === -1">{{ t('planPage.unlimited') }}</span>
            <span v-else>{{ t('planPage.ratePerMin', row.api_rate_limit) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('planPage.col.wikiSpace')" width="110">
          <template #default="{ row }">
            <span v-if="row.wiki_space_limit === -1">{{ t('planPage.unlimited') }}</span>
            <span v-else>{{ t('planPage.spaces', row.wiki_space_limit) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('planPage.col.price')" width="110">
          <template #default="{ row }">
            <span v-if="row.price === 0">
              <el-tag type="success" size="small" effect="plain">{{ t('planPage.free') }}</el-tag>
            </span>
            <span v-else>¥{{ row.price }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('planPage.col.subscribers')" width="110">
          <template #default="{ row }">
            {{ row.subscriber_cnt ?? 0 }}
          </template>
        </el-table-column>
        <el-table-column prop="description" :label="t('planPage.col.description')" min-width="180" />
        <el-table-column :label="t('planPage.col.status')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? t('planPage.status.enabled') : t('planPage.status.disabled') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('planPage.col.createdAt')" width="170" />
        <el-table-column :label="t('planPage.col.operation')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">{{ t('planPage.action.edit') }}</el-button>
            <el-button link :type="row.status === 1 ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.status === 1 ? t('planPage.status.disabled') : t('planPage.status.enabled') }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ t('planPage.action.delete') }}</el-button>
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

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? t('planPage.dialog.createTitle') : t('planPage.dialog.editTitle')" width="560px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('planPage.form.nameLabel')" prop="name"><el-input v-model="form.name" :placeholder="t('planPage.form.namePlaceholder')" /></el-form-item>
        <el-form-item :label="t('planPage.form.codeLabel')" prop="code">
          <el-input v-model="form.code" :disabled="dialogMode === 'edit'" :placeholder="t('planPage.form.codePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('planPage.form.userLimitLabel')" prop="user_limit">
          <el-input-number v-model="form.user_limit" :min="-1" :max="999999" />
          <span class="tip">{{ t('planPage.form.tipUnlimited') }}</span>
        </el-form-item>
        <el-form-item :label="t('planPage.form.storageLabel')" prop="storage_limit">
          <el-input-number v-model="form.storage_limit" :min="-1" :max="102400" :precision="0" />
          <span class="tip">{{ t('planPage.form.tipStorageUnlimited') }}</span>
        </el-form-item>
        <el-form-item :label="t('planPage.form.apiLabel')" prop="api_rate_limit">
          <el-input-number v-model="form.api_rate_limit" :min="-1" :max="1000000" :step="60" />
          <span class="tip">{{ t('planPage.form.tipApi') }}</span>
        </el-form-item>
        <el-form-item :label="t('planPage.form.wikiLabel')" prop="wiki_space_limit">
          <el-input-number v-model="form.wiki_space_limit" :min="-1" :max="100000" />
          <span class="tip">{{ t('planPage.form.tipUnlimited') }}</span>
        </el-form-item>
        <el-form-item :label="t('planPage.form.priceLabel')" prop="price">
          <el-input-number v-model="form.price" :min="0" :precision="2" :step="10" />
          <span class="tip">{{ t('planPage.form.tipPriceFree') }}</span>
        </el-form-item>
        <el-form-item :label="t('planPage.form.statusLabel')" prop="status">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" :active-text="t('planPage.status.enabled')" :inactive-text="t('planPage.status.disabled')" />
        </el-form-item>
        <el-form-item :label="t('planPage.form.descLabel')" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from '@/composables/useI18n'
import {
  getPlanList,
  createPlan,
  updatePlan,
  deletePlan,
  type PlanItem
} from '@/api/modules/plan'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const { t } = useI18n()

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
  storage_limit: -1, // storage in GB within the form, -1 means unlimited
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

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('planPage.form.nameRequired'), trigger: 'blur' }],
  code: [{ required: true, message: t('planPage.form.codeRequired'), trigger: 'blur' }],
  user_limit: [{ required: true, message: t('planPage.form.userLimitRequired'), trigger: 'blur' }]
}))

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
        ElMessage.success(t('planPage.created'))
      } else if (editingId.value) {
        await updatePlan(editingId.value, payload)
        ElMessage.success(t('planPage.updated'))
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function handleDelete(row: PlanItem) {
  ElMessageBox.confirm(t('planPage.deleteConfirm', row.name), t('common.tip'), { type: 'warning' })
    .then(async () => {
      if (!row.id) return
      await deletePlan(row.id)
      ElMessage.success(t('planPage.deleted'))
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
  try {
    await ElMessageBox.confirm(
      newStatus === 1 ? t('planPage.enableConfirm', row.name) : t('planPage.disableConfirm', row.name),
      t('common.tip'),
      { type: 'warning' }
    )
    await updatePlan(row.id, { status: newStatus })
    ElMessage.success(newStatus === 1 ? t('planPage.enabled') : t('planPage.disabled'))
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