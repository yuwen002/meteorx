<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="search.name"
          placeholder="搜索租户名称"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select v-model="search.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="openRecycleBin">
          <el-icon><Delete /></el-icon>回收站
        </el-button>
        <el-button type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>新增租户
        </el-button>
      </div>

      <!-- 批量操作 -->
      <div class="batch-bar" v-if="selectedIds.length > 0">
        <el-button type="danger" @click="handleBatchDelete">批量删除</el-button>
        <el-button type="warning" @click="handleBatchDisable">批量禁用</el-button>
        <el-button type="success" @click="handleBatchEnable">批量启用</el-button>
        <span style="margin-left: 8px; color: #666">已选择 {{ selectedIds.length }} 项</span>
      </div>

      <!-- 列表 -->
      <el-table
        :data="list"
        border
        stripe
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" label="租户名称" min-width="150" />
        <el-table-column prop="domain" label="域名" min-width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.domain }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="contact_email" label="联系邮箱" min-width="180" />
        <el-table-column prop="region" label="地区" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="套餐" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.plan_name" :type="row.plan_expired ? 'danger' : 'primary'" effect="plain" size="small">
              {{ row.plan_name }}<template v-if="row.plan_expired">(已过期)</template>
            </el-tag>
            <el-tag v-else type="info" effect="plain" size="small">未开通</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="400" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="info" @click="openViewUsers(row)">查看用户</el-button>
            <el-button link type="warning" @click="openPlanDialog(row)">套餐</el-button>
            <el-button
              link
              :type="row.status === 1 ? 'warning' : 'success'"
              @click="handleToggleStatus(row)"
            >
              {{ row.status === 1 ? '禁用' : '启用' }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
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

    <!-- 查看用户弹窗 -->
    <el-dialog
      v-model="usersDialogVisible"
      :title="`租户用户 - ${currentTenant?.name || ''}`"
      width="1000px"
      :close-on-click-modal="false"
    >
      <!-- 搜索栏 -->
      <div class="search-bar" style="margin-bottom: 16px;">
        <el-input
          v-model="userSearch.keyword"
          placeholder="搜索用户名/昵称/邮箱"
          clearable
          style="width: 220px"
          @keyup.enter="loadTenantUsers"
        />
        <el-button type="primary" @click="loadTenantUsers">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button @click="resetUserSearch">重置</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="toggleUserRecycleBin">
          <el-icon><Delete /></el-icon>{{ isUserRecycleBin ? '返回列表' : '回收站' }}
        </el-button>
        <el-button type="success" v-if="!isUserRecycleBin" @click="openCreateUserDialog">
          <el-icon><Plus /></el-icon>新增用户
        </el-button>
      </div>

      <!-- 批量操作 -->
      <div v-if="userSelectedIds.length > 0 && !isUserRecycleBin" class="batch-bar" style="margin-bottom: 16px;">
        <el-button type="danger" size="small" @click="handleBatchDeleteUsers">批量删除</el-button>
        <el-button type="warning" size="small" @click="handleBatchDisableUsers">批量禁用</el-button>
        <el-button type="success" size="small" @click="handleBatchEnableUsers">批量启用</el-button>
        <span style="margin-left: 8px; color: #666">已选择 {{ userSelectedIds.length }} 项</span>
      </div>

      <!-- 用户列表 -->
      <el-table
        :data="userList"
        v-loading="userLoading"
        border
        stripe
        @selection-change="handleUserSelectionChange"
      >
        <el-table-column v-if="!isUserRecycleBin" type="selection" width="55" />
        <el-table-column prop="username" label="用户名" min-width="120" />
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column label="角色" min-width="150">
          <template #default="{ row }">
            <el-tag
              v-for="role in row.role_list"
              :key="role.id"
              size="small"
              type="primary"
              style="margin-right: 4px; margin-bottom: 4px;"
            >
              {{ role.name }}
            </el-tag>
            <span v-if="!row.role_list || row.role_list.length === 0" style="color: #909399; font-size: 12px;">无角色</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160" />
        <el-table-column v-if="isUserRecycleBin" prop="deleted_at" label="删除时间" width="160" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <template v-if="isUserRecycleBin">
              <el-button link type="success" @click="handleRestoreUser(row)">恢复</el-button>
              <el-button link type="danger" @click="handlePermanentDeleteUser(row)">永久删除</el-button>
            </template>
            <template v-else>
              <el-button link type="primary" @click="openEditUserDialog(row)">编辑</el-button>
              <el-button link type="warning" @click="openResetPasswordDialog(row)">重置密码</el-button>
              <el-button link type="info" @click="openViewUserRoles(row)">查看角色</el-button>
              <el-button link type="primary" @click="openLoginLogs(row)">登录日志</el-button>
              <el-button
                link
                :type="row.status === 1 ? 'warning' : 'success'"
                @click="handleToggleUserStatus(row)"
              >
                {{ row.status === 1 ? '禁用' : '启用' }}
              </el-button>
              <el-button link type="danger" @click="handleDeleteUser(row)">删除</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination" style="margin-top: 16px;">
        <el-pagination
          v-if="!isUserRecycleBin"
          v-model:current-page="userPage"
          v-model:page-size="userPageSize"
          :total="userTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadTenantUsers"
          @current-change="loadTenantUsers"
        />
        <div v-else style="text-align: center; color: #909399; font-size: 14px;">
          共 {{ userTotal }} 条已删除记录
        </div>
      </div>
    </el-dialog>

    <!-- 新增/编辑用户弹窗 -->
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
      <el-table :data="loginLogsList" border stripe v-loading="loginLogsLoading" style="width: 100%">
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

    <!-- 查看用户角色弹窗 -->
    <el-dialog
      v-model="userRolesDialogVisible"
      :title="`用户角色 - ${currentUser?.username || ''}`"
      width="500px"
      :close-on-click-modal="false"
    >
      <div v-loading="userRolesLoading">
        <div v-if="currentUserRoles.length > 0">
          <el-table :data="currentUserRoles" border stripe size="small">
            <el-table-column prop="name" label="角色名称" min-width="120" />
            <el-table-column prop="code" label="角色编码" min-width="120" />
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" size="small" @click="handleRemoveUserRole(row)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 16px; text-align: center;">
            <el-button type="danger" @click="handleRemoveAllUserRoles">移除所有角色</el-button>
          </div>
        </div>
        <el-empty v-else description="该用户暂无角色" />
      </div>
    </el-dialog>

    <el-dialog
      v-model="userDialogVisible"
      :title="userDialogMode === 'create' ? '新增用户' : '编辑用户'"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="userFormRef" :model="userForm" :rules="userRules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="userForm.username" placeholder="请输入用户名" :disabled="userDialogMode === 'edit'" />
        </el-form-item>
        <el-form-item label="密码" prop="password" v-if="userDialogMode === 'create'">
          <el-input v-model="userForm.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="userForm.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="角色" prop="role_ids">
          <el-select
            v-model="userForm.role_ids"
            multiple
            placeholder="请选择角色"
            style="width: 100%"
            :loading="roleLoading"
          >
            <el-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="userSaving" @click="submitUserForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增租户' : '编辑租户'"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="租户名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入租户名称" />
        </el-form-item>
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" placeholder="请输入域名" :disabled="dialogMode === 'edit'">
            <template #append>.meteorx.com</template>
          </el-input>
        </el-form-item>
        <el-form-item label="联系邮箱" prop="contact_email">
          <el-input v-model="form.contact_email" placeholder="请输入联系邮箱" />
        </el-form-item>
        <el-form-item label="地区" prop="region">
          <el-input v-model="form.region" placeholder="请输入地区" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="Logo" prop="logo">
          <el-input v-model="form.logo" placeholder="请输入Logo URL" />
        </el-form-item>
        <el-form-item label="状态" prop="status" v-if="dialogMode === 'create'">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 初始管理员信息（仅创建时显示） -->
        <template v-if="dialogMode === 'create'">
          <el-divider>初始管理员信息</el-divider>
          <el-form-item label="用户名" prop="admin_user.username">
            <el-input v-model="form.admin_user.username" placeholder="请输入管理员用户名" />
          </el-form-item>
          <el-form-item label="密码" prop="admin_user.password">
            <el-input v-model="form.admin_user.password" type="password" show-password placeholder="请输入密码" />
          </el-form-item>
          <el-form-item label="昵称" prop="admin_user.nickname">
            <el-input v-model="form.admin_user.nickname" placeholder="请输入管理员昵称" />
          </el-form-item>
          <el-form-item label="邮箱" prop="admin_user.email">
            <el-input v-model="form.admin_user.email" placeholder="请输入管理员邮箱" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 套餐管理弹窗 -->
    <el-dialog
      v-model="planDialogVisible"
      :title="`套餐配置 - ${currentPlanTenant?.name || ''}`"
      width="560px"
      :close-on-click-modal="false"
    >
      <template v-if="currentPlan">
        <div class="plan-info">
          <div class="plan-info-item">
            <span class="label">当前套餐</span>
            <el-tag type="primary" effect="plain">{{ currentPlan.plan_name }}</el-tag>
            <el-tag v-if="currentPlan.status === 2" type="danger" effect="plain" style="margin-left: 6px">已过期</el-tag>
          </div>
          <div class="plan-info-item">
            <span class="label">用户用量</span>
            <el-progress :percentage="planUsagePercent" :status="planUsagePercent >= 100 ? 'exception' : undefined" style="width: 300px" />
            <span class="usage-text">{{ currentPlan.current_users }} / {{ currentPlan.user_limit === -1 ? '不限' : currentPlan.user_limit }}</span>
          </div>
          <div class="plan-info-item">
            <span class="label">生效时间</span>
            <span>{{ currentPlan.started_at || '-' }}</span>
          </div>
          <div class="plan-info-item" v-if="currentPlan.expires_at">
            <span class="label">到期时间</span>
            <span>{{ currentPlan.expires_at }}（剩余 {{ currentPlan.effective_days ?? 0 }} 天）</span>
          </div>
          <div class="plan-info-item" v-else>
            <span class="label">到期时间</span>
            <span>长期有效</span>
          </div>
        </div>
        <el-divider />
      </template>
      <el-alert v-else title="该租户尚未开通套餐" type="info" show-icon :closable="false" style="margin-bottom: 16px" />
      <template v-if="planList.length > 0">
        <div class="plan-info-item">
          <span class="label">选择套餐</span>
          <el-select v-model="assignForm.plan_id" placeholder="请选择套餐" style="width: 300px" :loading="planLoading">
            <el-option v-for="p in planList" :key="p.id" :label="`${p.name}（${p.user_limit === -1 ? '不限' : p.user_limit + ' 人'}）`" :value="p.id" />
          </el-select>
        </div>
        <div class="plan-info-item">
          <span class="label">到期时间</span>
          <el-date-picker
            v-model="assignForm.expires_at"
            type="datetime"
            placeholder="不设置则长期有效"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 300px"
          />
        </div>
      </template>
      <el-alert v-else title="暂无可用套餐，请先在套餐管理中创建" type="warning" show-icon :closable="false" />
      <template #footer>
        <el-button @click="planDialogVisible = false">关闭</el-button>
        <el-button type="primary" :loading="planSaving" :disabled="!assignForm.plan_id || planList.length === 0" @click="submitPlanAssign">
          保存套餐
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Search, Plus, Delete } from '@element-plus/icons-vue'
import {
  getTenantList,
  createTenant,
  updateTenant,
  updateTenantStatus,
  deleteTenant,
  batchUpdateTenantStatus,
  batchDeleteTenants,
  type TenantItem,
  type CreateTenantParams,
  type UpdateTenantParams
} from '@/api/modules/tenant'
import {
  getTenantUsers,
  getAllDeletedTenantUsers,
  createTenantUser,
  updateTenantUser,
  deleteTenantUser,
  updateTenantUserStatus,
  batchDeleteTenantUsers,
  batchUpdateTenantUserStatus,
  restoreTenantUser,
  permanentDeleteTenantUser,
  resetTenantUserPassword,
  type UserItem,
  type UserCreateParams,
  type UserUpdateParams
} from '@/api/modules/user'
import { getAuditLogList, type AuditLogItem } from '@/api/modules/audit'
import { getRoleList, getRolesForSelect, getUserRoles, removeUserRole, removeAllUserRoles, type RoleItem } from '@/api/modules/role'
import { getPlanSelect, getTenantPlan, assignTenantPlan, type PlanItem, type CurrentPlan } from '@/api/modules/plan'

