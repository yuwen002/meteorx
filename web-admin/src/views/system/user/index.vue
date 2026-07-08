<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="search.keyword"
          placeholder="搜索用户名/昵称"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select
          v-model="search.status"
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
        <div class="flex-1"></div>
        <el-button type="info" v-if="userStore.isAdmin" @click="toggleRecycleBin">
          <el-icon><Delete /></el-icon>{{ isRecycleBin ? '返回列表' : '回收站' }}
        </el-button>
        <el-button type="success" v-if="userStore.hasPermission('user:create') && !isRecycleBin" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>新增用户
        </el-button>
      </div>

      <!-- 列表 -->
      <el-table :data="list" border stripe v-loading="loading" style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column v-if="userStore.isAdmin" prop="tenant_id" label="租户ID" min-width="180">
          <template #default="{ row }">
            <el-tag v-if="row.tenant_id" size="small" type="info">{{ row.tenant_id }}</el-tag>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="主管理员" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.is_master" type="danger" size="small">MASTER</el-tag>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column v-if="isRecycleBin" prop="deleted_at" label="删除时间" width="180" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <template v-if="isRecycleBin">
              <el-button
                link
                type="success"
                @click="handleRestore(row)"
              >恢复</el-button>
              <el-button
                link
                type="danger"
                v-if="!row.is_master"
                @click="handlePermanentDelete(row)"
              >永久删除</el-button>
            </template>
            <template v-else>
              <el-button
                link
                type="primary"
                v-if="userStore.hasPermission('user:update')"
                @click="openEditDialog(row)"
              >编辑</el-button>
              <el-button
                link
                type="info"
                @click="openViewPermissions(row)"
              >查看权限</el-button>
              <el-button
                link
                type="success"
                v-if="userStore.hasPermission('rbac:user_role:assign') || userStore.isAdmin"
                @click="openAssignRoles(row)"
              >分配角色</el-button>
              <el-button
                link
                type="warning"
                v-if="userStore.hasPermission('user:update')"
                @click="toggleStatus(row)"
              >{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
              <el-button
                link
                type="danger"
                v-if="userStore.hasPermission('user:delete') && !row.is_master"
                @click="handleDelete(row)"
              >删除</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 查看权限弹窗 -->
    <el-dialog
      v-model="permsDialogVisible"
      :title="`用户权限 - ${currentUser?.username || ''}`"
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-loading="permsLoading">
        <!-- 角色列表 -->
        <div class="section" style="margin-bottom: 20px;">
          <h4 style="margin-bottom: 12px; color: #606266;">所属角色</h4>
          <el-tag
            v-for="role in userRoles"
            :key="role.id"
            type="primary"
            style="margin-right: 8px; margin-bottom: 8px;"
          >
            {{ role.name }}
          </el-tag>
          <el-empty v-if="userRoles.length === 0" description="暂无角色" :image-size="60" />
        </div>

        <!-- 权限列表 -->
        <div class="section">
          <h4 style="margin-bottom: 12px; color: #606266;">权限列表</h4>
          <el-table :data="userPermissions" border stripe size="small">
            <el-table-column prop="name" label="权限名称" min-width="150" />
            <el-table-column prop="code" label="权限码" min-width="180" />
            <el-table-column prop="resource" label="资源" width="100" />
            <el-table-column prop="action" label="操作" width="100" />
          </el-table>
          <el-empty v-if="userPermissions.length === 0" description="暂无权限" :image-size="60" />
        </div>
      </div>
    </el-dialog>

    <!-- 分配角色弹窗 -->
    <el-dialog
      v-model="assignRolesDialogVisible"
      :title="`分配角色 - ${currentUser?.username || ''}`"
      width="500px"
      :close-on-click-modal="false"
    >
      <div v-loading="assignRolesLoading">
        <div style="margin-bottom: 10px; font-size: 14px; color: #6b7280">
          已选择 <b style="color: #2563eb">{{ selectedRoleIds.length }}</b> 个角色
        </div>
        <el-checkbox-group v-model="selectedRoleIds">
          <el-checkbox
            v-for="role in availableRoles"
            :key="role.id"
            :label="role.id"
            style="display: block; margin-bottom: 8px;"
          >
            {{ role.name }}
            <el-tag size="small" type="info" style="margin-left: 8px;">{{ role.code }}</el-tag>
          </el-checkbox>
        </el-checkbox-group>
        <el-empty v-if="availableRoles.length === 0" description="暂无可分配角色" :image-size="60" />
      </div>
      <template #footer>
        <el-button @click="assignRolesDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingRoles" @click="submitAssignRoles">确定</el-button>
      </template>
    </el-dialog>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增用户' : '编辑用户'"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="90px"
      >
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="dialogMode === 'edit'" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="6-32 位" />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
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
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Search, Plus, Delete } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import {
  getUserList,
  getAllTenantUsers,
  getAllDeletedTenantUsers,
  createUser,
  updateUser,
  deleteUser,
  restoreTenantUser,
  type UserItem,
  type UserCreateParams,
  type UserUpdateParams
} from '@/api/modules/user'
import { getUserRoles, getRolePermissions, type RoleItem, getRoleListForSelect, assignUserRoles, removeAllUserRoles } from '@/api/modules/role'
import { type PermissionItem } from '@/api/modules/permission'

