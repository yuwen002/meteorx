<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>成员邀请</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>邀请新成员
          </el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-input v-model="keyword" placeholder="搜索邮箱" clearable style="width: 200px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option label="待处理" value="pending" />
          <el-option label="已接受" value="accepted" />
          <el-option label="已取消" value="cancelled" />
          <el-option label="已过期" value="expired" />
        </el-select>
        <el-button type="primary" @click="loadList">查询</el-button>
      </div>

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;">
        <el-table-column prop="email" label="邮箱" min-width="200" />
        <el-table-column label="角色" min-width="150">
          <template #default="{ row }">
            <el-tag v-for="rid in row.role_ids" :key="rid" size="small" type="info" style="margin-right: 4px;">
              {{ getRoleName(rid) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" min-width="160" />
        <el-table-column prop="accepted_at" label="接受时间" min-width="160">
          <template #default="{ row }">{{ row.accepted_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="160" />
        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'pending'" type="primary" link size="small" @click="handleResend(row)">重发</el-button>
            <el-button v-if="row.status === 'pending'" type="warning" link size="small" @click="handleCancel(row)">取消</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog v-model="createDialogVisible" title="邀请新成员" width="480px" :close-on-click-modal="false">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="80px">
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="createForm.email" placeholder="请输入被邀请人邮箱" />
        </el-form-item>
        <el-form-item label="角色" prop="role_ids">
          <el-select v-model="createForm.role_ids" multiple placeholder="请选择角色" style="width: 100%;">
            <el-option v-for="role in roleList" :key="role.id" :label="role.name" :value="role.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">发送邀请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getInvitationList, createInvitation, cancelInvitation, resendInvitation, deleteInvitation, getAssignableRoles, type InvitationItem, type RoleOption } from '@/api/modules/invitation'
import { toPageResult } from '@/types/pagination'

const loading = ref(false)
const list = ref<InvitationItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref('')

const roleList = ref<RoleOption[]>([])

async function loadRoles() {
  try {
    roleList.value = await getAssignableRoles() || []
  } catch (e) {
    console.error('加载角色失败', e)
  }
}

function getRoleName(id: string) {
  const role = roleList.value.find(r => r.id === id)
  return role ? role.name : id
}

function statusLabel(status: string) {
  const map: Record<string, string> = { pending: '待处理', accepted: '已接受', cancelled: '已取消', expired: '已过期' }
  return map[status] || status
}

function statusTagType(status: string) {
  const map: Record<string, string> = { pending: 'warning', accepted: 'success', cancelled: 'info', expired: 'danger' }
  return map[status] || 'info'
}

async function loadList() {
  loading.value = true
  try {
    const raw = await getInvitationList({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value,
      status: statusFilter.value || undefined
    })
    const result = toPageResult<InvitationItem>(raw)
    list.value = result.list
    total.value = result.total
  } catch (e) {
    ElMessage.error('获取邀请列表失败')
  } finally {
    loading.value = false
  }
}

const createDialogVisible = ref(false)
const createFormRef = ref<FormInstance>()
const createLoading = ref(false)
const createForm = reactive({
  email: '',
  role_ids: [] as string[]
})
const createRules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入正确的邮箱', trigger: 'blur' }],
  role_ids: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

function openCreateDialog() {
  createForm.email = ''
  createForm.role_ids = []
  createDialogVisible.value = true
}

async function submitCreate() {
  if (!createFormRef.value) return
  await createFormRef.value.validate(async (valid) => {
    if (!valid) return
    createLoading.value = true
    try {
      await createInvitation(createForm)
      ElMessage.success('邀请已发送')
      createDialogVisible.value = false
      loadList()
    } catch (e) {
      ElMessage.error((e as Error).message || '创建邀请失败')
    } finally {
      createLoading.value = false
    }
  })
}

async function handleResend(row: InvitationItem) {
  try {
    await resendInvitation(row.id)
    ElMessage.success('邀请邮件已重发')
  } catch (e) {
    ElMessage.error((e as Error).message || '重发失败')
  }
}

async function handleCancel(row: InvitationItem) {
  try {
    await ElMessageBox.confirm('确定取消此邀请？', '取消确认', { type: 'warning' })
    await cancelInvitation(row.id)
    ElMessage.success('邀请已取消')
    loadList()
  } catch { /* cancelled */ }
}

async function handleDelete(row: InvitationItem) {
  try {
    await ElMessageBox.confirm('确定删除此邀请记录？', '删除确认', { type: 'warning' })
    await deleteInvitation(row.id)
    ElMessage.success('已删除')
    loadList()
  } catch { /* cancelled */ }
}

onMounted(() => {
  loadRoles()
  loadList()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>