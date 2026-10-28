<template>
  <div class="page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <div class="search-left">
          <el-input
            v-model="query.keyword"
            :placeholder="t('masterAdminPage.searchPlaceholder')"
            clearable
            style="width: 220px"
            @keyup.enter="loadList"
          />
          <el-select
            v-model="query.status"
            :placeholder="t('masterAdminPage.statusFilter')"
            clearable
            style="width: 120px"
          >
            <el-option :label="t('masterAdminPage.status.enabled')" :value="1" />
            <el-option :label="t('masterAdminPage.status.disabled')" :value="0" />
          </el-select>
          <el-button type="primary" @click="loadList">
            <el-icon><Search /></el-icon>{{ t('common.query') }}
          </el-button>
          <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        </div>
        <div class="search-right">
          <el-button type="info" @click="goToRecycle">
            <el-icon><DeleteFilled /></el-icon>{{ t('masterAdminPage.recycleBin') }}
          </el-button>
          <el-button type="success" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>{{ t('masterAdminPage.create') }}
          </el-button>
        </div>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length > 0" class="batch-bar">
        <el-alert
          :title="t('masterAdminPage.selected', selectedIds.length)"
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <el-button link type="primary" @click="handleBatchEnable">{{ t('masterAdminPage.batchEnable') }}</el-button>
            <el-button link type="warning" @click="handleBatchDisable">{{ t('masterAdminPage.batchDisable') }}</el-button>
            <el-button link type="danger" @click="handleBatchDelete">{{ t('masterAdminPage.batchDelete') }}</el-button>
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
        <el-table-column :label="t('masterAdminPage.col.username')" prop="username" min-width="120" />
        <el-table-column :label="t('masterAdminPage.col.nickname')" prop="nickname" min-width="120" />
        <el-table-column :label="t('masterAdminPage.col.email')" prop="email" min-width="180" />
        <el-table-column :label="t('masterAdminPage.col.role')" min-width="200">
          <template #default="{ row }">
            <div v-if="row.role_ids && row.role_ids.length > 0">
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
            <el-tag v-else type="info" size="small">{{ t('masterAdminPage.noRole') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('masterAdminPage.col.status')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? t('masterAdminPage.status.enabled') : t('masterAdminPage.status.disabled') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('masterAdminPage.col.lastLogin')" prop="last_login_at" min-width="160">
          <template #default="{ row }">
            <span v-if="row.last_login_at">{{ row.last_login_at }}</span>
            <span v-else style="color: #9ca3af">{{ t('masterAdminPage.notLogin') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('masterAdminPage.col.createdAt')" prop="created_at" min-width="160" />
        <el-table-column :label="t('masterAdminPage.col.operation')" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">{{ t('masterAdminPage.action.edit') }}</el-button>
            <el-button link type="warning" @click="toggleStatus(row)">
              {{ row.status === 1 ? t('masterAdminPage.status.disabled') : t('masterAdminPage.status.enabled') }}
            </el-button>
            <el-button
              v-if="!isProtectedUser(row)"
              link
              type="danger"
              @click="handleDelete(row)"
            >
              {{ t('masterAdminPage.action.delete') }}
            </el-button>
            <el-tag v-else type="info" size="small">{{ t('masterAdminPage.protectedTag') }}</el-tag>
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
      :title="dialogMode === 'create' ? t('masterAdminPage.dialog.createTitle') : t('masterAdminPage.dialog.editTitle')"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
      >
        <el-form-item :label="t('masterAdminPage.form.usernameLabel')" prop="username">
          <el-input
            v-model="form.username"
            :placeholder="t('masterAdminPage.form.usernamePlaceholder')"
            :disabled="dialogMode === 'edit'"
          />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'create'" :label="t('masterAdminPage.form.passwordLabel')" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            :placeholder="t('masterAdminPage.form.passwordPlaceholder')"
            show-password
          />
        </el-form-item>
        <el-form-item v-else :label="t('masterAdminPage.form.passwordLabel')" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            :placeholder="t('masterAdminPage.form.passwordKeepPlaceholder')"
            show-password
          />
        </el-form-item>
        <el-form-item :label="t('masterAdminPage.form.nicknameLabel')" prop="nickname">
          <el-input v-model="form.nickname" :placeholder="t('masterAdminPage.form.nicknamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('masterAdminPage.form.emailLabel')" prop="email">
          <el-input v-model="form.email" :placeholder="t('masterAdminPage.form.emailPlaceholder')" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" :label="t('masterAdminPage.form.departmentLabel')" prop="department">
          <el-input v-model="form.department" :placeholder="t('masterAdminPage.form.departmentPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" :label="t('masterAdminPage.form.positionLabel')" prop="position">
          <el-input v-model="form.position" :placeholder="t('masterAdminPage.form.positionPlaceholder')" maxlength="100" />
        </el-form-item>
        <el-form-item :label="t('masterAdminPage.form.remarkLabel')" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="2" :placeholder="t('masterAdminPage.form.remarkPlaceholder')" maxlength="255" />
        </el-form-item>
        <el-form-item :label="t('masterAdminPage.form.roleLabel')" prop="role_id">
          <el-select v-model="form.role_id" :placeholder="t('masterAdminPage.form.rolePlaceholder')" style="width: 100%">
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
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from '@/composables/useI18n'
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
} = useTableList<UserItem, { keyword: string; status?: number }>({
  fetchList: async (params) => {
    const req: { page?: number; page_size?: number; keyword?: string; status?: number } = { page: params.page, page_size: params.page_size }
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
  department: '',
  position: '',
  remark: '',
  role_id: ''
})

const rules = computed<FormRules>(() => ({
  username: [
    { required: true, message: t('masterAdminPage.form.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 50, message: t('masterAdminPage.form.usernameLength'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('masterAdminPage.form.passwordRequired'), trigger: 'blur', validator: (rule, value, callback) => {
      if (dialogMode.value === 'create' && !value) {
        callback(new Error(t('masterAdminPage.form.passwordRequired')))
      } else {
        callback()
      }
    }}
  ],
  nickname: [{ max: 50, message: t('masterAdminPage.form.nicknameMax'), trigger: 'blur' }],
  email: [{ type: 'email', message: t('masterAdminPage.form.emailInvalid'), trigger: 'blur' }],
  role_id: [{ required: true, message: t('masterAdminPage.form.roleRequired'), trigger: 'change' }]
}))

function resetSearch() {
  reset()
}

function handleSelectionChange(selection: UserItem[]) {
  selectedIds.value = selection.map(item => item.id)
}

async function loadRoles() {
  try {
    const res = await getSystemAdminRoles()
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
  form.department = ''
  form.position = ''
  form.remark = ''
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
  form.department = row.department || ''
  form.position = row.position || ''
  form.remark = row.remark || ''
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
          remark: form.remark,
          role_id: form.role_id
        }
        await createMasterAdmin(data)
        ElMessage.success(t('masterAdminPage.created'))
      } else {
        const data: MasterAdminUpdateParams = {
          nickname: form.nickname,
          email: form.email,
          department: form.department,
          position: form.position,
          remark: form.remark,
          role_id: form.role_id
        }
        if (form.password) {
          data.password = form.password
        }
        await updateMasterAdmin(editingId.value!, data)
        ElMessage.success(t('masterAdminPage.updated'))
      }
      dialogVisible.value = false
      loadList()
    } catch (e) {
      ElMessage.error((e as { response?: { data?: { message?: string } } })?.response?.data?.message || t('masterAdminPage.opFailed'))
    }
  })
}