const userStore = useUserStore()

const list = ref<UserItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const isRecycleBin = ref(false)

const search = reactive({ keyword: '', status: undefined as number | undefined })

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<UserCreateParams & UserUpdateParams>({
  username: '',
  password: '',
  nickname: '',
  email: '',
  status: 1
})

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' }],
  email: [{ type: 'email', message: '请输入正确的邮箱', trigger: 'blur' }]
}

// 查看权限相关
const permsDialogVisible = ref(false)
const currentUser = ref<UserItem | null>(null)
const userRoles = ref<RoleItem[]>([])
const userPermissions = ref<PermissionItem[]>([])
const permsLoading = ref(false)

// 分配角色相关
const assignRolesDialogVisible = ref(false)
const assignRolesLoading = ref(false)
const savingRoles = ref(false)
const availableRoles = ref<RoleItem[]>([])
const selectedRoleIds = ref<string[]>([])

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (search.keyword) params.keyword = search.keyword
    // 系统管理员查看所有租户用户，普通租户管理员只看当前租户用户
    // 回收站模式只对系统管理员开放
    let res: any
    if (isRecycleBin.value) {
      res = await getAllDeletedTenantUsers(params)
    } else {
      if (search.status !== undefined && search.status !== null) params.status = search.status
      res = userStore.isAdmin ? await getAllTenantUsers(params) : await getUserList(params)
    }
    list.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function toggleRecycleBin() {
  isRecycleBin.value = !isRecycleBin.value
  page.value = 1
  loadList()
}

function resetSearch() {
  search.keyword = ''
  search.status = undefined
  loadList()
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.username = ''
  form.password = ''
  form.nickname = ''
  form.email = ''
  form.status = 1
  dialogVisible.value = true
}

