<template>
  <div class="page">
    <el-card shadow="never">
      <div class="search-bar">
        <el-input v-model="search.keyword" placeholder="搜索角色名" clearable style="width: 200px" @keyup.enter="loadList" />
        <el-button type="primary" @click="loadList"><el-icon><Search /></el-icon>查询</el-button>
        <div class="flex-1"></div>
        <el-button type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>新增角色
        </el-button>
      </div>

      <el-table :data="list" border stripe v-loading="loading" style="width: 100%">
        <el-table-column prop="name" label="角色名" min-width="140" />
        <el-table-column prop="code" label="编码" min-width="140" />
        <el-table-column prop="description" label="描述" min-width="200" />
        <el-table-column label="系统角色" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.is_system" type="warning" size="small">系统</el-tag>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openBindPerm(row)">绑定权限</el-button>
            <el-button link type="primary" @click="openEditDialog(row)" v-if="!row.is_system">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)" v-if="!row.is_system">删除</el-button>
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
    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增角色' : '编辑角色'" width="480px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="角色名" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="编码" prop="code"><el-input v-model="form.code" :disabled="dialogMode === 'edit'" /></el-form-item>
        <el-form-item label="描述" prop="description"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 绑定权限弹窗 -->
    <el-dialog v-model="bindDialogVisible" title="绑定权限" width="560px" :close-on-click-modal="false">
      <div style="margin-bottom: 10px; font-size: 14px; color: #6b7280">
        当前角色：<b style="color: #111827">{{ currentRole?.name }}</b>
        ，已选中 <b style="color: #2563eb">{{ checkedPermIds.length }}</b> 个权限
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
        <template #default="{ node, data }">
          <span style="display: inline-flex; align-items: center; gap: 6px">
            <span>{{ data.name }}</span>
            <el-tag size="small" type="info" style="font-size: 11px">{{ data.code }}</el-tag>
          </span>
        </template>
      </el-tree>
      <template #footer>
        <el-button @click="bindDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitBind">保存绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import type { ElTree } from 'element-plus'
import {
  getRoleList,
  createRole,
  updateRole,
  deleteRole,
  bindRolePermissions,
  getRolePermissionIds,
  type RoleItem
} from '@/api/modules/role'
import { getPermissionList } from '@/api/modules/permission'

const list = ref<RoleItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const search = reactive({ keyword: '' })

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ name: '', code: '', description: '', status: 1 })

const rules: FormRules = {
  name: [{ required: true, message: '请输入角色名', trigger: 'blur' }],
  code: [{ required: true, message: '请输入编码', trigger: 'blur' }]
}

// 权限绑定相关
const bindDialogVisible = ref(false)
const currentRole = ref<RoleItem | null>(null)
const treeRef = ref<InstanceType<typeof ElTree>>()
const permTreeData = ref<any[]>([])
const allPermIds = ref<string[]>([])

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (search.keyword) params.keyword = search.keyword
    const res: any = await getRoleList(params)
    list.value = res.list || []
    total.value = res.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function loadAllPermissions() {
  const res: any = await getPermissionList({ page: 1, page_size: 500 })
  const perms = (res.list || []) as any[]
  allPermIds.value = perms.map((p) => p.id)
  // 按 resource 分组（简单实现：resource 作为父节点，权限作为子节点）
  const groups = new Map<string, { name: string; children: any[] }>()
  for (const p of perms) {
    if (!groups.has(p.resource)) groups.set(p.resource, { name: p.resource, children: [] })
    groups.get(p.resource)!.children.push(p)
  }
  permTreeData.value = Array.from(groups.values())
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.name = ''
  form.code = ''
  form.description = ''
  form.status = 1
  dialogVisible.value = true
}

function openEditDialog(row: RoleItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.name = row.name
  form.code = row.code
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
        await createRole({ name: form.name, code: form.code, description: form.description, status: form.status })
        ElMessage.success('新增成功')
      } else if (editingId.value) {
        await updateRole(editingId.value, { name: form.name, description: form.description, status: form.status })
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function handleDelete(row: RoleItem) {
  ElMessageBox.confirm(`确定要删除角色 "${row.name}" 吗？`, '提示', { type: 'warning' })
    .then(async () => {
      if (!row.id) return
      await deleteRole(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

const checkedPermIds = ref<string[]>([])

async function openBindPerm(row: RoleItem) {
  currentRole.value = row
  saving.value = true
  try {
    if (allPermIds.value.length === 0) await loadAllPermissions()
    const res: any = await getRolePermissionIds(row.id)
    // 兼容不同返回结构
    let ids: string[] = []
    if (res && Array.isArray(res.ids)) ids = res.ids
    else if (res && Array.isArray(res.permission_ids)) ids = res.permission_ids
    else if (res && res.data && Array.isArray(res.data)) ids = res.data.map((p: any) => p.id)
    else if (res && res.data && Array.isArray(res.data.ids)) ids = res.data.ids
    checkedPermIds.value = ids
    bindDialogVisible.value = true
  } catch (e) {
    ElMessage.error('加载权限数据失败')
  } finally {
    saving.value = false
  }
}

async function submitBind() {
  if (!currentRole.value) return
  // 从树中取出选中的节点 id（叶子节点 = 权限本身）
  const checked: any[] = treeRef.value?.getCheckedNodes(true) || []
  const leafIds = checked.filter((n) => !n.children || n.children.length === 0).map((n) => n.id)
  saving.value = true
  try {
    await bindRolePermissions(currentRole.value.id, leafIds)
    ElMessage.success('权限绑定成功')
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
.flex-1 { flex: 1; }
.pagination { display: flex; justify-content: flex-end; padding-top: 16px; }
</style>