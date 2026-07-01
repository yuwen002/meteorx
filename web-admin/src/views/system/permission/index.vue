<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input v-model="search.keyword" placeholder="搜索权限名/编码" clearable style="width: 240px" @keyup.enter="loadList" />
        <el-input v-model="search.resource" placeholder="按资源过滤" clearable style="width: 160px" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog"><el-icon><Plus /></el-icon>新增权限</el-button>
      </div>

      <div class="batch-bar" v-if="selectedIds.length > 0">
        <el-button type="danger" @click="handleBatchDelete">批量删除</el-button>
        <el-button type="warning" @click="handleBatchDisable">批量禁用</el-button>
        <el-button type="success" @click="handleBatchEnable">批量启用</el-button>
        <span style="margin-left: 8px; color: #666;">已选择 {{ selectedIds.length }} 项</span>
      </div>

      <el-table :data="list" border stripe v-loading="loading" style="width: 100%" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" label="权限名" min-width="140" />
        <el-table-column prop="code" label="编码" min-width="180">
          <template #default="{ row }">
            <el-tag type="primary" effect="plain" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源" width="120" />
        <el-table-column prop="action" label="操作" width="100" />
        <el-table-column prop="description" label="描述" min-width="180" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link :type="row.status === 1 ? 'warning' : 'success'" @click="handleToggleStatus(row)">
              {{ row.status === 1 ? '禁用' : '启用' }}
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

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增权限' : '编辑权限'" width="500px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="权限名" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="编码" prop="code"><el-input v-model="form.code" :disabled="dialogMode === 'edit'" placeholder="如: user:list" /></el-form-item>
        <el-form-item label="资源" prop="resource"><el-input v-model="form.resource" placeholder="如: user" /></el-form-item>
        <el-form-item label="操作" prop="action"><el-input v-model="form.action" placeholder="如: list/create/read/update/delete" /></el-form-item>
        <el-form-item label="描述" prop="description"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
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

const list = ref<PermissionItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const search = reactive({ keyword: '', resource: '' })
const selectedIds = ref<string[]>([])

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ name: '', code: '', resource: '', action: '', description: '' })

const rules: FormRules = {
  name: [{ required: true, message: '请输入权限名', trigger: 'blur' }],
  code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
  resource: [{ required: true, message: '请输入资源', trigger: 'blur' }],
  action: [{ required: true, message: '请输入操作', trigger: 'blur' }]
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (search.keyword) params.keyword = search.keyword
    if (search.resource) params.resource = search.resource
    const res: any = await getPermissionList(params)
    list.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

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
        ElMessage.success('新增成功')
      } else if (editingId.value) {
        await updatePermission(editingId.value, { ...form })
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function handleDelete(row: PermissionItem) {
  ElMessageBox.confirm(`确定要删除权限 "${row.name}" 吗？`, '提示', { type: 'warning' })
    .then(async () => {
      if (!row.id) return
      await deletePermission(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

function handleSelectionChange(selection: PermissionItem[]) {
  selectedIds.value = selection.map(item => item.id).filter((id): id is string => !!id)
}

function resetSearch() {
  search.keyword = ''
  search.resource = ''
  page.value = 1
  loadList()
}

async function handleToggleStatus(row: PermissionItem) {
  if (!row.id) return
  const newStatus = row.status === 1 ? 0 : 1
  const actionText = newStatus === 1 ? '启用' : '禁用'
  try {
    await ElMessageBox.confirm(`确定要${actionText}权限 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await updatePermissionStatus(row.id, newStatus)
    ElMessage.success(`${actionText}成功`)
    loadList()
  } catch (e) {
    // 用户取消或请求失败
  }
}

function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个权限吗？`, '提示', { type: 'warning' })
    .then(async () => {
      await batchDeletePermissions(selectedIds.value)
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchEnable() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(`确定要启用选中的 ${selectedIds.value.length} 个权限吗？`, '提示', { type: 'warning' })
    .then(async () => {
      await batchUpdatePermissionStatus(selectedIds.value, 1)
      ElMessage.success('批量启用成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDisable() {
  if (selectedIds.value.length === 0) return
  ElMessageBox.confirm(`确定要禁用选中的 ${selectedIds.value.length} 个权限吗？`, '提示', { type: 'warning' })
    .then(async () => {
      await batchUpdatePermissionStatus(selectedIds.value, 0)
      ElMessage.success('批量禁用成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

onMounted(loadList)
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 16px; }
.search-bar { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; }
.batch-bar { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; padding: 12px; background-color: #f5f7fa; border-radius: 4px; }
.flex-1 { flex: 1; }
.pagination { display: flex; justify-content: flex-end; padding-top: 16px; }
</style>