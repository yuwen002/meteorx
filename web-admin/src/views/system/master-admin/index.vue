<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <div class="search-left">
          <el-input
            v-model="query.keyword"
            placeholder="搜索用户名/昵称"
            clearable
            style="width: 220px"
            @keyup.enter="loadList"
          />
          <el-select
            v-model="query.status"
            placeholder="状态"
            clearable
            style="width: 120px"
          >
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
          <el-button type="primary" @click="loadList">
            <el-icon><Search /></el-icon>查询
          </el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
        <div class="search-right">
          <el-button type="info" @click="goToRecycle">
            <el-icon><DeleteFilled /></el-icon>回收站
          </el-button>
          <el-button type="success" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>新增系统管理员
          </el-button>
        </div>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-alert
          :title="`已选择 ${selectedIds.length} 项`"
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <el-button link type="primary" @click="handleBatchEnable">批量启用</el-button>
            <el-button link type="warning" @click="handleBatchDisable">批量禁用</el-button>
            <el-button link type="danger" @click="handleBatchDelete">批量删除</el-button>
          </template>
        </el-alert>
      </div>

      <!-- 列表 -->
      <el-table
        v-loading="loading"
        :data="list"
        row-key="id"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column label="用户名" prop="username" min-width="120" />
        <el-table-column label="昵称" prop="nickname" min-width="120" />
        <el-table-column label="邮箱" prop="email" min-width="180" />
        <el-table-column label="角色" min-width="200">
          <template #default="{ row }">
            <div v-if="row.role_ids && row.role_ids.length > 0">，
              <el-tag
                v-for="(roleId, index) in row.role_ids"
                :key="roleId"
                size="small"
                class="role-tag"
                closable
                @close="handleUnbindRole(row, roleId, row.roles[index])"
              >
                {{ row.roles[index] || roleId }}
              </el-tag>
            </div>
            <el-tag v-else type="info" size="small">未分配角色</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="created_at" min-width="160" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="warning" @click="toggleStatus(row)">
              {{ row.status === 1 ? '禁用' : '启用' }}
            </el-button>
            <el-button
              v-if="!isProtectedUser(row)"
              link
              type="danger"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
            <el-tag v-else type="info" size="small">系统保护</el-tag>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        class="pagination"
        @size-change="loadList"
        @current-change="loadList"
      />
    </el-card>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增系统管理员' : '编辑系统管理员'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="form.username"
            placeholder="请输入用户名"
            :disabled="dialogMode === 'edit'"
          />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>
        <el-form-item v-else label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="不修改请留空"
            show-password
          />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="角色" prop="role_id">
          <el-select v-model="form.role_id" placeholder="请选择角色" style="width: 100%">
            <el-option
              v-for="role in roleOptions"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus, DeleteFilled } from '@element-plus/icons-vue'
import {
  getMasterAdminList,
  createMasterAdmin,
  updateMasterAdmin,
  deleteMasterAdmin,
  updateMasterAdminStatus,
  batchUpdateMasterAdminStatus,
  batchDeleteMasterAdmins,
  unbindUserRole,
  type UserItem,
  type MasterAdminCreateParams,
  type MasterAdminUpdateParams
} from '@/api/modules/user'
import { getSystemAdminRoles, type RoleOption } from '@/api/modules/role'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'

const router = useRouter()

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<UserItem, { keyword: string; status?: number }>({
  fetchList: async (params) => {
    const req: any = { page: params.page, page_size: params.page_size }
    if (params.keyword) req.keyword = params.keyword
    if (params.status !== undefined && params.status !== null) req.status = params.status
    const res = await getMasterAdminList(req)
    return toPageResult(res)
  },
  initialQuery: { keyword: '', status: undefined },
  immediate: false
})
const loadList = reload
const selectedIds = ref<string[]>([])

// 系统保护用户ID列表（初始管理员，不允许删除）
const protectedUserIDs = ['admin-id-000001']

// 检查用户是否为系统保护用户
function isProtectedUser(row: UserItem): boolean {
  return protectedUserIDs.includes(row.id)
}

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)
const formRef = ref<FormInstance>()
const roleOptions = ref<RoleOption[]>([])

const form = reactive({
  username: '',
  password: '',
  nickname: '',
  email: '',
  role_id: ''
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '长度在 3 到 50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur', validator: (rule, value, callback) => {
      if (dialogMode.value === 'create' && !value) {
        callback(new Error('请输入密码'))
      } else {
        callback()
      }
    }}
  ],
  nickname: [{ max: 50, message: '最多 50 个字符', trigger: 'blur' }],
  email: [{ type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }],
  role_id: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

function resetSearch() {
  reset()
}

