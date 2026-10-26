<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
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
        <template v-if="!isRecycleBin && !userStore.isAdmin">
          <el-select v-model="query.role" placeholder="角色" clearable filterable style="width: 140px">
            <el-option v-for="r in roleFilterOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
          <el-select v-model="query.sort" placeholder="排序" clearable style="width: 130px">
            <el-option label="注册时间" value="created_at" />
            <el-option label="用户名" value="username" />
            <el-option label="状态" value="status" />
            <el-option label="最后登录" value="last_login_at" />
          </el-select>
          <el-select v-model="query.order" placeholder="顺序" clearable style="width: 100px">
            <el-option label="降序" value="desc" />
            <el-option label="升序" value="asc" />
          </el-select>
          <el-date-picker
            v-model="createdRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            range-separator="至"
            start-placeholder="注册起始"
            end-placeholder="注册结束"
            clearable
            style="width: 240px"
          />
        </template>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="toggleRecycleBin">
          <el-icon><Delete /></el-icon>{{ isRecycleBin ? '返回列表' : '回收站' }}
        </el-button>
        <el-button v-if="userStore.hasPermission('user:create') && !isRecycleBin" type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>新增用户
        </el-button>
        <el-button
          v-if="userStore.hasPermission('user:create') && !isRecycleBin"
          type="primary"
          plain
          @click="openImportDialog"
        >
          <el-icon><Upload /></el-icon>批量导入
        </el-button>
      </div>
      <!-- 列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column prop="department" label="部门" min-width="120">
          <template #default="{ row }">
            <span v-if="row.department">{{ row.department }}</span>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
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
        <el-table-column prop="last_login_at" label="最后登录" width="180">
          <template #default="{ row }">
            <span v-if="row.last_login_at">{{ row.last_login_at }}</span>
            <span v-else style="color: #9ca3af">未登录</span>
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
                v-if="!row.is_master"
                link
                type="danger"
                @click="handlePermanentDelete(row)"
              >永久删除</el-button>
            </template>
            <template v-else>
              <el-button
                v-if="userStore.hasPermission('user:update')"
                link
                type="primary"
                @click="openEditDialog(row)"
              >编辑</el-button>
              <el-button
                v-if="userStore.hasPermission('user:reset_password')"
                link
                type="warning"
                @click="openResetPasswordDialog(row)"
              >重置密码</el-button>
              <el-button
                link
                type="info"
                @click="openViewPermissions(row)"
              >查看权限</el-button>
              <el-button
                link
                type="primary"
                @click="openLoginLogs(row)"
              >登录日志</el-button>
              <el-button
                v-if="userStore.hasPermission('rbac:user_role:assign') || userStore.isAdmin"
                link
                type="success"
                @click="openAssignRoles(row)"
              >分配角色</el-button>
              <el-button
                v-if="userStore.hasPermission('user:update')"
                link
                type="warning"
                @click="toggleStatus(row)"
              >{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
              <el-button
                v-if="userStore.hasPermission('user:delete') && !row.is_master"
                link
                type="danger"
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

    <!-- 批量导入弹窗：租户管理员在当前租户内导入，系统管理员在弹窗内选择目标租户 -->
    <UserImportDialog v-model="importDialogVisible" :admin-mode="userStore.isAdmin" @imported="onImported" />

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
        <el-form-item v-if="dialogMode === 'create' && userStore.isAdmin" label="目标租户" required>
          <el-select
            v-model="createTenantId"
            filterable
            clearable
            :loading="createTenantLoading"
            placeholder="选择在哪个租户下创建"
            style="width: 100%"
          >
            <el-option v-for="t in createTenantOptions" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
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
        <el-form-item label="部门" prop="department">
          <el-input v-model="form.department" placeholder="请输入部门" maxlength="100" />
        </el-form-item>
        <el-form-item label="岗位" prop="position">
          <el-input v-model="form.position" placeholder="请输入岗位" maxlength="100" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" label="角色" prop="role_ids">
          <!-- 后端创建接口的 role_ids 为必填（min=1），不选会直接 400 -->
          <el-select
            v-model="createRoleIds"
            multiple
            filterable
            clearable
            :loading="createRoleLoading"
            placeholder="至少选择一个角色"
            style="width: 100%"
          >
            <el-option v-for="r in createRoleOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" label="备注" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="管理员备注" maxlength="255" />
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

    <!-- 重置密码弹窗 -->
    <el-dialog
      v-model="resetPwdDialogVisible"
      title="重置密码"
      width="400px"
      :close-on-click-modal="false"
      @close="closeResetPwdDialog"
    >
      <p style="margin-bottom: 16px;">正在为用户 <strong>{{ resetPwdUser?.username }}</strong> 重置密码</p>
      <el-form
        ref="resetPwdFormRef"
        :model="resetPwdForm"
        :rules="resetPwdRules"
        label-width="100px"
      >
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="resetPwdForm.new_password"
            type="password"
            placeholder="请输入新密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input
            v-model="resetPwdForm.confirm_password"
            type="password"
            placeholder="请再次输入新密码"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeResetPwdDialog">取消</el-button>
        <el-button type="primary" :loading="resetPwdSaving" @click="submitResetPassword">确定</el-button>
      </template>
    </el-dialog>

    <!-- 登录日志弹窗 -->
    <el-dialog
      v-model="loginLogsDialogVisible"
      :title="`登录日志 - ${loginLogsUser?.username || ''}`"
      width="900px"
      :close-on-click-modal="false"
    >
      <el-table v-loading="loginLogsLoading" :data="loginLogsList" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (loginLogsPage - 1) * loginLogsPageSize + i + 1" />
        <el-table-column prop="action" label="操作" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.action === 'login' ? 'success' : 'info'">
              {{ row.action === 'login' ? '登录' : '登出' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="result" label="结果" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
              {{ row.result === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" label="登录IP" width="140" />
        <el-table-column prop="user_agent" label="用户代理" min-width="200" show-overflow-tooltip />
        <el-table-column prop="created_at" label="时间" width="180" />
      </el-table>
      <div class="pagination" style="margin-top: 16px; display: flex; justify-content: flex-end;">
        <el-pagination
          v-model:current-page="loginLogsPage"
          v-model:page-size="loginLogsPageSize"
          :page-sizes="[10, 20, 50]"
          :total="loginLogsTotal"
          layout="total, sizes, prev, pager, next"
          @size-change="loadLoginLogs"
          @current-change="loadLoginLogs"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus, Delete, Upload } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import {
  getUserList,
  getAllTenantUsers,
  getAllDeletedTenantUsers,
  getDeletedUserList,
  createUser,
  createTenantUser,
  updateUser,
  updateTenantUser,
  resetUserPassword,
  resetTenantUserPassword,
  deleteUser,
  restoreTenantUser,
  permanentDeleteTenantUser,
  restoreUser,
  permanentDeleteUser,
  type UserItem,
  type UserCreateParams,
  type UserUpdateParams,
  type UserListParams
} from '@/api/modules/user'
import { getUserRoles, getRolePermissions, type RoleItem, getRoleListForSelect, assignUserRoles, removeAllUserRoles } from '@/api/modules/role'
import { getTenantList, type TenantItem } from '@/api/modules/tenant'
import { type PermissionItem } from '@/api/modules/permission'
import { getAuditLogList, type AuditLogItem } from '@/api/modules/audit'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import UserImportDialog from './components/UserImportDialog.vue'

const userStore = useUserStore()

const isRecycleBin = ref(false)

const {
  list,
  total,
  page,
  pageSize,
  loading,
  query,
  reset,
  reload
} = useTableList<UserItem, {
  keyword: string
  status?: number
  role?: string
  sort?: string
  order?: 'asc' | 'desc'
  created_from?: string
  created_to?: string
}>({
  fetchList: async (params) => {
    const req: UserListParams = { page: params.page, page_size: params.page_size }
    if (params.keyword) req.keyword = params.keyword
    // 系统管理员查看所有租户用户，普通租户管理员只看当前租户用户
    let res: unknown
    if (isRecycleBin.value) {
      // 回收站模式：系统管理员看所有，普通租户管理员只看当前租户
      res = userStore.isAdmin ? await getAllDeletedTenantUsers(req) : await getDeletedUserList(req)
    } else {
      if (params.status !== undefined && params.status !== null) req.status = params.status
      if (userStore.isAdmin) {
        res = await getAllTenantUsers(req)
      } else {
        // 租户内列表支持角色/排序/注册时间区间筛选（仅非管理员生效）
        if (params.role) req.role = params.role
        if (params.sort) req.sort = params.sort
        if (params.order) req.order = params.order
        if (params.created_from) req.created_from = params.created_from
        if (params.created_to) req.created_to = params.created_to
        res = await getUserList(req)
      }
    }
    return toPageResult<UserItem>(res)
  },
  initialQuery: { keyword: '', status: undefined, role: undefined, sort: undefined, order: undefined, created_from: undefined, created_to: undefined }
})
const loadList = reload

// 注册时间区间：映射到 query.created_from / query.created_to
const createdRange = computed<[string, string] | null>({
  get: () => (query.created_from && query.created_to ? [query.created_from, query.created_to] as [string, string] : null),
  set: (val) => {
    if (val && val.length === 2) {
      query.created_from = val[0]
      query.created_to = val[1]
    } else {
      query.created_from = undefined
      query.created_to = undefined
    }
  }
})

// 角色筛选下拉项（仅租户管理员视图使用）
const roleFilterOptions = ref<RoleItem[]>([])
async function loadRoleFilterOptions() {
  try {
    roleFilterOptions.value = await getRoleListForSelect('tenant') || []
  } catch (e) {
    console.error('加载角色筛选选项失败', e)
  }
}
onMounted(() => {
  if (!userStore.isAdmin) loadRoleFilterOptions()
})

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)
const editingTenantId = ref<string | null>(null)
// 系统管理员新建用户时必须显式选租户：登录态租户是虚拟的 SYSTEM_ROOT，后端会拒绝
const createTenantId = ref('')
const createTenantOptions = ref<TenantItem[]>([])
const createTenantLoading = ref(false)
const createTenantLoaded = ref(false)
// 创建接口要求至少一个角色（role_ids min=1），不选会直接 400
const createRoleIds = ref<string[]>([])
const createRoleOptions = ref<RoleItem[]>([])
const createRoleLoading = ref(false)
const createRoleLoaded = ref(false)
const form = reactive<UserCreateParams & UserUpdateParams>({
  username: '',
  password: '',
  nickname: '',
  email: '',
  department: '',
  position: '',
  remark: '',
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

// 重置密码相关
const resetPwdDialogVisible = ref(false)
const resetPwdFormRef = ref<FormInstance>()
const resetPwdSaving = ref(false)
const resetPwdUser = ref<UserItem | null>(null)
const resetPwdForm = reactive({
  new_password: '',
  confirm_password: ''
})
const resetPwdRules: FormRules = {
  new_password: [{ required: true, min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' }],
  confirm_password: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    {
      validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
        if (value !== resetPwdForm.new_password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 登录日志相关
const loginLogsDialogVisible = ref(false)
const loginLogsLoading = ref(false)
const loginLogsUser = ref<UserItem | null>(null)
const loginLogsList = ref<AuditLogItem[]>([])
const loginLogsPage = ref(1)
const loginLogsPageSize = ref(10)
const loginLogsTotal = ref(0)

// 批量导入相关
const importDialogVisible = ref(false)

function openImportDialog() {
  importDialogVisible.value = true
}

function onImported() {
  loadList()
}

function toggleRecycleBin() {
  isRecycleBin.value = !isRecycleBin.value
  page.value = 1
  void reload()
}

function resetSearch() {
  reset()
}

async function loadCreateTenantOptions() {
  if (createTenantLoaded.value) return
  createTenantLoaded.value = true
  createTenantLoading.value = true
  try {
    const res = await getTenantList({ page: 1, page_size: 200 })
    // 停用租户下的账号无法登录，不作为新建目标
    createTenantOptions.value = (res?.data ?? []).filter((t) => t.status === 1)
  } catch {
    createTenantLoaded.value = false
  } finally {
    createTenantLoading.value = false
  }
}

async function loadCreateRoleOptions() {
  if (createRoleLoaded.value) return
  createRoleLoading.value = true
  try {
    // 租户级角色（含 scope=all 的平台角色）才能分配给租户用户
    createRoleOptions.value = (await getRoleListForSelect('tenant')) ?? []
    createRoleLoaded.value = true
  } catch {
    // 拉取失败不阻断弹窗：不选角色提交时后端会给出明确提示
  } finally {
    createRoleLoading.value = false
  }
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  editingTenantId.value = null
  createTenantId.value = ''
  createRoleIds.value = []
  if (userStore.isAdmin) loadCreateTenantOptions()
  loadCreateRoleOptions()
  form.username = ''
  form.password = ''
  form.nickname = ''
  form.email = ''
  form.department = ''
  form.position = ''
  form.remark = ''
  form.status = 1
  dialogVisible.value = true
}

function openEditDialog(row: UserItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  editingTenantId.value = row.tenant_id || null
  form.username = row.username
  form.password = ''
  form.nickname = row.nickname || ''
  form.email = row.email || ''
  form.department = row.department || ''
  form.position = row.position || ''
  form.remark = row.remark || ''
  form.status = row.status ?? 1
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    // 系统管理员不选租户就提交，会落到登录态的虚拟租户里，直接拦住
    if (dialogMode.value === 'create' && userStore.isAdmin && !createTenantId.value) {
      ElMessage.warning('请先选择目标租户')
      return
    }
    if (dialogMode.value === 'create' && !createRoleIds.value.length) {
      ElMessage.warning('请至少选择一个角色')
      return
    }
    saving.value = true
    try {
      if (dialogMode.value === 'create') {
        const payload: UserCreateParams = {
          username: form.username.trim(),
          password: form.password,
          nickname: form.nickname,
          email: form.email,
          department: form.department,
          position: form.position,
          role_ids: createRoleIds.value
        }
        // 系统管理员的登录态租户是虚拟租户 SYSTEM_ROOT，必须走跨租户接口带上目标租户
        if (userStore.isAdmin) {
          await createTenantUser(createTenantId.value, payload)
        } else {
          await createUser(payload)
        }
        ElMessage.success('新增成功')
      } else if (editingId.value) {
        const updateData: UserUpdateParams = {
          nickname: form.nickname,
          email: form.email,
          department: form.department,
          position: form.position,
          remark: form.remark,
          status: form.status
        }
        if (form.password) updateData.password = form.password
        // 系统管理员使用跨租户接口，普通租户管理员使用当前租户接口
        if (userStore.isAdmin && editingTenantId.value) {
          await updateTenantUser(editingTenantId.value, editingId.value, updateData)
        } else {
          await updateUser(editingId.value, updateData)
        }
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
      // 系统管理员使用跨租户接口，普通租户管理员使用当前租户接口
      if (userStore.isAdmin && row.tenant_id) {
        await updateTenantUser(row.tenant_id, row.id, { status: row.status === 1 ? 0 : 1 })
      } else {
        await updateUser(row.id, { status: row.status === 1 ? 0 : 1 })
      }
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
  if (!row.id) return
  try {
    await ElMessageBox.confirm(`确定要恢复用户 "${row.username}" 吗？`, '提示', {
      type: 'warning'
    })
    // 系统管理员使用跨租户接口，普通租户管理员使用当前租户接口
    if (userStore.isAdmin && row.tenant_id) {
      await restoreTenantUser(row.tenant_id, row.id)
    } else {
      await restoreUser(row.id)
    }
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
    // 系统管理员使用跨租户接口，普通租户管理员使用当前租户接口
    if (userStore.isAdmin && row.tenant_id) {
      await permanentDeleteTenantUser(row.tenant_id, row.id)
    } else {
      await permanentDeleteUser(row.id)
    }
    ElMessage.success('永久删除成功')
    loadList()
  } catch (e) {
    // 用户取消
  }
}

// 打开重置密码对话框
function openResetPasswordDialog(row: UserItem) {
  resetPwdUser.value = row
  resetPwdForm.new_password = ''
  resetPwdForm.confirm_password = ''
  resetPwdDialogVisible.value = true
}

// 关闭重置密码对话框
function closeResetPwdDialog() {
  resetPwdDialogVisible.value = false
  resetPwdUser.value = null
  resetPwdForm.new_password = ''
  resetPwdForm.confirm_password = ''
}

// 提交重置密码
async function submitResetPassword() {
  if (!resetPwdFormRef.value) return
  await resetPwdFormRef.value.validate(async (valid) => {
    if (!valid) return
    if (!resetPwdUser.value?.id) return

    resetPwdSaving.value = true
    try {
      // 系统管理员且用户有租户ID时，使用管理员专用接口
      if (userStore.isAdmin && resetPwdUser.value.tenant_id) {
        await resetTenantUserPassword(resetPwdUser.value.tenant_id, resetPwdUser.value.id, {
          new_password: resetPwdForm.new_password,
          confirm_password: resetPwdForm.confirm_password
        })
      } else {
        await resetUserPassword(resetPwdUser.value.id, {
          new_password: resetPwdForm.new_password,
          confirm_password: resetPwdForm.confirm_password
        })
      }
      ElMessage.success('密码重置成功')
      closeResetPwdDialog()
    } catch (e) {
      // 错误已在拦截器处理
    } finally {
      resetPwdSaving.value = false
    }
  })
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
      selectedRoleIds.value = (userRolesRes || []).map((r) => r.id)
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
  } catch (e) {
    ElMessage.error((e as Error).message || '角色分配失败')
  } finally {
    savingRoles.value = false
  }
}

// 打开登录日志弹窗
function openLoginLogs(row: UserItem) {
  loginLogsUser.value = row
  loginLogsPage.value = 1
  loginLogsPageSize.value = 10
  loginLogsDialogVisible.value = true
  loadLoginLogs()
}

// 加载登录日志
async function loadLoginLogs() {
  if (!loginLogsUser.value?.id) return
  
  loginLogsLoading.value = true
  try {
    const res = await getAuditLogList({
      page: loginLogsPage.value,
      page_size: loginLogsPageSize.value,
      user_id: loginLogsUser.value.id,
      action: 'login'  // 只查询登录相关的日志
    })
    loginLogsList.value = res.items
    loginLogsTotal.value = res.total
  } catch (e) {
    ElMessage.error('加载登录日志失败')
  } finally {
    loginLogsLoading.value = false
  }
}

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