const router = useRouter()

const list = ref<TenantItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const search = reactive({ name: '', status: undefined as number | undefined })
const selectedIds = ref<string[]>([])

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const formRef = ref<FormInstance>()
const saving = ref(false)
const editingId = ref<string | null>(null)

// 查看用户相关
const usersDialogVisible = ref(false)
const currentTenant = ref<TenantItem | null>(null)
const userList = ref<UserItem[]>([])
const userPage = ref(1)
const userPageSize = ref(10)
const userTotal = ref(0)
const userLoading = ref(false)
const userSearch = reactive({ keyword: '' })
const userSelectedIds = ref<string[]>([])
const isUserRecycleBin = ref(false)

// 用户新增/编辑相关
const userDialogVisible = ref(false)
const userDialogMode = ref<'create' | 'edit'>('create')
const userFormRef = ref<FormInstance>()
const userSaving = ref(false)
const editingUserId = ref<string | null>(null)
const userForm = reactive<UserCreateParams & UserUpdateParams & { role_ids: string[] }>({
  username: '',
  password: '',
  nickname: '',
  email: '',
  role_ids: []
})
const userRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' }
  ],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  role_ids: [{ required: true, message: '请选择角色', trigger: 'change', type: 'array' }]
}

// 角色列表
const roleList = ref<RoleItem[]>([])
const roleLoading = ref(false)

