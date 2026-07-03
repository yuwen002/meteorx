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
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
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
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
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