function toggleStatus(row: UserItem) {
  const newStatus = row.status === 1 ? 0 : 1
  ElMessageBox.confirm(
    newStatus === 1 ? t('masterAdminPage.enableConfirm', row.username) : t('masterAdminPage.disableConfirm', row.username),
    t('masterAdminPage.confirmTitle'),
    {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    }
  )
    .then(async () => {
      await updateMasterAdminStatus(row.id, newStatus)
      ElMessage.success(newStatus === 1 ? t('masterAdminPage.enableSuccess') : t('masterAdminPage.disableSuccess'))
      loadList()
    })
    .catch(() => {})
}

function handleDelete(row: UserItem) {
  ElMessageBox.confirm(t('masterAdminPage.deleteConfirm', row.username), t('masterAdminPage.deleteTitle'), {
    type: 'error',
    confirmButtonText: t('masterAdminPage.confirmDeleteBtn'),
    cancelButtonText: t('common.cancel')
  })
    .then(async () => {
      await deleteMasterAdmin(row.id)
      ElMessage.success(t('masterAdminPage.deleted'))
      loadList()
    })
    .catch(() => {})
}

// 解绑角色
function handleUnbindRole(row: UserItem, roleId: string, roleName: string) {
  ElMessageBox.confirm(
    t('masterAdminPage.unbindRoleConfirm', row.username, roleName || roleId),
    t('masterAdminPage.unbindRoleTitle'),
    {
      type: 'warning',
      confirmButtonText: t('masterAdminPage.confirmUnbindBtn'),
      cancelButtonText: t('common.cancel')
    }
  )
    .then(async () => {
      await unbindUserRole(row.id, roleId)
      ElMessage.success(t('masterAdminPage.roleUnbindSuccess'))
      loadList()
    })
    .catch(() => {})
}

function handleBatchEnable() {
  ElMessageBox.confirm(t('masterAdminPage.batchEnableConfirm', selectedIds.value.length), t('masterAdminPage.batchEnable'), {
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'warning'
  })
    .then(async () => {
      await batchUpdateMasterAdminStatus(selectedIds.value, 1)
      ElMessage.success(t('masterAdminPage.batchEnabled'))
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDisable() {
  ElMessageBox.confirm(t('masterAdminPage.batchDisableConfirm', selectedIds.value.length), t('masterAdminPage.batchDisable'), {
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'warning'
  })
    .then(async () => {
      await batchUpdateMasterAdminStatus(selectedIds.value, 0)
      ElMessage.success(t('masterAdminPage.batchDisabled'))
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
    ElMessage.warning(t('masterAdminPage.batchDeleteProtectedWarn'))
    return
  }

  let confirmMessage = t('masterAdminPage.batchDeleteConfirm', deletableIds.length)
  if (protectedCount > 0) {
    confirmMessage += t('masterAdminPage.batchDeleteSkipSuffix', protectedCount)
  }

  ElMessageBox.confirm(confirmMessage, t('masterAdminPage.dangerTitle'), {
    type: 'error',
    confirmButtonText: t('masterAdminPage.confirmDeleteBtn'),
    cancelButtonText: t('common.cancel')
  })
    .then(async () => {
      await batchDeleteMasterAdmins(deletableIds)
      ElMessage.success(t('masterAdminPage.batchDeleted'))
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