// 查看用户角色相关
const userRolesDialogVisible = ref(false)
const currentUser = ref<UserItem | null>(null)
const currentUserRoles = ref<RoleItem[]>([])
const userRolesLoading = ref(false)

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
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' }
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

// 套餐管理相关
const planDialogVisible = ref(false)
const planLoading = ref(false)
const planSaving = ref(false)
const currentPlanTenant = ref<TenantItem | null>(null)
const currentPlan = ref<CurrentPlan | null>(null)
const planList = ref<PlanItem[]>([])
const assignForm = reactive<{ plan_id: string; expires_at: string }>({
  plan_id: '',
  expires_at: ''
})
const planUsagePercent = computed(() => {
  if (!currentPlan.value || currentPlan.value.user_limit === -1 || currentPlan.value.user_limit <= 0) return 0
  return Math.min(100, Math.round((currentPlan.value.current_users / currentPlan.value.user_limit) * 100))
})

// 打开套餐配置弹窗
async function openPlanDialog(row: TenantItem) {
  currentPlanTenant.value = row
  currentPlan.value = null
  assignForm.plan_id = ''
  assignForm.expires_at = ''
  planDialogVisible.value = true
  planLoading.value = true
  try {
    const [plans, plan] = await Promise.all([getPlanSelect(), getTenantPlan(row.id)])
    planList.value = plans || []
    if (plan) {
      currentPlan.value = plan
      assignForm.plan_id = plan.plan_id
      if (plan.expires_at) assignForm.expires_at = plan.expires_at
    }
  } catch (e) {
    planList.value = planList.value || []
  } finally {
    planLoading.value = false
  }
}

