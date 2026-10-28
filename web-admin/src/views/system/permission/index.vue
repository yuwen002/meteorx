<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input v-model="query.keyword" :placeholder="t('permPage.searchNameCodePlaceholder')" clearable style="width: 240px" @keyup.enter="loadList" />
        <el-input v-model="query.resource" :placeholder="t('permPage.filterResourcePlaceholder')" clearable style="width: 160px" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>{{ t('common.query') }}</el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog"><el-icon><Plus /></el-icon>{{ t('permPage.create') }}</el-button>
      </div>

      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-button type="danger" @click="handleBatchDelete">{{ t('permPage.batchDelete') }}</el-button>
        <el-button type="warning" @click="handleBatchDisable">{{ t('permPage.batchDisable') }}</el-button>
        <el-button type="success" @click="handleBatchEnable">{{ t('permPage.batchEnable') }}</el-button>
        <span style="margin-left: 8px; color: #666;">{{ t('permPage.selected', selectedIds.length) }}</span>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" :label="t('permPage.col.name')" min-width="140" />
        <el-table-column prop="code" :label="t('permPage.col.code')" min-width="180">
          <template #default="{ row }">
            <el-tag type="primary" effect="plain" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" :label="t('permPage.col.resource')" width="120" />
        <el-table-column prop="action" :label="t('permPage.col.action')" width="100" />
        <el-table-column prop="description" :label="t('permPage.col.description')" min-width="180" />
        <el-table-column :label="t('permPage.col.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? t('permPage.status.enabled') : t('permPage.status.disabled') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('permPage.col.createdAt')" width="180" />
        <el-table-column :label="t('permPage.col.operation')" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">{{ t('permPage.edit') }}</el-button>
            <el-button link :type="row.status === 1 ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.status === 1 ? t('permPage.status.disabled') : t('permPage.status.enabled') }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ t('permPage.delete') }}</el-button>
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

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? t('permPage.dialog.createTitle') : t('permPage.dialog.editTitle')" width="500px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item :label="t('permPage.form.nameLabel')" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('permPage.form.codeLabel')" prop="code"><el-input v-model="form.code" :disabled="dialogMode === 'edit'" :placeholder="t('permPage.form.codePlaceholder')" /></el-form-item>
        <el-form-item :label="t('permPage.form.resourceLabel')" prop="resource"><el-input v-model="form.resource" :placeholder="t('permPage.form.resourcePlaceholder')" /></el-form-item>
        <el-form-item :label="t('permPage.form.actionLabel')" prop="action"><el-input v-model="form.action" :placeholder="t('permPage.form.actionPlaceholder')" /></el-form-item>
        <el-form-item :label="t('permPage.form.descLabel')" prop="description"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
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
import {
  getPermissionList,
  createPermission,
  updatePermission,
  deletePermission,
  updatePermissionStatus,
  batchUpdatePermissionStatus,
  batchDeletePermissions,
  type PermissionItem
} from '@/api/modules/permission'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

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
} = useTableList<PermissionItem, { keyword: string; resource: string }>({
  fetchList: async (params) =>
    toPageResult(
      await getPermissionList({
        page: params.page,
        page_size: params.page_size,
        keyword: params.keyword || undefined,
        resource: params.resource || undefined
      })
    ),
  initialQuery: { keyword: '', resource: '' }
})
const loadList = reload
const selectedIds = ref<string[]>([])

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ name: '', code: '', resource: '', action: '', description: '' })

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('permPage.form.nameRequired'), trigger: 'blur' }],
  code: [{ required: true, message: t('permPage.form.codeRequired'), trigger: 'blur' }],
  resource: [{ required: true, message: t('permPage.form.resourceRequired'), trigger: 'blur' }],
  action: [{ required: true, message: t('permPage.form.actionRequired'), trigger: 'blur' }]
}))

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.name = ''
  form.code = ''
  form.resource = ''
  form.action = ''
  form.description = ''
  dialogVisible.value = true
}

function openEditDialog(row: PermissionItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.name = row.name
  form.code = row.code
  form.resource = row.resource
  form.action = row.action
  form.description = row.description || ''
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (dialogMode.value === 'create') {
        await createPermission({ ...form })
        ElMessage.success(t('permPage.created'))
      } else if (editingId.value) {
        await updatePermission(editingId.value, { ...form })
        ElMessage.success(t('permPage.updated'))
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function handleDelete(row: PermissionItem) {
  ElMessageBox.confirm(t('permPage.deleteConfirm', row.name), t('common.tip'), { type: 'warning' })
    .then(async () => {
      if (!row.id) return
      await deletePermission(row.id)
      ElMessage.success(t('permPage.deleted'))
      loadList()
    })
    .catch(() => {})
}

function handleSelectionChange(selection: PermissionItem[]) {
  selectedIds.value = selection.map(item => item.id).filter((id): id is string => !!id)
}

function resetSearch() {
  reset()
}

async function handleToggleStatus(row: PermissionItem) {
  if (!row.id) return
  const newStatus = row.status === 1 ? 0 : 1
  try {
    await ElMessageBox.confirm(
      newStatus === 1 ? t('permPage.enableConfirm', row.name) : t('permPage.disableConfirm', row.name),
      t('common.tip'),
      { type: 'warning' }
    )
    await updatePermissionStatus(row.id, newStatus)
    ElMessage.success(newStatus === 1 ? t('permPage.enabled') : t('permPage.disabled'))
    loadList()
  } catch (e) {
    // 用户取消或请求失败
  }
}

function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(t('permPage.batchDeleteConfirm', selectedIds.value.length), t('common.tip'), { type: 'warning' })
    .then(async () => {
      await batchDeletePermissions(selectedIds.value)
      ElMessage.success(t('permPage.batchDeleted'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchEnable() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(t('permPage.batchEnableConfirm', selectedIds.value.length), t('common.tip'), { type: 'warning' })
    .then(async () => {
      await batchUpdatePermissionStatus(selectedIds.value, 1)
      ElMessage.success(t('permPage.batchEnabled'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDisable() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(t('permPage.batchDisableConfirm', selectedIds.value.length), t('common.tip'), { type: 'warning' })
    .then(async () => {
      await batchUpdatePermissionStatus(selectedIds.value, 0)
      ElMessage.success(t('permPage.batchDisabled'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 16px; }
.search-bar { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; }
.batch-bar { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; padding: 12px; background-color: #f5f7fa; border-radius: 4px; }
.flex-1 { flex: 1; }
.pagination { display: flex; justify-content: flex-end; padding-top: 16px; }
</style>