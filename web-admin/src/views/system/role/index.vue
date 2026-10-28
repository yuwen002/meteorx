<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input v-model="query.keyword" :placeholder="t('rolePage.searchPlaceholder')" clearable style="width: 200px" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>{{ t('common.query') }}</el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="openRecycleBin">
          <el-icon><Delete /></el-icon>{{ t('rolePage.recycleBin') }}
        </el-button>
        <el-button type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>{{ t('rolePage.create') }}
        </el-button>
      </div>

      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-button type="danger" @click="handleBatchDelete">{{ t('rolePage.batchDelete') }}</el-button>
        <el-button type="warning" @click="handleBatchDisable">{{ t('rolePage.batchDisable') }}</el-button>
        <el-button type="success" @click="handleBatchEnable">{{ t('rolePage.batchEnable') }}</el-button>
        <span style="margin-left: 8px; color: #666;">{{ t('rolePage.selected', selectedIds.length) }}</span>
      </div>

      <el-table v-loading="loading" :data="list" border stripe style="width: 100%" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="55" :selectable="(row: RoleItem) => !row.is_system" />
        <el-table-column prop="name" :label="t('rolePage.col.name')" min-width="140" />
        <el-table-column prop="code" :label="t('rolePage.col.code')" min-width="140" />
        <el-table-column prop="description" :label="t('rolePage.col.description')" min-width="200" />
        <el-table-column :label="t('rolePage.col.systemRole')" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.is_system" type="warning" size="small">{{ t('rolePage.systemTag') }}</el-tag>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('rolePage.col.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? t('rolePage.status.enabled') : t('rolePage.status.disabled') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('rolePage.col.createdAt')" width="180" />
        <el-table-column :label="t('rolePage.col.operation')" width="380" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openViewPerms(row)">{{ t('rolePage.action.viewPerms') }}</el-button>
            <el-button link type="success" @click="openBindPerm(row)">{{ t('rolePage.action.bindPerms') }}</el-button>
            <el-button v-if="!row.is_system" link :type="row.status === 1 ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.status === 1 ? t('rolePage.status.off') : t('rolePage.status.enabled') }}
            </el-button>
            <el-button v-if="!row.is_system" link type="primary" @click="openEditDialog(row)">{{ t('rolePage.action.edit') }}</el-button>
            <el-button v-if="!row.is_system" link type="danger" @click="handleDelete(row)">{{ t('rolePage.action.delete') }}</el-button>
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

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? t('rolePage.dialog.createTitle') : t('rolePage.dialog.editTitle')" width="480px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item :label="t('rolePage.form.nameLabel')" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('rolePage.form.codeLabel')" prop="code"><el-input v-model="form.code" :disabled="dialogMode === 'edit'" /></el-form-item>
        <el-form-item :label="t('rolePage.form.scopeLabel')" prop="scope">
          <el-select v-model="form.scope" style="width: 100%" :disabled="dialogMode === 'edit'">
            <el-option :label="t('rolePage.scope.system')" value="system" />
            <el-option :label="t('rolePage.scope.tenant')" value="tenant" />
            <el-option :label="t('rolePage.scope.all')" value="all" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('rolePage.form.descLabel')" prop="description"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="t('rolePage.form.statusLabel')">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">{{ t('rolePage.status.enabled') }}</el-radio>
            <el-radio :value="0">{{ t('rolePage.status.disabled') }}</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 查看权限弹窗 -->
    <el-dialog v-model="viewPermsDialogVisible" :title="t('rolePage.view.title')" width="700px" :close-on-click-modal="false">
      <div style="margin-bottom: 10px; font-size: 14px; color: #6b7280">
        {{ t('rolePage.view.currentRole') }}<b style="color: #111827">{{ currentRole?.name }}</b>
        {{ t('rolePage.view.boundPrefix') }}<b style="color: #2563eb">{{ currentRolePerms.length }}</b>{{ t('rolePage.view.boundSuffix') }}
        <span v-if="selectedPermIds.length > 0" style="margin-left: 16px; color: #f56c6c;">{{ t('rolePage.view.selectedPerms', selectedPermIds.length) }}</span>
      </div>
      <div v-if="selectedPermIds.length > 0" style="margin-bottom: 10px;">
        <el-button type="danger" size="small" @click="handleBatchUnbindPerms">{{ t('rolePage.view.batchUnbind') }}</el-button>
      </div>
      <el-table
        :data="currentRolePerms"
        border
        stripe
        style="width: 100%"
        max-height="400"
        @selection-change="handlePermSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" :label="t('rolePage.view.colPermName')" min-width="140" />
        <el-table-column prop="code" :label="t('rolePage.view.colCode')" min-width="180">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" :label="t('rolePage.view.colResource')" width="100" />
        <el-table-column prop="action" :label="t('rolePage.view.colAction')" width="100" />
        <el-table-column :label="t('rolePage.view.colOperation')" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="handleUnbindPerm(row)">{{ t('rolePage.view.unbind') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="viewPermsDialogVisible = false">{{ t('rolePage.view.close') }}</el-button>
      </template>
    </el-dialog>

    <!-- 绑定权限弹窗 -->
    <el-dialog v-model="bindDialogVisible" :title="t('rolePage.bind.title')" width="560px" :close-on-click-modal="false">
      <div style="margin-bottom: 10px; font-size: 14px; color: #6b7280">
        {{ t('rolePage.bind.currentRole') }}<b style="color: #111827">{{ currentRole?.name }}</b>
        {{ t('rolePage.bind.checkedPrefix') }}<b style="color: #2563eb">{{ checkedPermIds.length }}</b>{{ t('rolePage.bind.checkedSuffix') }}
      </div>
      <el-tree
        ref="treeRef"
        :data="permTreeData"
        show-checkbox
        node-key="id"
        :default-checked-keys="checkedPermIds"
        :props="{ label: 'name', children: 'children' }"
        style="max-height: 400px; overflow: auto"
      >
        <template #default="{ data }">
          <span style="display: inline-flex; align-items: center; gap: 6px">
            <span>{{ data.name }}</span>
            <el-tag size="small" type="info" style="font-size: 11px">{{ data.code }}</el-tag>
          </span>
        </template>
      </el-tree>
      <template #footer>
        <el-button @click="bindDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitBind">{{ t('rolePage.bind.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { Search, Delete, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import type { ElTree } from 'element-plus'
import { useI18n } from '@/composables/useI18n'
import {
  getRoleList,
  createRole,
  updateRole,
  updateRoleStatus,
  deleteRole,
  batchDeleteRoles,
  batchUpdateRoleStatus,
  bindRolePermissions,
  getRolePermissions,
  unbindRolePermission,
  unbindRolePermissions,
  type RoleItem
} from '@/api/modules/role'
import type { PermissionItem } from '@/api/modules/permission'
import { getPermissionList } from '@/api/modules/permission'
import { useRouter } from 'vue-router'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const router = useRouter()
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
} = useTableList<RoleItem, { keyword: string }>({
  fetchList: async (params) =>
    toPageResult(
      await getRoleList({
        page: params.page,
        page_size: params.page_size,
        keyword: params.keyword || undefined
      })
    ),
  initialQuery: { keyword: '' },
  immediate: false
})
const loadList = reload
const selectedIds = ref<string[]>([])

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ name: '', code: '', description: '', scope: 'system', status: 1 })

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('rolePage.form.nameRequired'), trigger: 'blur' }],
  code: [{ required: true, message: t('rolePage.form.codeRequired'), trigger: 'blur' }]
}))

