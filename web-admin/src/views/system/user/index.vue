<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="query.keyword"
          :placeholder="t('userPage.searchPlaceholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select
          v-model="query.status"
          :placeholder="t('userPage.colStatus')"
          clearable
          style="width: 120px"
        >
          <el-option :label="t('common.status.enabled')" :value="1" />
          <el-option :label="t('common.status.disabled')" :value="0" />
        </el-select>
        <template v-if="!isRecycleBin && !userStore.isAdmin">
          <el-select v-model="query.role" :placeholder="t('userPage.colRole')" clearable filterable style="width: 140px">
            <el-option v-for="r in roleFilterOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
          <el-select v-model="query.sort" :placeholder="t('userPage.sortPlaceholder')" clearable style="width: 130px">
            <el-option :label="t('userPage.sort.created_at')" value="created_at" />
            <el-option :label="t('auth.username')" value="username" />
            <el-option :label="t('userPage.colStatus')" value="status" />
            <el-option :label="t('userPage.sort.last_login_at')" value="last_login_at" />
          </el-select>
          <el-select v-model="query.order" :placeholder="t('userPage.orderPlaceholder')" clearable style="width: 100px">
            <el-option :label="t('userPage.order.desc')" value="desc" />
            <el-option :label="t('userPage.order.asc')" value="asc" />
          </el-select>
          <el-date-picker
            v-model="createdRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            :range-separator="t('userPage.dateTo')"
            :start-placeholder="t('userPage.dateStart')"
            :end-placeholder="t('userPage.dateEnd')"
            clearable
            style="width: 240px"
          />
        </template>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('userPage.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="info" @click="toggleRecycleBin">
          <el-icon><Delete /></el-icon>{{ isRecycleBin ? t('userPage.backToList') : t('userPage.recycleBin') }}
        </el-button>
        <el-button v-if="userStore.hasPermission('user:create') && !isRecycleBin" type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>{{ t('userPage.createUser') }}
        </el-button>
        <el-button
          v-if="userStore.hasPermission('user:create') && !isRecycleBin"
          type="primary"
          plain
          @click="openImportDialog"
        >
          <el-icon><Upload /></el-icon>{{ t('userPage.batchImport') }}
        </el-button>
      </div>
      <!-- 批量操作栏：勾选后才出现，系统管理员按租户分组提交 -->
      <div v-if="batchEnabled && selectedRows.length" class="batch-bar">
        <span class="batch-count">{{ t('userPage.selectedCount', selectedRows.length) }}</span>
        <el-button
          v-if="canBatchStatus"
          size="small"
          type="success"
          :loading="batchLoading"
          @click="handleBatchStatus(1)"
        >{{ t('userPage.batchEnable') }}</el-button>
        <el-button
          v-if="canBatchStatus"
          size="small"
          type="warning"
          :loading="batchLoading"
          @click="handleBatchStatus(0)"
        >{{ t('userPage.batchDisable') }}</el-button>
        <el-button
          v-if="canBatchAssign"
          size="small"
          type="primary"
          :loading="batchLoading"
          @click="openBatchAssignRoles"
        >{{ t('userPage.assignRoles') }}</el-button>
        <el-button
          v-if="canBatchExport"
          size="small"
          type="info"
          :loading="batchLoading"
          @click="handleExportSelected"
        >{{ t('userPage.exportSelected') }}</el-button>
        <el-button
          v-if="canBatchDelete"
          size="small"
          type="danger"
          :loading="batchLoading"
          @click="handleBatchDelete"
        >{{ t('userPage.batchDelete') }}</el-button>
        <el-button size="small" link @click="clearSelection">{{ t('userPage.clearSelection') }}</el-button>
      </div>
      <!-- 列表 -->
      <el-table
        ref="tableRef"
        v-loading="loading"
        :data="list"
        border
        stripe
        style="width: 100%"
        @selection-change="onSelectionChange"
      >
        <el-table-column v-if="batchEnabled" type="selection" width="48" :selectable="isRowSelectable" />
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="username" :label="t('auth.username')" min-width="140" />
        <el-table-column prop="nickname" :label="t('register.nickname')" min-width="120" />
        <el-table-column prop="email" :label="t('common.email')" min-width="180" />
        <el-table-column prop="department" :label="t('userPage.colDepartment')" min-width="120">
          <template #default="{ row }">
            <span v-if="row.department">{{ row.department }}</span>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column v-if="userStore.isAdmin" prop="tenant_id" :label="t('userPage.colTenantId')" min-width="180">
          <template #default="{ row }">
            <el-tag v-if="row.tenant_id" size="small" type="info">{{ row.tenant_id }}</el-tag>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('userPage.colStatus')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? t('common.status.enabled') : t('common.status.disabled') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('userPage.colMaster')" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.is_master" type="danger" size="small">MASTER</el-tag>
            <span v-else style="color: #9ca3af">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="last_login_at" :label="t('userPage.colLastLogin')" width="180">
          <template #default="{ row }">
            <span v-if="row.last_login_at">{{ row.last_login_at }}</span>
            <span v-else style="color: #9ca3af">{{ t('userPage.neverLoggedIn') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('userPage.colCreatedAt')" width="180" />
        <el-table-column v-if="isRecycleBin" prop="deleted_at" :label="t('userPage.colDeletedAt')" width="180" />
        <el-table-column :label="t('userPage.colAction')" width="280" fixed="right">
          <template #default="{ row }">
            <template v-if="isRecycleBin">
              <el-button
                link
                type="success"
                @click="handleRestore(row)"
              >{{ t('userPage.restore') }}</el-button>
              <el-button
                v-if="!row.is_master"
                link
                type="danger"
                @click="handlePermanentDelete(row)"
              >{{ t('userPage.permanentDelete') }}</el-button>
            </template>
            <template v-else>
              <el-button
                v-if="userStore.hasPermission('user:update')"
                link
                type="primary"
                @click="openEditDialog(row)"
              >{{ t('common.edit') }}</el-button>
              <el-button
                v-if="userStore.hasPermission('user:reset_password')"
                link
                type="warning"
                @click="openResetPasswordDialog(row)"
              >{{ t('auth.resetPassword') }}</el-button>
              <el-button
                link
                type="info"
                @click="openViewPermissions(row)"
              >{{ t('userPage.viewPerms') }}</el-button>
              <el-button
                link
                type="primary"
                @click="openLoginLogs(row)"
              >{{ t('userPage.loginLogs') }}</el-button>
              <el-button
                v-if="userStore.hasPermission('rbac:user_role:assign') || userStore.isAdmin"
                link
                type="success"
                @click="openAssignRoles(row)"
              >{{ t('userPage.assignRoles') }}</el-button>
              <el-button
                v-if="userStore.hasPermission('user:update')"
                link
                type="warning"
                @click="toggleStatus(row)"
              >{{ row.status === 1 ? t('common.status.disabled') : t('common.status.enabled') }}</el-button>
              <el-button
                v-if="userStore.hasPermission('user:delete') && !row.is_master"
                link
                type="danger"
                @click="handleDelete(row)"
              >{{ t('common.delete') }}</el-button>
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
      :title="t('userPage.permsTitle', currentUser?.username || '')"
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-loading="permsLoading">
        <!-- 角色列表 -->
        <div class="section" style="margin-bottom: 20px;">
          <h4 style="margin-bottom: 12px; color: #606266;">{{ t('userPage.rolesBelong') }}</h4>
          <el-tag
            v-for="role in userRoles"
            :key="role.id"
            type="primary"
            style="margin-right: 8px; margin-bottom: 8px;"
          >
            {{ role.name }}
          </el-tag>
          <el-empty v-if="userRoles.length === 0" :description="t('userPage.noRoles')" :image-size="60" />
        </div>

        <!-- 权限列表 -->
        <div class="section">
          <h4 style="margin-bottom: 12px; color: #606266;">{{ t('userPage.permList') }}</h4>
          <el-table :data="userPermissions" border stripe size="small">
            <el-table-column prop="name" :label="t('userPage.colPermName')" min-width="150" />
            <el-table-column prop="code" :label="t('userPage.colPermCode')" min-width="180" />
            <el-table-column prop="resource" :label="t('userPage.colResource')" width="100" />
            <el-table-column prop="action" :label="t('userPage.colAction')" width="100" />
          </el-table>
          <el-empty v-if="userPermissions.length === 0" :description="t('userPage.noPerms')" :image-size="60" />
        </div>
      </div>
    </el-dialog>

    <!-- 分配角色弹窗 -->
    <el-dialog
      v-model="assignRolesDialogVisible"
      :title="t('userPage.assignRolesTitle', currentUser?.username || '')"
      width="500px"
      :close-on-click-modal="false"
    >
      <div v-loading="assignRolesLoading">
        <div style="margin-bottom: 10px; font-size: 14px; color: #6b7280">
          {{ t('userPage.selectedRolesPrefix') }} <b style="color: #2563eb">{{ selectedRoleIds.length }}</b> {{ t('userPage.roleUnit') }}
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
        <el-empty v-if="availableRoles.length === 0" :description="t('userPage.noAssignableRoles')" :image-size="60" />
      </div>
      <template #footer>
        <el-button @click="assignRolesDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="savingRoles" @click="submitAssignRoles">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 批量分配角色弹窗：提交后覆盖所选用户的全部角色 -->
    <el-dialog
      v-model="batchAssignVisible"
      :title="t('userPage.batchAssignTitle', selectedRows.length)"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        :title="t('userPage.batchAssignWarn')"
        style="margin-bottom: 12px;"
      />
      <div v-loading="batchAssignLoading">
        <div style="margin-bottom: 10px; font-size: 14px; color: #6b7280">
          {{ t('userPage.selectedRolesPrefix') }} <b style="color: #2563eb">{{ batchAssignRoleIds.length }}</b> {{ t('userPage.roleUnit') }}
        </div>
        <el-checkbox-group v-model="batchAssignRoleIds">
          <el-checkbox
            v-for="role in batchAssignRoles"
            :key="role.id"
            :label="role.id"
            style="display: block; margin-bottom: 8px;"
          >
            {{ role.name }}
            <el-tag size="small" type="info" style="margin-left: 8px;">{{ role.code }}</el-tag>
          </el-checkbox>
        </el-checkbox-group>
        <el-empty v-if="batchAssignRoles.length === 0" :description="t('userPage.noAssignableRoles')" :image-size="60" />
      </div>
      <template #footer>
        <el-button @click="batchAssignVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="batchAssignSaving" @click="submitBatchAssignRoles">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? t('userPage.createUser') : t('userPage.editUser')"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="90px"
      >
        <el-form-item v-if="dialogMode === 'create' && userStore.isAdmin" :label="t('userPage.targetTenant')" required>
          <el-select
            v-model="createTenantId"
            filterable
            clearable
            :loading="createTenantLoading"
            :placeholder="t('userPage.targetTenantPlaceholder')"
            style="width: 100%"
          >
            <el-option v-for="tenant in createTenantOptions" :key="tenant.id" :label="tenant.name" :value="tenant.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('auth.username')" prop="username">
          <el-input v-model="form.username" :disabled="dialogMode === 'edit'" :placeholder="t('userPage.usernamePlaceholder')" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" :label="t('auth.password')" prop="password">
          <el-input v-model="form.password" type="password" show-password :placeholder="t('userPage.passwordPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('register.nickname')" prop="nickname">
          <el-input v-model="form.nickname" :placeholder="t('userPage.nicknamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('common.email')" prop="email">
          <el-input v-model="form.email" :placeholder="t('register.emailPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('userPage.colDepartment')" prop="department">
          <el-input v-model="form.department" :placeholder="t('userPage.departmentPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item :label="t('userPage.positionLabel')" prop="position">
          <el-input v-model="form.position" :placeholder="t('userPage.positionPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" :label="t('userPage.colRole')" prop="role_ids">
          <!-- 后端创建接口的 role_ids 为必填（min=1），不选会直接 400 -->
          <el-select
            v-model="createRoleIds"
            multiple
            filterable
            clearable
            :loading="createRoleLoading"
            :placeholder="t('userPage.roleRequiredPlaceholder')"
            style="width: 100%"
          >
            <el-option v-for="r in createRoleOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" :label="t('userPage.remarkLabel')" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="2" :placeholder="t('userPage.remarkPlaceholder')" maxlength="255" />
        </el-form-item>
        <el-form-item :label="t('userPage.colStatus')">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">{{ t('common.status.enabled') }}</el-radio>
            <el-radio :value="0">{{ t('common.status.disabled') }}</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码弹窗 -->
    <el-dialog
      v-model="resetPwdDialogVisible"
      :title="t('auth.resetPassword')"
      width="400px"
      :close-on-click-modal="false"
      @close="closeResetPwdDialog"
    >
      <p style="margin-bottom: 16px;">{{ t('userPage.resetPwdTip', resetPwdUser?.username || '') }}</p>
      <el-form
        ref="resetPwdFormRef"
        :model="resetPwdForm"
        :rules="resetPwdRules"
        label-width="100px"
      >
        <el-form-item :label="t('auth.newPassword')" prop="new_password">
          <el-input
            v-model="resetPwdForm.new_password"
            type="password"
            :placeholder="t('userPage.newPwdPlaceholder')"
            show-password
          />
        </el-form-item>
        <el-form-item :label="t('auth.confirmPassword')" prop="confirm_password">
          <el-input
            v-model="resetPwdForm.confirm_password"
            type="password"
            :placeholder="t('userPage.confirmPwdPlaceholder')"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeResetPwdDialog">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="resetPwdSaving" @click="submitResetPassword">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 登录日志弹窗 -->
    <el-dialog
      v-model="loginLogsDialogVisible"
      :title="t('userPage.loginLogs') + ' - ' + (loginLogsUser?.username || '')"
      width="900px"
      :close-on-click-modal="false"
    >
      <el-table v-loading="loginLogsLoading" :data="loginLogsList" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (loginLogsPage - 1) * loginLogsPageSize + i + 1" />
        <el-table-column prop="action" :label="t('userPage.logsColAction')" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.action === 'login' ? 'success' : 'info'">
              {{ row.action === 'login' ? t('userPage.actionLogin') : t('userPage.actionLogout') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="result" :label="t('userPage.logsColResult')" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'success' ? 'success' : 'danger'">
              {{ row.result === 'success' ? t('userPage.resultSuccess') : t('userPage.resultFail') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" :label="t('userPage.logsColIp')" width="140" />
        <el-table-column prop="user_agent" :label="t('userPage.logsColUA')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="created_at" :label="t('userPage.logsColTime')" width="180" />
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
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules, TableInstance } from 'element-plus'
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
  batchUpdateUserStatus,
  batchDeleteUsers,
  batchUpdateTenantUserStatus,
  batchDeleteTenantUsers,
  restoreTenantUser,
  permanentDeleteTenantUser,
  restoreUser,
  permanentDeleteUser,
  type UserItem,
  type UserCreateParams,
  type UserUpdateParams,
  type UserListParams
} from '@/api/modules/user'
import { getUserRoles, getRolePermissions, type RoleItem, getRoleListForSelect, assignUserRoles, removeAllUserRoles, batchAssignUserRoles } from '@/api/modules/role'
import { getTenantList, type TenantItem } from '@/api/modules/tenant'
import { type PermissionItem } from '@/api/modules/permission'
import { getAuditLogList, type AuditLogItem } from '@/api/modules/audit'
import { createExport } from '@/api/modules/export'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'
import UserImportDialog from './components/UserImportDialog.vue'

const { t } = useI18n()
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

// ---------- 批量操作 ----------
const tableRef = ref<TableInstance>()
const selectedRows = ref<UserItem[]>([])
const batchLoading = ref(false)
// 批量分配角色弹窗状态
const batchAssignVisible = ref(false)
const batchAssignLoading = ref(false)
const batchAssignSaving = ref(false)
const batchAssignRoles = ref<RoleItem[]>([])
const batchAssignRoleIds = ref<string[]>([])
const canBatchStatus = computed(() => userStore.hasPermission('user:batch_status'))
const canBatchDelete = computed(() => userStore.hasPermission('user:batch_delete'))
// 批量分配角色与单行分配角色同门槛，系统管理员默认拥有
const canBatchAssign = computed(() => userStore.hasPermission('rbac:user_role:batch_assign') || userStore.isAdmin)
// 导出所选走异步导出任务，需要 export:create 权限
const canBatchExport = computed(() => userStore.hasPermission('export:create') || userStore.isAdmin)
// 回收站里的行不支持批量操作（批量接口只面向在用用户）
const batchEnabled = computed(
  () => !isRecycleBin.value && (canBatchStatus.value || canBatchDelete.value || canBatchAssign.value || canBatchExport.value)
)

// 系统管理员的批量接口把租户ID放在路径上，缺租户ID的行无法提交
function isRowSelectable(row: UserItem) {
  return !!row.id && (!userStore.isAdmin || !!row.tenant_id)
}

function onSelectionChange(rows: UserItem[]) {
  selectedRows.value = rows
}

function clearSelection() {
  selectedRows.value = []
  tableRef.value?.clearSelection()
}

// 翻页/刷新后列表已换血，残留的勾选会指向旧行
watch(list, () => clearSelection())

// 把勾选行按 tenant_id 分组，供系统管理员逐个租户提交
function groupIdsByTenant(rows: UserItem[]): Map<string, string[]> {
  const grouped = new Map<string, string[]>()
  for (const row of rows) {
    if (!row.id || !row.tenant_id) continue
    const ids = grouped.get(row.tenant_id) ?? []
    ids.push(row.id)
    grouped.set(row.tenant_id, ids)
  }
  return grouped
}

async function handleBatchStatus(status: number) {
  const rows = selectedRows.value.filter((row) => row.id)
  if (!rows.length) return
  const action = status === 1 ? t('common.status.enabled') : t('common.status.disabled')
  try {
    await ElMessageBox.confirm(t('userPage.batchActionConfirm', action, rows.length), t('common.tip'), { type: 'warning' })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    if (userStore.isAdmin) {
      const grouped = groupIdsByTenant(rows)
      if (!grouped.size) {
        ElMessage.warning(t('userPage.noTenantForBatch'))
        return
      }
      // 逐个租户提交：某个租户失败会中断后续请求，已提交的租户不会回滚
      for (const [tenantId, ids] of grouped) {
        await batchUpdateTenantUserStatus(tenantId, ids, status)
      }
    } else {
      await batchUpdateUserStatus(rows.map((row) => row.id), status)
    }
    ElMessage.success(t('userPage.batchActionDone', action, rows.length))
    clearSelection()
    reload()
  } finally {
    batchLoading.value = false
  }
}

async function handleBatchDelete() {
  const rows = selectedRows.value.filter((row) => row.id)
  if (!rows.length) return
  try {
    await ElMessageBox.confirm(t('userPage.batchDeleteConfirm', rows.length), t('userPage.dangerOp'), {
      type: 'error',
      confirmButtonText: t('userPage.confirmDelete'),
      cancelButtonText: t('common.cancel')
    })
  } catch {
    return
  }
  batchLoading.value = true
  try {
    if (userStore.isAdmin) {
      const grouped = groupIdsByTenant(rows)
      if (!grouped.size) {
        ElMessage.warning(t('userPage.noTenantForBatch'))
        return
      }
      for (const [tenantId, ids] of grouped) {
        await batchDeleteTenantUsers(tenantId, ids)
      }
    } else {
      await batchDeleteUsers(rows.map((row) => row.id))
    }
    ElMessage.success(t('userPage.batchDeleted', rows.length))
    clearSelection()
    reload()
  } finally {
    batchLoading.value = false
  }
}

// 打开批量分配角色弹窗：角色候选与单行分配保持一致
async function openBatchAssignRoles() {
  if (!selectedRows.value.length) return
  batchAssignVisible.value = true
  batchAssignRoleIds.value = []
  batchAssignLoading.value = true
  try {
    const scope = userStore.isAdmin ? 'all' : 'tenant'
    batchAssignRoles.value = (await getRoleListForSelect(scope)) || []
  } catch (e) {
    ElMessage.error(t('userPage.loadRolesFailed'))
  } finally {
    batchAssignLoading.value = false
  }
}

// 提交批量分配角色：每个用户的角色列表都是覆盖式更新，因此至少要选一个角色
async function submitBatchAssignRoles() {
  const rows = selectedRows.value.filter((row) => row.id)
  if (!rows.length) return
  if (!batchAssignRoleIds.value.length) {
    ElMessage.warning(t('userPage.selectAtLeastOneRole'))
    return
  }
  batchAssignSaving.value = true
  try {
    await batchAssignUserRoles(
      rows.map((row) => ({ user_id: row.id as string, role_ids: [...batchAssignRoleIds.value] }))
    )
    ElMessage.success(t('userPage.batchAssignedRoles', rows.length))
    batchAssignVisible.value = false
    clearSelection()
    reload()
  } catch (e) {
    ElMessage.error((e as Error).message || t('userPage.batchAssignFailed'))
  } finally {
    batchAssignSaving.value = false
  }
}

// 导出所选用户：创建异步导出任务，完成后在下载中心下载。
// 任务按租户归属落库，因此系统管理员一次只能导出一个租户的用户。
async function handleExportSelected() {
  const rows = selectedRows.value.filter((row) => row.id)
  if (!rows.length) return
  const tenantIds = new Set(rows.map((row) => row.tenant_id || ''))
  if (userStore.isAdmin && tenantIds.size > 1) {
    ElMessage.warning(t('userPage.multiTenantExport'))
    return
  }
  batchLoading.value = true
  try {
    await createExport({
      resource: 'user',
      format: 'csv',
      ...(userStore.isAdmin && rows[0].tenant_id ? { tenant_id: rows[0].tenant_id } : {}),
      filters: { ids: rows.map((row) => row.id as string) }
    })
    ElMessage.success(t('userPage.exportTaskCreated', rows.length))
    clearSelection()
  } catch (e) {
    ElMessage.error((e as Error).message || t('userPage.exportTaskFailed'))
  } finally {
    batchLoading.value = false
  }
}

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
    console.error(t('userPage.loadRoleFilterFailed'), e)
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

const rules = computed<FormRules>(() => ({
  username: [{ required: true, message: t('userPage.usernameRequired'), trigger: 'blur' }],
  password: [{ required: true, min: 6, max: 32, message: t('userPage.pwdLength'), trigger: 'blur' }],
  email: [{ type: 'email', message: t('userPage.emailInvalid'), trigger: 'blur' }]
}))

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
const resetPwdRules = computed<FormRules>(() => ({
  new_password: [{ required: true, min: 6, max: 32, message: t('userPage.pwdLength'), trigger: 'blur' }],
  confirm_password: [
    { required: true, message: t('userPage.pwdConfirmRequired'), trigger: 'blur' },
    {
      validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
        if (value !== resetPwdForm.new_password) {
          callback(new Error(t('userPage.pwdMismatch')))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}))

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
      ElMessage.warning(t('userPage.selectTargetTenantFirst'))
      return
    }
    if (dialogMode.value === 'create' && !createRoleIds.value.length) {
      ElMessage.warning(t('userPage.selectAtLeastOneRole'))
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
        ElMessage.success(t('userPage.createSuccess'))
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
        ElMessage.success(t('userPage.updateSuccess'))
      }
      dialogVisible.value = false
      loadList()
    } finally {
      saving.value = false
    }
  })
}

function toggleStatus(row: UserItem) {
  ElMessageBox.confirm(t('userPage.toggleStatusConfirm', row.status === 1 ? t('common.status.disabled') : t('common.status.enabled')), t('common.tip'), {
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
      ElMessage.success(t('userPage.opSuccess'))
      loadList()
    })
    .catch(() => {})
}

function handleDelete(row: UserItem) {
  ElMessageBox.confirm(t('userPage.deleteConfirm', row.username), t('userPage.dangerOp'), {
    type: 'error',
    confirmButtonText: t('userPage.confirmDelete'),
    cancelButtonText: t('common.cancel')
  })
    .then(async () => {
      if (!row.id) return
      await deleteUser(row.id)
      ElMessage.success(t('userPage.deleteSuccess'))
      loadList()
    })
    .catch(() => {})
}

// 恢复用户
async function handleRestore(row: UserItem) {
  if (!row.id) return
  try {
    await ElMessageBox.confirm(t('userPage.restoreConfirm', row.username), t('common.tip'), {
      type: 'warning'
    })
    // 系统管理员使用跨租户接口，普通租户管理员使用当前租户接口
    if (userStore.isAdmin && row.tenant_id) {
      await restoreTenantUser(row.tenant_id, row.id)
    } else {
      await restoreUser(row.id)
    }
    ElMessage.success(t('userPage.restoreSuccess'))
    loadList()
  } catch (e) {
    // 用户取消
  }
}

// 永久删除用户
async function handlePermanentDelete(row: UserItem) {
  if (!row.id) return
  try {
    await ElMessageBox.confirm(t('userPage.permDeleteConfirm', row.username), t('userPage.dangerOp'), {
      type: 'error',
      confirmButtonText: t('userPage.confirmPermDelete'),
      cancelButtonText: t('common.cancel')
    })
    // 系统管理员使用跨租户接口，普通租户管理员使用当前租户接口
    if (userStore.isAdmin && row.tenant_id) {
      await permanentDeleteTenantUser(row.tenant_id, row.id)
    } else {
      await permanentDeleteUser(row.id)
    }
    ElMessage.success(t('userPage.permDeleteSuccess'))
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
      ElMessage.success(t('userPage.resetPwdSuccess'))
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
    ElMessage.error(t('userPage.getPermsFailed'))
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
    ElMessage.error(t('userPage.loadRoleDataFailed'))
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
    
    ElMessage.success(t('userPage.assignRoleSuccess'))
    assignRolesDialogVisible.value = false
  } catch (e) {
    ElMessage.error((e as Error).message || t('userPage.assignRoleFailed'))
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
      action: 'login'  // only query login-related logs
    })
    loginLogsList.value = res.items
    loginLogsTotal.value = res.total
  } catch (e) {
    ElMessage.error(t('userPage.loadLogsFailed'))
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
.batch-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}
.batch-count {
  font-size: 13px;
  color: #606266;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
</style>