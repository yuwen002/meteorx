<template>
  <div class="page">
    <el-card shadow="never" class="table-card">
      <!-- 工具栏 -->
      <div class="toolbar">
        <el-input
          v-model="query.keyword"
          :placeholder="t('sysAnnouncementPage.search.placeholder')"
          clearable
          style="width: 220px"
          @keyup.enter="loadList"
        />
        <el-select v-model="query.status" :placeholder="t('sysAnnouncementPage.filter.status')" clearable style="width: 140px">
          <el-option :label="t('sysAnnouncementPage.status.draft')" :value="0" />
          <el-option :label="t('sysAnnouncementPage.status.published')" :value="1" />
          <el-option :label="t('sysAnnouncementPage.status.offline')" :value="2" />
        </el-select>
        <el-select v-model="query.scope" :placeholder="t('sysAnnouncementPage.filter.scope')" clearable style="width: 140px">
          <el-option :label="t('sysAnnouncementPage.scope.all')" value="all" />
          <el-option :label="t('sysAnnouncementPage.scope.tenant')" value="tenant" />
        </el-select>
        <el-button type="primary" @click="loadList">
          <el-icon><Search /></el-icon>{{ t('common.query') }}
        </el-button>
        <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
        <div class="flex-1"></div>
        <el-button type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>{{ t('sysAnnouncementPage.action.create') }}
        </el-button>
      </div>

      <!-- 列表 -->
      <el-table v-loading="loading" :data="list" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="55" :index="(i: number) => (page - 1) * pageSize + i + 1" />
        <el-table-column prop="title" :label="t('sysAnnouncementPage.col.title')" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="title-text">{{ row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="scope" :label="t('sysAnnouncementPage.col.scope')" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.scope === 'all' ? 'primary' : 'warning'">
              {{ row.scope === 'all' ? t('sysAnnouncementPage.scope.all') : t('sysAnnouncementPage.scope.tenant') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_tenant_id" :label="t('sysAnnouncementPage.col.targetTenant')" width="110">
          <template #default="{ row }">
            {{ row.target_tenant_id || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('sysAnnouncementPage.col.status')" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="getStatusType(row.status)">{{ row.status_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="publish_at" :label="t('sysAnnouncementPage.col.publishTime')" width="160">
          <template #default="{ row }">{{ row.publish_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="expire_at" :label="t('sysAnnouncementPage.col.expireTime')" width="160">
          <template #default="{ row }">{{ row.expire_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('sysAnnouncementPage.col.createdAt')" width="160" />
        <el-table-column :label="t('sysAnnouncementPage.col.operation')" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">{{ t('sysAnnouncementPage.row.detail') }}</el-button>
            <el-button link type="warning" @click="openEdit(row)">{{ t('sysAnnouncementPage.row.edit') }}</el-button>
            <el-button
              v-if="row.status !== 1"
              link
              type="success"
              @click="handleStatus(row, 1)"
            >{{ t('sysAnnouncementPage.row.publish') }}</el-button>
            <el-button
              v-else
              link
              type="warning"
              @click="handleStatus(row, 2)"
            >{{ t('sysAnnouncementPage.row.offline') }}</el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ t('sysAnnouncementPage.row.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 创建/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('sysAnnouncementPage.dialog.editTitle') : t('sysAnnouncementPage.dialog.createTitle')" width="680px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item :label="t('sysAnnouncementPage.form.title')" prop="title">
          <el-input v-model="form.title" :placeholder="t('sysAnnouncementPage.form.titlePlaceholder')" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('sysAnnouncementPage.form.content')" prop="content">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="6"
            :placeholder="t('sysAnnouncementPage.form.contentPlaceholder')"
            show-word-limit
            maxlength="5000"
          />
        </el-form-item>
        <el-form-item :label="t('sysAnnouncementPage.form.scope')" prop="scope">
          <el-radio-group v-model="form.scope">
            <el-radio-button value="all">{{ t('sysAnnouncementPage.scope.all') }}</el-radio-button>
            <el-radio-button value="tenant">{{ t('sysAnnouncementPage.scope.tenant') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.scope === 'tenant'" :label="t('sysAnnouncementPage.form.targetTenant')" prop="target_tenant_id">
          <el-input v-model="form.target_tenant_id" :placeholder="t('sysAnnouncementPage.form.targetTenantPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('sysAnnouncementPage.form.publishAt')">
          <el-date-picker
            v-model="form.publish_at"
            type="datetime"
            :placeholder="t('sysAnnouncementPage.form.publishAtPlaceholder')"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item :label="t('sysAnnouncementPage.form.expireAt')">
          <el-date-picker
            v-model="form.expire_at"
            type="datetime"
            :placeholder="t('sysAnnouncementPage.form.expireAtPlaceholder')"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item :label="t('sysAnnouncementPage.form.status')">
          <el-radio-group v-model="form.status">
            <el-radio-button :value="0">{{ t('sysAnnouncementPage.form.statusDraft') }}</el-radio-button>
            <el-radio-button :value="1">{{ t('sysAnnouncementPage.form.statusPublish') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" :title="t('sysAnnouncementPage.dialog.detailTitle')" width="680px" destroy-on-close>
      <el-descriptions v-if="current" :column="2" border>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.title')" :span="2">{{ current.title }}</el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.status')">
          <el-tag size="small" :type="getStatusType(current.status)">{{ current.status_text }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.scope')">
          {{ current.scope === 'all' ? t('sysAnnouncementPage.scope.all') : t('sysAnnouncementPage.scope.tenant') }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.targetTenant')" :span="2">{{ current.target_tenant_id || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.publishTime')">{{ current.publish_at || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.expireTime')">{{ current.expire_at || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.col.createdAt')" :span="2">{{ current.created_at }}</el-descriptions-item>
        <el-descriptions-item :label="t('sysAnnouncementPage.detail.content')" :span="2">
          <div class="content-box">{{ current.content }}</div>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import {
  getAnnouncementList,
  createAnnouncement,
  updateAnnouncement,
  updateAnnouncementStatus,
  deleteAnnouncement,
  getAnnouncementDetail,
  type AnnouncementItem
} from '@/api/modules/announcement'
import { useTableList } from '@/composables/useTableList'
import { toPageResult } from '@/types/pagination'
import { useI18n } from '@/composables/useI18n'

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
} = useTableList<AnnouncementItem, { keyword: string; status?: number; scope: string }>({
  fetchList: async (params) =>
    toPageResult(
      await getAnnouncementList({
        page: params.page,
        page_size: params.page_size,
        keyword: params.keyword || undefined,
        status: params.status,
        scope: params.scope || undefined
      })
    ),
  initialQuery: { keyword: '', status: undefined, scope: '' },
  defaultPageSize: 20
})
const loadList = reload

const dialogVisible = ref(false)
const detailVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const currentId = ref('')
const current = ref<AnnouncementItem | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({
  title: '',
  content: '',
  scope: 'all',
  target_tenant_id: '',
  status: 0,
  publish_at: '',
  expire_at: ''
})

const rules = computed<FormRules>(() => ({
  title: [{ required: true, message: t('sysAnnouncementPage.rules.titleRequired'), trigger: 'blur' }],
  content: [{ required: true, message: t('sysAnnouncementPage.rules.contentRequired'), trigger: 'blur' }],
  scope: [{ required: true, message: t('sysAnnouncementPage.rules.scopeRequired'), trigger: 'change' }],
  target_tenant_id: [
    {
      validator: (_: unknown, value: string, callback: (error?: string | Error) => void) => {
        if (form.scope === 'tenant' && !value) {
          callback(new Error(t('sysAnnouncementPage.rules.tenantIdRequired')))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}))

function resetSearch() {
  reset()
}

function openCreate() {
  isEdit.value = false
  currentId.value = ''
  form.title = ''
  form.content = ''
  form.scope = 'all'
  form.target_tenant_id = ''
  form.status = 0
  form.publish_at = ''
  form.expire_at = ''
  dialogVisible.value = true
}

async function openEdit(row: AnnouncementItem) {
  isEdit.value = true
  currentId.value = row.id
  try {
    const detail = await getAnnouncementDetail(row.id)
    form.title = detail.title
    form.content = detail.content
    form.scope = detail.scope
    form.target_tenant_id = detail.target_tenant_id
    form.status = detail.status
    form.publish_at = detail.publish_at || ''
    form.expire_at = detail.expire_at || ''
  } catch {
    // 回退到行数据
    form.title = row.title
    form.content = row.content
    form.scope = row.scope
    form.target_tenant_id = row.target_tenant_id
    form.status = row.status
    form.publish_at = row.publish_at || ''
    form.expire_at = row.expire_at || ''
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    const payload = {
      title: form.title,
      content: form.content,
      scope: form.scope,
      target_tenant_id: form.scope === 'tenant' ? form.target_tenant_id : '',
      status: form.status,
      publish_at: form.publish_at || undefined,
      expire_at: form.expire_at || undefined
    }
    if (isEdit.value) {
      await updateAnnouncement(currentId.value, payload)
      ElMessage.success(t('sysAnnouncementPage.update.success'))
    } else {
      await createAnnouncement(payload)
      ElMessage.success(t('sysAnnouncementPage.create.success'))
    }
    dialogVisible.value = false
    loadList()
  } catch (error) {
    console.error('failed to save announcement', error)
  } finally {
    submitting.value = false
  }
}

async function handleStatus(row: AnnouncementItem, status: number) {
  const publish = status === 1
  try {
    await ElMessageBox.confirm(
      publish
        ? t('sysAnnouncementPage.confirm.publish', row.title)
        : t('sysAnnouncementPage.confirm.offline', row.title),
      t('common.tip'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    await updateAnnouncementStatus(row.id, status)
    ElMessage.success(publish ? t('sysAnnouncementPage.publish.success') : t('sysAnnouncementPage.offline.success'))
    loadList()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(publish ? t('sysAnnouncementPage.publish.fail') : t('sysAnnouncementPage.offline.fail'))
    }
  }
}

async function handleDelete(row: AnnouncementItem) {
  try {
    await ElMessageBox.confirm(t('sysAnnouncementPage.confirm.delete', row.title), t('sysAnnouncementPage.confirm.warnTitle'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await deleteAnnouncement(row.id)
    ElMessage.success(t('sysAnnouncementPage.delete.success'))
    loadList()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('sysAnnouncementPage.delete.fail'))
    }
  }
}

function openDetail(row: AnnouncementItem) {
  current.value = row
  detailVisible.value = true
}

function getStatusType(status: number): string {
  const map: Record<number, string> = { 0: 'info', 1: 'success', 2: 'warning' }
  return map[status] || 'info'
}
</script>

<style scoped lang="scss">
.page {
  padding: 16px;
}

.table-card {
  .toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }
}

.title-text {
  font-weight: 500;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.flex-1 {
  flex: 1;
}

.content-box {
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow: auto;
  background: #f8fafc;
  padding: 12px;
  border-radius: 6px;
  font-size: 14px;
  line-height: 1.6;
}
</style>
