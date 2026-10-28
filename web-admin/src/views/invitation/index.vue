<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('nav.invitation') }}</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>{{ t('invitation.inviteNew') }}
          </el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-input v-model="keyword" :placeholder="t('invitation.searchPlaceholder')" clearable style="width: 200px; margin-right: 12px;" @clear="loadList" @keyup.enter="loadList" />
        <el-select v-model="statusFilter" :placeholder="t('invitation.col.status')" clearable style="width: 120px; margin-right: 12px;" @change="loadList">
          <el-option :label="t('invitation.status.pending')" value="pending" />
          <el-option :label="t('invitation.status.accepted')" value="accepted" />
          <el-option :label="t('invitation.status.cancelled')" value="cancelled" />
          <el-option :label="t('invitation.status.expired')" value="expired" />
        </el-select>
        <el-button type="primary" @click="loadList">{{ t('invitation.query') }}</el-button>
      </div>

      <el-table v-loading="loading" :data="list" stripe style="width: 100%;">
        <el-table-column prop="email" :label="t('invitation.col.email')" min-width="200" />
        <el-table-column :label="t('invitation.col.role')" min-width="150">
          <template #default="{ row }">
            <el-tag v-for="rid in row.role_ids" :key="rid" size="small" type="info" style="margin-right: 4px;">
              {{ getRoleName(rid) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('invitation.col.status')" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" :label="t('invitation.col.expiresAt')" min-width="160" />
        <el-table-column prop="accepted_at" :label="t('invitation.col.acceptedAt')" min-width="160">
          <template #default="{ row }">{{ row.accepted_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('invitation.col.createdAt')" min-width="160" />
        <el-table-column :label="t('invitation.col.actions')" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'pending'" type="primary" link size="small" @click="handleResend(row)">{{ t('invitation.resend') }}</el-button>
            <el-button v-if="row.status === 'pending'" type="warning" link size="small" @click="handleCancel(row)">{{ t('common.cancel') }}</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
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

    <el-dialog v-model="createDialogVisible" :title="t('invitation.inviteNew')" width="480px" :close-on-click-modal="false">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="80px">
        <el-form-item :label="t('common.email')" prop="emails">
          <el-select
            v-model="createForm.emails"
            multiple
            filterable
            allow-create
            default-first-option
            :reserve-query="false"
            :placeholder="t('invitation.emailsPlaceholder')"
            style="width: 100%;"
          />
        </el-form-item>
        <el-form-item :label="t('invitation.col.role')" prop="role_ids">
          <el-select v-model="createForm.role_ids" multiple :placeholder="t('invitation.rolePlaceholder')" style="width: 100%;">
            <el-option v-for="role in roleList" :key="role.id" :label="role.name" :value="role.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">{{ t('invitation.send') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getInvitationList, createInvitation, createInvitationsBatch, cancelInvitation, resendInvitation, deleteInvitation, getAssignableRoles, type InvitationItem, type RoleOption } from '@/api/modules/invitation'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

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
    console.error(t('invitation.roleLoadFailed'), e)
  }
}

function getRoleName(id: string) {
  const role = roleList.value.find(r => r.id === id)
  return role ? role.name : id
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    pending: t('invitation.status.pending'),
    accepted: t('invitation.status.accepted'),
    cancelled: t('invitation.status.cancelled'),
    expired: t('invitation.status.expired')
  }
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
    ElMessage.error(t('invitation.fetchListFailed'))
  } finally {
    loading.value = false
  }
}

const createDialogVisible = ref(false)
const createFormRef = ref<FormInstance>()
const createLoading = ref(false)
const createForm = reactive({
  emails: [] as string[],
  role_ids: [] as string[]
})
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const createRules = computed<FormRules>(() => ({
  emails: [
    {
      validator: (_rule, value: string[], callback) => {
        const arr = (value || []).map((e) => e.trim()).filter(Boolean)
        if (arr.length === 0) return callback(new Error(t('invitation.emailsRequired')))
        if (arr.length > 50) return callback(new Error(t('invitation.emailsMax')))
        const bad = arr.find((e) => !EMAIL_RE.test(e))
        if (bad) return callback(new Error(t('invitation.emailInvalid', bad)))
        callback()
      },
      trigger: 'change'
    }
  ],
  role_ids: [{ required: true, message: t('invitation.roleRequired'), trigger: 'change' }]
}))

function openCreateDialog() {
  createForm.emails = []
  createForm.role_ids = []
  createDialogVisible.value = true
}

async function submitCreate() {
  if (!createFormRef.value) return
  await createFormRef.value.validate(async (valid) => {
    if (!valid) return
    const emails = createForm.emails.map((e) => e.trim()).filter(Boolean)
    createLoading.value = true
    try {
      if (emails.length === 1) {
        await createInvitation({ email: emails[0], role_ids: createForm.role_ids })
        ElMessage.success(t('invitation.sent'))
      } else {
        const res = await createInvitationsBatch({ emails, role_ids: createForm.role_ids })
        const skipped = res?.skipped ?? []
        if (res && res.created > 0 && skipped.length === 0) {
          ElMessage.success(t('invitation.batchCreated', res.created))
        } else if (res && res.created > 0) {
          ElMessage.warning(t('invitation.batchPartial', res.created, skipped.length, skipped.map((s) => `${s.email}: ${s.reason}`).join('; ')))
        } else {
          ElMessage.error(t('invitation.batchAllSkipped'))
        }
      }
      createDialogVisible.value = false
      loadList()
    } catch (e) {
      ElMessage.error((e as Error).message || t('invitation.createFailed'))
    } finally {
      createLoading.value = false
    }
  })
}

async function handleResend(row: InvitationItem) {
  try {
    await resendInvitation(row.id)
    ElMessage.success(t('invitation.resendSuccess'))
  } catch (e) {
    ElMessage.error((e as Error).message || t('invitation.resendFailed'))
  }
}

async function handleCancel(row: InvitationItem) {
  try {
    await ElMessageBox.confirm(t('invitation.cancelConfirm'), t('common.tip'), { type: 'warning' })
    await cancelInvitation(row.id)
    ElMessage.success(t('invitation.canceled'))
    loadList()
  } catch { /* cancelled */ }
}

async function handleDelete(row: InvitationItem) {
  try {
    await ElMessageBox.confirm(t('invitation.deleteConfirm'), t('common.tip'), { type: 'warning' })
    await deleteInvitation(row.id)
    ElMessage.success(t('invitation.deleted'))
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