// 权限树节点（按 resource 分组，子节点为具体权限）
interface PermTreeNode {
  name: string
  children: PermissionItem[]
}

// 权限绑定相关
const bindDialogVisible = ref(false)
const currentRole = ref<RoleItem | null>(null)
const treeRef = ref<InstanceType<typeof ElTree>>()
const permTreeData = ref<PermTreeNode[]>([])
const allPermIds = ref<string[]>([])

// 查看权限相关
const viewPermsDialogVisible = ref(false)
const currentRolePerms = ref<PermissionItem[]>([])
const selectedPermIds = ref<string[]>([])

async function loadAllPermissions() {
  const res = await getPermissionList({ page: 1, page_size: 500 }) as unknown as { data?: PermissionItem[] }
  const perms = (res.data || []) as PermissionItem[]
  allPermissions.value = perms  // store all permissions
  allPermIds.value = perms.map((p) => p.id)
  // 按 resource 分组（简单实现：resource 作为父节点，权限作为子节点）
  const groups = new Map<string, PermTreeNode>()
  for (const p of perms) {
    if (!groups.has(p.resource)) groups.set(p.resource, { name: p.resource, children: [] })
    groups.get(p.resource)!.children.push(p)
  }
  permTreeData.value = Array.from(groups.values())
}

// 跳转到回收站页面
function openRecycleBin() {
  router.push('/system/role/recycle')
}

function resetSearch() {
  reset()
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.name = ''
  form.code = ''
  form.description = ''
  form.scope = 'system'
  form.status = 1
  dialogVisible.value = true
}