function handleSelectionChange(selection: UserItem[]) {
  selectedIds.value = selection.map(item => item.id)
}

async function loadRoles() {
  try {
    const res: any = await getSystemAdminRoles()
    roleOptions.value = res || []
  } catch (e) {
    roleOptions.value = []
  }
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.username = ''
  form.password = ''
  form.nickname = ''
  form.email = ''
  form.role_id = ''
  dialogVisible.value = true
}

function openEditDialog(row: UserItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.username = row.username
  form.password = ''
  form.nickname = row.nickname || ''
  form.email = row.email || ''
  form.role_id = row.role_ids && row.role_ids.length > 0 ? row.role_ids[0] : ''
  dialogVisible.value = true
}

function handleSubmit() {
  formRef.value?.validate(async (valid) => {
    if (!valid) return
    try {
      if (dialogMode.value === 'create') {
        const data: MasterAdminCreateParams = {
          username: form.username,
          password: form.password,
          nickname: form.nickname,
          email: form.email,
          role_id: form.role_id
        }
        await createMasterAdmin(data)
        ElMessage.success('创建成功')
      } else {
        const data: MasterAdminUpdateParams = {
          nickname: form.nickname,
          email: form.email,
          role_id: form.role_id
        }
        if (form.password) {
          data.password = form.password
        }
        await updateMasterAdmin(editingId.value!, data)
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadList()
    } catch (e: any) {
      ElMessage.error(e?.response?.data?.message || '操作失败')
    }
  })
}

function toggleStatus(row: UserItem) {
  const newStatus = row.status === 1 ? 0 : 1
  const action = newStatus === 1 ? '启用' : '禁用'
  ElMessageBox.confirm(`确定要${action}系统管理员 "${row.username}" 吗？`, '确认', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      await updateMasterAdminStatus(row.id, newStatus)
      ElMessage.success(`${action}成功`)
      loadList()
    })
    .catch(() => {})
}

function handleDelete(row: UserItem) {
  ElMessageBox.confirm(`确定要删除系统管理员 "${row.username}" 吗？`, '确认删除', {
    type: 'error',
    confirmButtonText: '确定删除',
    cancelButtonText: '取消'
  })
    .then(async () => {
      await deleteMasterAdmin(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

// 解绑角色
function handleUnbindRole(row: UserItem, roleId: string, roleName: string) {
  ElMessageBox.confirm(
    `确定要解除用户 "${row.username}" 的角色 "${roleName || roleId}" 吗？`,
    '确认解绑角色',
    {
      type: 'warning',
      confirmButtonText: '确定解绑',
      cancelButtonText: '取消'
    }
  )
    .then(async () => {
      await unbindUserRole(row.id, roleId)
      ElMessage.success('角色解绑成功')
      loadList()
    })
    .catch(() => {})
}

function handleBatchEnable() {
  ElMessageBox.confirm(`确定要启用选中的 ${selectedIds.value.length} 个系统管理员吗？`, '批量启用', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      await batchUpdateMasterAdminStatus(selectedIds.value, 1)
      ElMessage.success('批量启用成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDisable() {
  ElMessageBox.confirm(`确定要禁用选中的 ${selectedIds.value.length} 个系统管理员吗？`, '批量禁用', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      await batchUpdateMasterAdminStatus(selectedIds.value, 0)
      ElMessage.success('批量禁用成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDelete() {
  // 过滤掉保护用户
  const deletableIds = selectedIds.value.filter(id => !protectedUserIDs.includes(id))
  const protectedCount = selectedIds.value.length - deletableIds.length

  if (deletableIds.length === 0) {
    ElMessage.warning('选中的用户中包含系统保护用户，无法删除')
    return
  }

  let confirmMessage = `确定要删除选中的 ${deletableIds.length} 个系统管理员吗？此操作不可恢复`
  if (protectedCount > 0) {
    confirmMessage += `（已自动跳过 ${protectedCount} 个系统保护用户）`
  }

  ElMessageBox.confirm(confirmMessage, '危险操作', {
    type: 'error',
    confirmButtonText: '确定删除',
    cancelButtonText: '取消'
  })
    .then(async () => {
      await batchDeleteMasterAdmins(deletableIds)
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

// 跳转到回收站
function goToRecycle() {
  router.push('/system/master-admin/recycle')
}

onMounted(() => {
  loadList()
  loadRoles()
})
</script>

<style scoped lang="scss">
.page {
  padding: 20px;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;

  .search-left {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .search-right {
    display: flex;
    gap: 12px;
    align-items: center;
  }
}

.batch-bar {
  margin-bottom: 16px;
}

.role-tag {
  margin-right: 4px;
}

.pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>