// 提交套餐分配
async function submitPlanAssign() {
  if (!currentPlanTenant.value?.id || !assignForm.plan_id) return
  planSaving.value = true
  try {
    await assignTenantPlan(currentPlanTenant.value.id, {
      plan_id: assignForm.plan_id,
      expires_at: assignForm.expires_at || undefined
    })
    ElMessage.success('套餐保存成功')
    planDialogVisible.value = false
    loadList()
  } catch (e) {
    // 错误已在拦截器处理
  } finally {
    planSaving.value = false
  }
}

const form = reactive<CreateTenantParams & UpdateTenantParams>({
  name: '',
  domain: '',
  description: '',
  contact_email: '',
  region: '',
  logo: '',
  status: 1,
  admin_user: {
    username: '',
    password: '',
    nickname: '',
    email: ''
  }
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入租户名称', trigger: 'blur' }],
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  'admin_user.username': [{ required: true, message: '请输入管理员用户名', trigger: 'blur' }],
  'admin_user.password': [
    { required: true, message: '请输入管理员密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度 6-32 位', trigger: 'blur' }
  ],
  'admin_user.nickname': [{ required: true, message: '请输入管理员昵称', trigger: 'blur' }]
}

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (search.name) params.name = search.name
    if (search.status !== undefined) params.status = search.status
    const res: any = await getTenantList(params)
    list.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function resetSearch() {
  search.name = ''
  search.status = undefined
  page.value = 1
  loadList()
}

function handleSelectionChange(selection: TenantItem[]) {
  selectedIds.value = selection.map((row) => row.id).filter(Boolean) as string[]
}

function openCreateDialog() {
  dialogMode.value = 'create'
  editingId.value = null
  form.name = ''
  form.domain = ''
  form.description = ''
  form.contact_email = ''
  form.region = ''
  form.logo = ''
  form.status = 1
  form.admin_user = { username: '', password: '', nickname: '', email: '' }
  dialogVisible.value = true
}

function openEditDialog(row: TenantItem) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  form.name = row.name
  form.domain = row.domain
  form.description = row.description || ''
  form.contact_email = row.contact_email || ''
  form.region = row.region || ''
  form.logo = row.logo || ''
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (dialogMode.value === 'create') {
        await createTenant(form as CreateTenantParams)
        ElMessage.success('创建成功')
      } else if (editingId.value) {
        await updateTenant(editingId.value, form as UpdateTenantParams)
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

async function handleToggleStatus(row: TenantItem) {
  const newStatus = row.status === 1 ? 0 : 1
  const actionText = newStatus === 1 ? '启用' : '禁用'
  try {
    await ElMessageBox.confirm(`确定要${actionText}租户 "${row.name}" 吗？`, '提示', {
      type: 'warning'
    })
    await updateTenantStatus(row.id, newStatus)
    ElMessage.success(`${actionText}成功`)
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleDelete(row: TenantItem) {
  try {
    await ElMessageBox.confirm(`确定要删除租户 "${row.name}" 吗？`, '提示', { type: 'warning' })
    await deleteTenant(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个租户吗？`, '提示', {
      type: 'warning'
    })
    await batchDeleteTenants({ ids: selectedIds.value })
    ElMessage.success('批量删除成功')
    selectedIds.value = []
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleBatchEnable() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要启用选中的 ${selectedIds.value.length} 个租户吗？`, '提示', {
      type: 'warning'
    })
    await batchUpdateTenantStatus({ ids: selectedIds.value, status: 1 })
    ElMessage.success('批量启用成功')
    selectedIds.value = []
    loadList()
  } catch (e) {
    // 用户取消
  }
}

async function handleBatchDisable() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要禁用选中的 ${selectedIds.value.length} 个租户吗？`, '提示', {
      type: 'warning'
    })
    await batchUpdateTenantStatus({ ids: selectedIds.value, status: 0 })
    ElMessage.success('批量禁用成功')
    selectedIds.value = []
    loadList()
  } catch (e) {
    // 用户取消
  }
}

function openRecycleBin() {
  router.push('/system/tenant/recycle')
}

// 查看租户用户
function openViewUsers(row: TenantItem) {
  currentTenant.value = row
  usersDialogVisible.value = true
  userPage.value = 1
  userSearch.keyword = ''
  isUserRecycleBin.value = false
  loadTenantUsers()
}

function toggleUserRecycleBin() {
  isUserRecycleBin.value = !isUserRecycleBin.value
  userPage.value = 1
  loadTenantUsers()
}

async function loadTenantUsers() {
  if (!currentTenant.value) return
  userLoading.value = true
  try {
    const params: any = { page: userPage.value, page_size: userPageSize.value }
    if (userSearch.keyword) params.keyword = userSearch.keyword
    let res: any
    if (isUserRecycleBin.value) {
      // 回收站模式：获取所有已删除的用户，然后过滤当前租户的
      res = await getAllDeletedTenantUsers(params)
      // 过滤出当前租户的用户
      const allDeletedUsers = res.data || []
      const filteredUsers = allDeletedUsers.filter((u: UserItem) => u.tenant_id === currentTenant.value?.id)
      userList.value = filteredUsers
      userTotal.value = filteredUsers.length
    } else {
      res = await getTenantUsers(currentTenant.value.id, params)
      userList.value = res.data || []
      userTotal.value = res.pagination?.total || 0
    }
  } catch (e) {
    userList.value = []
    userTotal.value = 0
  } finally {
    userLoading.value = false
  }
}

function resetUserSearch() {
  userSearch.keyword = ''
  userPage.value = 1
  loadTenantUsers()
}

function handleUserSelectionChange(selection: UserItem[]) {
  userSelectedIds.value = selection.map((row) => row.id).filter(Boolean) as string[]
}

// 加载角色列表（只加载租户级角色）
async function loadRoleList() {
  if (!currentTenant.value) return
  roleLoading.value = true
  try {
    // 使用 scope=tenant 获取租户级角色（普通用户和租户管理员角色）
    const data = await getRolesForSelect('tenant')
    roleList.value = data || []
  } catch (e) {
    roleList.value = []
  } finally {
    roleLoading.value = false
  }
}

// 打开新增用户弹窗
function openCreateUserDialog() {
  userDialogMode.value = 'create'
  editingUserId.value = null
  userForm.username = ''
  userForm.password = ''
  userForm.nickname = ''
  userForm.email = ''
  userForm.role_ids = []
  loadRoleList()
  userDialogVisible.value = true
}

// 打开编辑用户弹窗
function openEditUserDialog(row: UserItem) {
  userDialogMode.value = 'edit'
  editingUserId.value = row.id
  userForm.username = row.username
  userForm.nickname = row.nickname
  userForm.email = row.email || ''
  userForm.role_ids = row.role_ids || []
  loadRoleList()
  userDialogVisible.value = true
}

// 提交用户表单
async function submitUserForm() {
  if (!userFormRef.value || !currentTenant.value) return
  const tenantId = currentTenant.value.id
  await userFormRef.value.validate(async (valid) => {
    if (!valid) return
    userSaving.value = true
    try {
      if (userDialogMode.value === 'create') {
        await createTenantUser(tenantId, {
          username: userForm.username,
          password: userForm.password,
          nickname: userForm.nickname,
          email: userForm.email,
          role_ids: userForm.role_ids
        } as UserCreateParams)
        ElMessage.success('创建成功')
      } else if (editingUserId.value) {
        await updateTenantUser(tenantId, editingUserId.value, {
          nickname: userForm.nickname,
          email: userForm.email,
          role_ids: userForm.role_ids
        } as UserUpdateParams)
        ElMessage.success('更新成功')
      }
      userDialogVisible.value = false
      loadTenantUsers()
    } finally {
      userSaving.value = false
    }
  })
}

// 切换用户状态
async function handleToggleUserStatus(row: UserItem) {
  if (!currentTenant.value) return
  const newStatus = row.status === 1 ? 0 : 1
  const actionText = newStatus === 1 ? '启用' : '禁用'
  try {
    await ElMessageBox.confirm(`确定要${actionText}用户 "${row.username}" 吗？`, '提示', {
      type: 'warning'
    })
    await updateTenantUserStatus(currentTenant.value.id, row.id, newStatus)
    ElMessage.success(`${actionText}成功`)
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 删除用户
async function handleDeleteUser(row: UserItem) {
  if (!currentTenant.value) return

  try {
    await ElMessageBox.confirm(`确定要删除用户 "${row.username}" 吗？删除后将自动解除角色绑定。`, '提示', { type: 'warning' })
    await deleteTenantUser(currentTenant.value.id, row.id)
    ElMessage.success('删除成功')
    loadTenantUsers()
  } catch (e) {
    // 用户取消或删除失败
  }
}

// 恢复用户
async function handleRestoreUser(row: UserItem) {
  if (!row.id || !row.tenant_id) return
  try {
    await ElMessageBox.confirm(`确定要恢复用户 "${row.username}" 吗？`, '提示', { type: 'warning' })
    await restoreTenantUser(row.tenant_id, row.id)
    ElMessage.success('恢复成功')
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 永久删除用户
async function handlePermanentDeleteUser(row: UserItem) {
  if (!row.id || !row.tenant_id) return
  try {
    await ElMessageBox.confirm(`确定要永久删除用户 "${row.username}" 吗？此操作不可恢复！`, '危险操作', {
      type: 'error',
      confirmButtonText: '确定永久删除',
      cancelButtonText: '取消'
    })
    await permanentDeleteTenantUser(row.tenant_id, row.id)
    ElMessage.success('永久删除成功')
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 查看用户角色
async function openViewUserRoles(row: UserItem) {
  currentUser.value = row
  userRolesDialogVisible.value = true
  userRolesLoading.value = true
  currentUserRoles.value = []

  try {
    if (!row.id) return
    const roles = await getUserRoles(row.id)
    currentUserRoles.value = roles || []
  } catch (e) {
    ElMessage.error('获取角色信息失败')
  } finally {
    userRolesLoading.value = false
  }
}

// 移除用户单个角色
async function handleRemoveUserRole(role: RoleItem) {
  if (!currentUser.value || !currentTenant.value) return

  try {
    await ElMessageBox.confirm(`确定要移除角色 "${role.name}" 吗？`, '提示', { type: 'warning' })
    await removeUserRole(currentUser.value.id, role.id)
    ElMessage.success('移除成功')
    // 刷新角色列表
    const roles = await getUserRoles(currentUser.value.id)
    currentUserRoles.value = roles || []
    // 刷新用户列表（更新角色显示）
    loadTenantUsers()
  } catch (e) {
    // 用户取消或操作失败
  }
}

// 移除用户所有角色
async function handleRemoveAllUserRoles() {
  if (!currentUser.value || !currentTenant.value) return

  try {
    await ElMessageBox.confirm(`确定要移除该用户的所有角色吗？`, '提示', { type: 'warning' })
    await removeAllUserRoles(currentUser.value.id)
    ElMessage.success('移除成功')
    currentUserRoles.value = []
    // 刷新用户列表（更新角色显示）
    loadTenantUsers()
  } catch (e) {
    // 用户取消或操作失败
  }
}

// 批量删除用户
async function handleBatchDeleteUsers() {
  if (!currentTenant.value || userSelectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${userSelectedIds.value.length} 个用户吗？`, '提示', {
      type: 'warning'
    })
    await batchDeleteTenantUsers(currentTenant.value.id, userSelectedIds.value)
    ElMessage.success('批量删除成功')
    userSelectedIds.value = []
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 批量启用用户
async function handleBatchEnableUsers() {
  if (!currentTenant.value || userSelectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要启用选中的 ${userSelectedIds.value.length} 个用户吗？`, '提示', {
      type: 'warning'
    })
    await batchUpdateTenantUserStatus(currentTenant.value.id, userSelectedIds.value, 1)
    ElMessage.success('批量启用成功')
    userSelectedIds.value = []
    loadTenantUsers()
  } catch (e) {
    // 用户取消
  }
}

// 批量禁用用户
async function handleBatchDisableUsers() {
  if (!currentTenant.value || userSelectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要禁用选中的 ${userSelectedIds.value.length} 个用户吗？`, '提示', {
      type: 'warning'
    })
    await batchUpdateTenantUserStatus(currentTenant.value.id, userSelectedIds.value, 0)
    ElMessage.success('批量禁用成功')
    userSelectedIds.value = []
    loadTenantUsers()
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
    if (!resetPwdUser.value?.id || !currentTenant.value?.id) return

    resetPwdSaving.value = true
    try {
      await resetTenantUserPassword(currentTenant.value.id, resetPwdUser.value.id, {
        new_password: resetPwdForm.new_password,
        confirm_password: resetPwdForm.confirm_password
      })
      ElMessage.success('密码重置成功')
      closeResetPwdDialog()
    } catch (e) {
      // 错误已在拦截器处理
    } finally {
      resetPwdSaving.value = false
    }
  })
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

onMounted(() => {
  loadList()
})
</script>

<style scoped lang="scss">
.page {
  padding: 20px;
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.batch-bar {
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.flex-1 {
  flex: 1;
}
</style>