function openEditDialog(row: RoleItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.name = row.name
  form.code = row.code
  form.scope = row.scope || 'system'
  form.description = row.description || ''
  form.status = row.status ?? 1
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (dialogMode.value === 'create') {
        await createRole({ name: form.name, code: form.code, scope: form.scope, description: form.description, status: form.status })
        ElMessage.success(t('rolePage.created'))
      } else if (editingId.value) {
        // 编辑时也要传递 code，防止后端将其清空
        await updateRole(editingId.value, { name: form.name, code: form.code, description: form.description, status: form.status })
        ElMessage.success(t('rolePage.updated'))
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function handleSelectionChange(selection: RoleItem[]) {
  selectedIds.value = selection.map((row) => row.id).filter(Boolean) as string[]
}

function handleDelete(row: RoleItem) {
  ElMessageBox.confirm(t('rolePage.deleteConfirm', row.name), t('common.tip'), { type: 'warning' })
    .then(async () => {
      if (!row.id) return
      await deleteRole(row.id)
      ElMessage.success(t('rolePage.deleted'))
      loadList()
    })
    .catch(() => {})
}

function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(t('rolePage.batchDeleteConfirm', selectedIds.value.length), t('common.tip'), { type: 'warning' })
    .then(async () => {
      await batchDeleteRoles(selectedIds.value)
      ElMessage.success(t('rolePage.batchDeleted'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchEnable() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(t('rolePage.batchEnableConfirm', selectedIds.value.length), t('common.tip'), { type: 'warning' })
    .then(async () => {
      await batchUpdateRoleStatus(selectedIds.value, 1)
      ElMessage.success(t('rolePage.batchEnabled'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDisable() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(t('rolePage.batchDisableConfirm', selectedIds.value.length), t('common.tip'), { type: 'warning' })
    .then(async () => {
      await batchUpdateRoleStatus(selectedIds.value, 0)
      ElMessage.success(t('rolePage.batchDisabled'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

async function handleToggleStatus(row: RoleItem) {
  if (!row.id) return
  const newStatus = row.status === 1 ? 0 : 1
  try {
    await ElMessageBox.confirm(
      newStatus === 1 ? t('rolePage.enableConfirm', row.name) : t('rolePage.disableConfirm', row.name),
      t('common.tip'),
      { type: 'warning' }
    )
    await updateRoleStatus(row.id, newStatus)
    ElMessage.success(newStatus === 1 ? t('rolePage.enabled') : t('rolePage.disabled'))
    loadList()
  } catch (e) {
    // 用户取消或请求失败
  }
}

const checkedPermIds = ref<string[]>([])

// 查看角色已绑定的权限
async function openViewPerms(row: RoleItem) {
  currentRole.value = row
  saving.value = true
  try {
    // 直接获取角色已绑定的权限列表
    const perms = await getRolePermissions(row.id)
    currentRolePerms.value = perms || []
    viewPermsDialogVisible.value = true
  } catch (e) {
    ElMessage.error(t('rolePage.loadPermsFailed'))
  } finally {
    saving.value = false
  }
}

// 解绑权限
async function handleUnbindPerm(perm: PermissionItem) {
  if (!currentRole.value) return
  try {
    await ElMessageBox.confirm(
      t('rolePage.unbindConfirm', currentRole.value.name, perm.name),
      t('rolePage.unbindTitle'),
      { type: 'warning' }
    )
    await unbindRolePermission(currentRole.value.id, perm.id)
    ElMessage.success(t('rolePage.unbindSuccess'))
    // 刷新列表
    currentRolePerms.value = currentRolePerms.value.filter(p => p.id !== perm.id)
  } catch (e) {
    // 用户取消或失败
  }
}

// 权限选择变化
function handlePermSelectionChange(selection: PermissionItem[]) {
  selectedPermIds.value = selection.map(p => p.id)
}

// 批量解绑权限
async function handleBatchUnbindPerms() {
  if (!currentRole.value || selectedPermIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(
      t('rolePage.batchUnbindConfirm', currentRole.value.name, selectedPermIds.value.length),
      t('rolePage.batchUnbindTitle'),
      { type: 'warning' }
    )
    await unbindRolePermissions(currentRole.value.id, selectedPermIds.value)
    ElMessage.success(t('rolePage.batchUnbindSuccess'))
    // 刷新列表
    currentRolePerms.value = currentRolePerms.value.filter(p => !selectedPermIds.value.includes(p.id))
    selectedPermIds.value = []
  } catch (e) {
    // 用户取消或失败
  }
}

// 存储所有权限列表
const allPermissions = ref<PermissionItem[]>([])

async function openBindPerm(row: RoleItem) {
  currentRole.value = row
  saving.value = true
  try {
    if (allPermIds.value.length === 0) await loadAllPermissions()
    // 获取角色已绑定的权限列表，提取ID
    const perms = await getRolePermissions(row.id)
    checkedPermIds.value = (perms || []).map((p: PermissionItem) => p.id)
    bindDialogVisible.value = true
  } catch (e) {
    ElMessage.error(t('rolePage.loadPermsFailed'))
  } finally {
    saving.value = false
  }
}

async function submitBind() {
  if (!currentRole.value) return
  // 从树中取出选中的节点 id（叶子节点 = 权限本身）
  const checked = (treeRef.value?.getCheckedNodes(true) || []) as Array<{ id: string; children?: unknown[] }>
  const leafIds = checked.filter((n) => !n.children || n.children.length === 0).map((n) => n.id)
  saving.value = true
  try {
    await bindRolePermissions(currentRole.value.id, leafIds)
    ElMessage.success(t('rolePage.bindSuccess'))
    bindDialogVisible.value = false
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadList(), loadAllPermissions().catch(() => {})])
})
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 16px; }
.search-bar { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; }
.batch-bar { display: flex; align-items: center; margin-bottom: 12px; padding: 8px 12px; background: #fef0f0; border-radius: 4px; }
.flex-1 { flex: 1; }
.pagination { display: flex; justify-content: flex-end; padding-top: 16px; }
</style>