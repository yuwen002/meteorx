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
        <el-button type="success" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>新增系统管理员
        </el-button>
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
        :data="list"
        border
        stripe
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column type="index" label="#" width="60" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="warning" @click="toggleStatus(row)">
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
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增系统管理员' : '编辑系统管理员'"
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
        <el-form-item v-else label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="不修改请留空" />
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
import {
  getMasterAdminList,
  createMasterAdmin,
  updateMasterAdmin,
  deleteMasterAdmin,
  updateMasterAdminStatus,
  batchUpdateMasterAdminStatus,
  batchDeleteMasterAdmins,
  type UserItem,
  type UserCreateParams,
  type UserUpdateParams
} from '@/api/modules/user'

const list = ref<UserItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const selectedIds = ref<string[]>([])

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

async function loadList() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (search.keyword) params.keyword = search.keyword
    if (search.status !== undefined && search.status !== null) params.status = search.status
    const res: any = await getMasterAdminList(params)
    list.value = res.list || []
    total.value = res.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function resetSearch() {
  search.keyword = ''
  search.status = undefined
  page.value = 1
  loadList()
}

function handleSelectionChange(selection: UserItem[]) {
  selectedIds.value = selection.map(item => item.id)
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
        await createMasterAdmin({
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
        await updateMasterAdmin(editingId.value, updateData)
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
  ElMessageBox.confirm(`确定要${row.status === 1 ? '禁用' : '启用'}该系统管理员吗？`, '提示', {
    type: 'warning'
  })
    .then(async () => {
      if (!row.id) return
      await updateMasterAdminStatus(row.id, row.status === 1 ? 0 : 1)
      ElMessage.success('操作成功')
      loadList()
    })
    .catch(() => {})
}

function handleDelete(row: UserItem) {
  ElMessageBox.confirm(`确定要删除系统管理员 "${row.username}" 吗？此操作不可恢复`, '危险操作', {
    type: 'error',
    confirmButtonText: '确定删除',
    cancelButtonText: '取消'
  })
    .then(async () => {
      if (!row.id) return
      await deleteMasterAdmin(row.id)
      ElMessage.success('删除成功')
      loadList()
    })
    .catch(() => {})
}

function handleBatchEnable() {
  handleBatchStatus(1)
}

function handleBatchDisable() {
  handleBatchStatus(0)
}

function handleBatchStatus(status: number) {
  const action = status === 1 ? '启用' : '禁用'
  ElMessageBox.confirm(`确定要${action}选中的 ${selectedIds.value.length} 个系统管理员吗？`, '提示', {
    type: 'warning'
  })
    .then(async () => {
      await batchUpdateMasterAdminStatus(selectedIds.value, status)
      ElMessage.success('批量操作成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
}

function handleBatchDelete() {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个系统管理员吗？此操作不可恢复`, '危险操作', {
    type: 'error',
    confirmButtonText: '确定删除',
    cancelButtonText: '取消'
  })
    .then(async () => {
      await batchDeleteMasterAdmins(selectedIds.value)
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      loadList()
    })
    .catch(() => {})
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
.batch-bar {
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