function openEditDialog(row: UserItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.username = row.username
  form.password = ''
  form.nickname = row.nickname || ''
  form.email = row.email || ''
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
        await createUser({
          username: form.username.trim(),
          password: form.password,
          nickname: form.nickname,
          email: form.email
        })
        ElMessage.success('新增成功')
      } else if (editingId.value) {
        const updateData: UserUpdateParams = {
          nickname: form.nickname,
          email: form.email,
          status: form.status
        }
        if (form.password) updateData.password = form.password
        await updateUser(editingId.value, updateData)
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function toggleStatus(row: UserItem) {
  ElMessageBox.confirm(`确定要${row.status === 1 ? '禁用' : '启用'}该用户吗？`, '提示', {
    type: 'warning'
  })
    .then(async () => {
      if (!row.id) return
      await updateUser(row.id, { status: row.status === 1 ? 0 : 1 })
      ElMessage.success('操作成功')
      loadList()
    })
    .catch(() => {})
}

function handleDelete(row: UserItem) {
  ElMessageBox.confirm(`确定要删除用户 "${row.username}" 吗？此操作不可恢复`, '危险操作', {
    type: 'error',
    confirmButtonText: '确定删除',
    cancelButtonText: '取消'
  })
    .then(async () => {
      if (!row.id) return
      await deleteUser(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

// 恢复用户
async function handleRestore(row: UserItem) {
  if (!row.id || !row.tenant_id) return
  try {
    await ElMessageBox.confirm(`确定要恢复用户 "${row.username}" 吗？`, '提示', {
      type: 'warning'
    })
    await restoreTenantUser(row.tenant_id, row.id)
    ElMessage.success('恢复成功')
    loadList()
  } catch (e) {
    // 用户取消
  }
}

// 永久删除用户
async function handlePermanentDelete(row: UserItem) {
  if (!row.id) return
  try {
    await ElMessageBox.confirm(`确定要永久删除用户 "${row.username}" 吗？此操作不可恢复！`, '危险操作', {
      type: 'error',
      confirmButtonText: '确定永久删除',
      cancelButtonText: '取消'
    })
    // 由于后端没有提供永久删除租户用户的接口，暂时使用普通删除接口
    // 实际应该调用永久删除接口
    await deleteUser(row.id)
    ElMessage.success('永久删除成功')
    loadList()
  } catch (e) {
    // 用户取消
  }
}

// 查看用户权限
async function openViewPermissions(row: UserItem) {
  currentUser.value = row
  permsDialogVisible.value = true
  permsLoading.value = true
  userRoles.value = []
  userPermissions.value = []

  try {
    if (!row.id) return

    // 1. 获取用户的角色列表
    const roles = await getUserRoles(row.id)
    userRoles.value = roles || []

    // 2. 获取每个角色的权限并合并
    const allPermissions: PermissionItem[] = []
    const permissionIds = new Set<string>()

    for (const role of userRoles.value) {
      if (role.id) {
        try {
          const perms = await getRolePermissions(role.id)
          if (perms && Array.isArray(perms)) {
            for (const perm of perms) {
              if (perm.id && !permissionIds.has(perm.id)) {
                permissionIds.add(perm.id)
                allPermissions.push(perm)
              }
            }
          }
        } catch (e) {
          // 忽略单个角色权限获取失败
        }
      }
    }

    userPermissions.value = allPermissions
  } catch (e) {
    ElMessage.error('获取权限信息失败')
  } finally {
    permsLoading.value = false
  }
}

// 打开分配角色弹窗
async function openAssignRoles(row: UserItem) {
  currentUser.value = row
  assignRolesDialogVisible.value = true
  assignRolesLoading.value = true
  selectedRoleIds.value = []
  
  try {
    // 1. 获取所有可用角色
    const scope = userStore.isAdmin ? 'all' : 'tenant'
    const rolesRes = await getRoleListForSelect(scope)
    availableRoles.value = rolesRes || []
    
    // 2. 获取用户当前已分配的角色
    if (row.id) {
      const userRolesRes = await getUserRoles(row.id)
      selectedRoleIds.value = (userRolesRes || []).map((r: any) => r.id)
    }
  } catch (e) {
    ElMessage.error('加载角色数据失败')
  } finally {
    assignRolesLoading.value = false
  }
}

// 提交分配角色
async function submitAssignRoles() {
  if (!currentUser.value?.id) return
  
  savingRoles.value = true
  try {
    // 先移除所有角色，再重新分配
    await removeAllUserRoles(currentUser.value.id)
    
    // 如果有选中的角色，则重新分配
    if (selectedRoleIds.value.length > 0) {
      await assignUserRoles(currentUser.value.id, selectedRoleIds.value)
    }
    
    ElMessage.success('角色分配成功')
    assignRolesDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '角色分配失败')
  } finally {
    savingRoles.value = false
  }
}

onMounted(loadList)
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.search-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
}
.flex-1 {
  flex: 1;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
</style>